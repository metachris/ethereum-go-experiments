package ethtools

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jmoiron/sqlx"
)

func GetBlockWithTxReceipts(client *ethclient.Client, height int64) (res BlockWithTxReceipts) {
	// fmt.Println("getBlockWithTxReceipts", height)

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
		Perror(err)
		res.txReceipts[tx.Hash()] = receipt
	}

	// fmt.Println(height, len(res.txReceipts))
	return res
}

func GetBlockWithTxReceiptsWorker(wg *sync.WaitGroup, blockHeightChan <-chan int64, blockChan chan<- *BlockWithTxReceipts) {
	defer wg.Done()

	client, err := ethclient.Dial(GetConfig().EthNode)
	Perror(err)

	for blockHeight := range blockHeightChan {
		res := GetBlockWithTxReceipts(client, blockHeight)
		blockChan <- &res
	}
}

// // Worker to add blocks to DB
// func addBlockToDbWorker(db *sqlx.DB, jobChan <-chan *types.Block) {
// 	defer wg.Done()

// 	for block := range jobChan {
// 		AddBlockToDatabase(db, block)
// 	}
// }

// var wg sync.WaitGroup // for waiting until all blocks are written into DB

// Analyze blocks starting at specific block number, until a certain target timestamp
func AnalyzeBlocks(client *ethclient.Client, db *sqlx.DB, startBlockNumber int64, endBlockNumber int64) *AnalysisResult {
	result := NewResult()
	result.StartBlockNumber = startBlockNumber

	// Setup database worker pool, for saving blocks to database
	// numDbWorkers := 5
	// blocksForDbQueue := make(chan *types.Block, 100)
	// if db != nil {
	// 	for w := 1; w <= numDbWorkers; w++ {
	// 		wg.Add(1)
	// 		go addBlockToDbWorker(db, blocksForDbQueue)
	// 	}
	// }

	blockHeightChan := make(chan int64, 100)          // blockHeight to fetch with receipts
	blockChan := make(chan *BlockWithTxReceipts, 100) // channel for resulting BlockWithTxReceipt

	// start block fetcher pool
	var blockWorkerWg sync.WaitGroup
	for w := 1; w <= 5; w++ {
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

	analyzeLock.Lock()
	go analyzeWorker(blockChan)

	timeStartBlockProcessing := time.Now()
	for currentBlockNumber := startBlockNumber; currentBlockNumber <= endBlockNumber; currentBlockNumber++ {
		blockHeightChan <- currentBlockNumber
	}

	fmt.Println("Waiting for block workers...")
	close(blockHeightChan)
	blockWorkerWg.Wait()

	fmt.Println("Waiting for Analysis workers...")
	close(blockChan)
	analyzeLock.Lock() // try to get lock again

	// Close DB-add-block channel and wait for all to be saved
	// close(blocksForDbQueue)
	// if db != nil {
	// 	fmt.Println("Waiting for workers to finish adding blocks...")
	// }
	// wg.Wait()

	timeNeededBlockProcessing := time.Since(timeStartBlockProcessing)
	fmt.Printf("Reading blocks done (%.3fs). Sorting %d addresses...\n", timeNeededBlockProcessing.Seconds(), len(result.Addresses))

	// Sort now
	timeStartSort := time.Now()
	result.SortTopAddresses(client)
	timeNeededSort := time.Since(timeStartSort)
	fmt.Printf("Sorting done (%.3fs)\n", timeNeededSort.Seconds())

	result.UpdateTxAddressDetails(client)

	// // Ensure top addresses have details from blockchain
	// timeStartUpdateAddresses := time.Now()
	// result.EnsureAddressDetails(client)
	// timeNeededUpdateAddresses := time.Since(timeStartUpdateAddresses)
	// fmt.Printf("Updating addresses done (in %.3fs)\n", timeNeededUpdateAddresses.Seconds())
	return result
}

// Functionality for getting the first block after a certain timestamp
type Int64RingBuffer []int64

func (list Int64RingBuffer) Has(a int64) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func (list *Int64RingBuffer) Add(a int64) {
	// fmt.Println("len", len(*list), "cap", cap(*list))
	// if len(*list) == cap(*list) {
	if len(*list) == 6 {
		*list = (*list)[1:]
	}
	*list = append(*list, a)
}

// GetBlockHeaderAtTimestamp returns the header of the first block at or after the timestamp.
func GetBlockHeaderAtTimestamp(client *ethclient.Client, targetTimestamp int64, verbose bool) (header *types.Header, secDiff int64) {
	latestBlockHeader, err := client.HeaderByNumber(context.Background(), nil)
	Perror(err)

	currentBlockNumber := estimateTargetBlocknumber(targetTimestamp)
	if currentBlockNumber > latestBlockHeader.Number.Int64() {
		currentBlockNumber = latestBlockHeader.Number.Int64()
	}

	// TODO: check that blockNumber <= latestHeight
	var isNarrowingDownFromBelow = false

	divideSec := int64(13)                      // average block time is 13 sec
	lastSecDiffs := make(Int64RingBuffer, 0, 6) // if secDiffs keep repeating, need to adjust the divider

	var verbosePrintf = func(format string, a ...interface{}) {
		if verbose {
			fmt.Printf(format, a...)
		}
	}

	verbosePrintf("Finding start block:\n")
	for {
		// fmt.Println("Checking block:", currentBlockNumber)
		blockNumber := big.NewInt(currentBlockNumber)
		header, err := client.HeaderByNumber(context.Background(), blockNumber)
		Perror(err)

		secDiff = int64(header.Time) - targetTimestamp

		// Check if this secDiff was already seen (avoid circular endless loop)
		if lastSecDiffs.Has(secDiff) {
			divideSec += 1
		}

		lastSecDiffs.Add(secDiff)

		verbosePrintf("%d \t blockTime: %d / %v \t secDiff: %5d\n", currentBlockNumber, header.Time, time.Unix(int64(header.Time), 0).UTC(), secDiff)
		if Abs(secDiff) < 60 {
			if secDiff < 0 {
				// still before wanted startTime. Increase by 1 from here...
				isNarrowingDownFromBelow = true
				currentBlockNumber += 1
				continue
			}

			// Only return if coming block-by-block from below, making sure to take first block after target time
			if isNarrowingDownFromBelow {
				return header, secDiff
			} else {
				currentBlockNumber -= 1
				continue
			}
		}

		// Try for better block in big steps
		blockDiff := secDiff / divideSec
		currentBlockNumber -= blockDiff
	}
}
