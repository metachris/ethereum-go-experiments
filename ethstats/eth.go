package ethstats

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
	"github.com/metachris/ethereum-go-experiments/addressdata"
	"github.com/metachris/ethereum-go-experiments/core"
	"github.com/metachris/ethereum-go-experiments/database"
	"github.com/metachris/go-ethutils/utils"
)

type BlockWithTxReceipts struct {
	block      *types.Block
	txReceipts map[common.Hash]*types.Receipt
}

// GetBlockWithTxReceipts downloads a block and receipts for all transactions
func GetBlockWithTxReceipts(client *ethclient.Client, height int64) (res BlockWithTxReceipts) {
	var err error
	if client == nil {
		client, err = ethclient.Dial(core.Cfg.EthNode)
		utils.Perror(err)
	}
	res.block, err = client.BlockByNumber(context.Background(), big.NewInt(height))
	utils.Perror(err)

	res.txReceipts = make(map[common.Hash]*types.Receipt)
	if core.Cfg.LowApiCallMode {
		return res
	}

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

	client, err := ethclient.Dial(core.Cfg.EthNode)
	utils.Perror(err)

	db := database.NewStatsService(core.Cfg.Database)
	defer db.Close()

	for blockHeight := range blockHeightChan {
		res := GetBlockWithTxReceipts(client, blockHeight)
		db.AddBlock(res.block)
		blockChan <- &res
	}
}

// Analyze blocks starting at specific block number, until a certain target timestamp
func AnalyzeBlocks(client *ethclient.Client, startBlockNumber int64, endBlockNumber int64) *core.Analysis {
	ads := addressdata.NewAddressDetailService(client)
	analysis := core.NewAnalysis(core.Cfg, client, ads)
	analysis.Data.StartBlockNumber = startBlockNumber

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
			utils.PrintBlock(block.block)
			ProcessBlockWithReceipts(block, client, analysis)
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
	fmt.Printf("Reading blocks done (%.3fs). Sorting %d addresses and checking address information...\n", timeNeededBlockProcessing.Seconds(), len(analysis.Addresses))

	// Sort now
	timeStartSort := time.Now()
	analysis.BuildTopAddresses()
	timeNeededSort := time.Since(timeStartSort)
	fmt.Printf("Sorting & checking addresses done (%.3fs)\n", timeNeededSort.Seconds())

	// Update address details for top transactions
	// result.EnsureTopTransactionAddressDetails(client)

	return analysis
}
