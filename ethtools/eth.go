package ethtools

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jmoiron/sqlx"
)

// GetBlockWithTxReceipts downloads a block and receipts for all transactions
func GetBlockWithTxReceipts(client *ethclient.Client, height int64) (res BlockWithTxReceipts) {
	var err error
	if client == nil {
		client, err = ethclient.Dial(GetConfig().EthNode)
		Perror(err)
	}
	res.block, err = client.BlockByNumber(context.Background(), big.NewInt(height))
	Perror(err)

	res.txReceipts = make(map[common.Hash]*types.Receipt)
	for _, tx := range res.block.Transactions() {
		receipt, err := client.TransactionReceipt(context.Background(), tx.Hash())
		if err != nil {
			if errors.Is(err, ethereum.NotFound) {
				// can apparently happen if 0 tx: https://etherscan.io/block/10102170
				// fmt.Println("aaa", tx.Hash())
				continue
			}
			panic(err)

		}
		res.txReceipts[tx.Hash()] = receipt
	}

	return res
}

// GetBlockWithTxReceiptsWorker creates a geth connection, listens for blockHeights and fetches block with all tx receipts
func GetBlockWithTxReceiptsWorker(wg *sync.WaitGroup, blockHeightChan <-chan int64, blockChan chan<- *BlockWithTxReceipts) {
	defer wg.Done()

	client, err := ethclient.Dial(GetConfig().EthNode)
	Perror(err)

	for blockHeight := range blockHeightChan {
		res := GetBlockWithTxReceipts(client, blockHeight)
		blockChan <- &res
	}
}

// Analyze blocks starting at specific block number, until a certain target timestamp
func AnalyzeBlocks(client *ethclient.Client, db *sqlx.DB, startBlockNumber int64, endBlockNumber int64) *AnalysisResult {
	result := NewResult()
	result.StartBlockNumber = startBlockNumber

	blockHeightChan := make(chan int64, 100)          // blockHeight to fetch with receipts
	blockChan := make(chan *BlockWithTxReceipts, 100) // channel for resulting BlockWithTxReceipt

	// start block fetcher pool
	var blockWorkerWg sync.WaitGroup
	for w := 1; w <= 10; w++ {
		blockWorkerWg.Add(1)
		go GetBlockWithTxReceiptsWorker(&blockWorkerWg, blockHeightChan, blockChan)
	}

	// Start block processor
	var analyzeLock sync.Mutex
	analyzeWorker := func(blockChan <-chan *BlockWithTxReceipts) {
		defer analyzeLock.Unlock() // we unlock when done

		for block := range blockChan {
			printBlock(block.block)
			result.AddBlockWithReceipts(block, client)
		}
	}

	// Set the analyzer lock, to wait until it's done processing all blocks
	analyzeLock.Lock()
	go analyzeWorker(blockChan)

	// Add block heights to worker queue
	timeStartBlockProcessing := time.Now()
	for currentBlockNumber := startBlockNumber; currentBlockNumber <= endBlockNumber; currentBlockNumber++ {
		blockHeightChan <- currentBlockNumber
	}

	fmt.Println("Waiting for block workers...")
	close(blockHeightChan)
	blockWorkerWg.Wait()

	fmt.Println("Waiting for Analysis workers...")
	close(blockChan)
	analyzeLock.Lock()

	timeNeededBlockProcessing := time.Since(timeStartBlockProcessing)
	fmt.Printf("Reading blocks done (%.3fs). Sorting %d addresses and checking address information...\n", timeNeededBlockProcessing.Seconds(), len(result.Addresses))

	// Sort now
	timeStartSort := time.Now()
	result.SortTopAddresses(client)
	timeNeededSort := time.Since(timeStartSort)
	fmt.Printf("Sorting & checking addresses done (%.3fs)\n", timeNeededSort.Seconds())

	// Update address details for top transactions
	// result.UpdateTxAddressDetails(client)

	return result
}

// GetBlockHeaderAtTimestamp returns the header of the first block at or after the timestamp. If timestamp is after
// latest block, then return latest block.
func GetBlockHeaderAtTimestamp(client *ethclient.Client, targetTimestamp int64, verbose bool) (header *types.Header, err error) {
	// Get latest header
	latestBlockHeader, err := client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		return header, err
	}

	// Ensure target timestamp is before latest block
	if uint64(targetTimestamp) > latestBlockHeader.Time {
		return header, errors.New("target timestamp after latest block")
	}

	// Estimate a target block number
	currentBlockNumber := estimateTargetBlocknumber(targetTimestamp)

	// If estimation later than latest block, then use latest block as estimation base
	if currentBlockNumber > latestBlockHeader.Number.Int64() {
		currentBlockNumber = latestBlockHeader.Number.Int64()
	}

	// approach the target block from below, to be sure it's the first one at/after the timestamp
	var isNarrowingDownFromBelow = false

	// Ringbuffer for the latest secDiffs, to avoid going in circles when narrowing down
	lastSecDiffs := make([]int64, 7)
	lastSecDiffsIncludes := func(a int64) bool {
		for _, b := range lastSecDiffs {
			if b == a {
				return true
			}
		}
		return false
	}

	DebugPrintf("Finding start block:\n")
	var secDiff int64
	blockSecAvg := int64(13) // average block time. is adjusted when narrowing down

	for {
		// DebugPrintln("Checking block:", currentBlockNumber)
		blockNumber := big.NewInt(currentBlockNumber)
		header, err := client.HeaderByNumber(context.Background(), blockNumber)
		if err != nil {
			return header, err
		}

		secDiff = int64(header.Time) - targetTimestamp

		DebugPrintf("%d \t blockTime: %d / %v \t secDiff: %5d\n", currentBlockNumber, header.Time, time.Unix(int64(header.Time), 0).UTC(), secDiff)

		// Check if this secDiff was already seen (avoid circular endless loop)
		if lastSecDiffsIncludes(secDiff) && blockSecAvg < 25 {
			blockSecAvg += 1
			DebugPrintln("- Increase blockSecAvg to", blockSecAvg)
		}

		// Pop & add secDiff to array of last values
		lastSecDiffs = lastSecDiffs[1:]
		lastSecDiffs = append(lastSecDiffs, secDiff)
		// DebugPrintln("lastSecDiffs:", lastSecDiffs)

		if Abs(secDiff) < 80 || isNarrowingDownFromBelow { // getting close
			if secDiff < 0 {
				// still before wanted startTime. Increase by 1 from here...
				isNarrowingDownFromBelow = true
				currentBlockNumber += 1
				continue
			}

			// Only return if coming block-by-block from below, making sure to take first block after target time
			if isNarrowingDownFromBelow {
				return header, nil
			} else {
				currentBlockNumber -= 1
				continue
			}
		}

		// Try for better block in big steps
		blockDiff := secDiff / blockSecAvg
		currentBlockNumber -= blockDiff
	}
}
