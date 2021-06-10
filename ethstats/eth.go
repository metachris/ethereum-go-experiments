package ethstats

import (
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/metachris/ethereum-go-experiments/addressdata"
	"github.com/metachris/ethereum-go-experiments/core"
	"github.com/metachris/go-ethutils/blockswithtx"
	"github.com/metachris/go-ethutils/utils"
)

// Analyze blocks starting at specific block number, until a certain target timestamp
func AnalyzeBlocks(client *ethclient.Client, startHeight int64, endHeight int64) *core.Analysis {
	ads := addressdata.NewAddressDetailService(client)
	analysis := core.NewAnalysis(core.Cfg, client, ads)
	analysis.Data.StartBlockNumber = startHeight

	blockChan := make(chan *blockswithtx.BlockWithTxReceipts, 100) // channel for resulting BlockWithTxReceipt

	// Start block processor
	var analyzeLock sync.Mutex
	go func() {
		analyzeLock.Lock()
		defer analyzeLock.Unlock() // we unlock when done

		for block := range blockChan {
			utils.PrintBlock(block.Block)
			ProcessBlockWithReceipts(block, client, analysis)
		}
	}()

	// Start timer
	timeStartBlockProcessing := time.Now()

	// Start fetching and processing blocks
	blockswithtx.GetBlocksWithTxReceipts(client, blockChan, startHeight, endHeight, 5)

	// Wait for processing to finish
	fmt.Println("Waiting for Analysis workers...")
	close(blockChan)
	analyzeLock.Lock() // wait until all blocks have been processed

	// End timer
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
