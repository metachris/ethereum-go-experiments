package ethtools

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jmoiron/sqlx"
)

type AddressStats struct {
	Address                      string
	AddressDetail                AddressDetail
	NumTxSent                    int
	NumTxReceived                int
	NumFailedTransfersSender     int
	NumFailedTransfersReceiver   int
	NumTxWithData                int
	NumTxTokenTransfer           int
	NumTxTokenMethodTransfer     int
	NumTxTokenMethodTransferFrom int
	ValueSentWei                 *big.Int
	ValueSentEth                 string
	ValueReceivedWei             *big.Int
	ValueReceivedEth             string
	TokensTransferred            *big.Int
}

func NewAddressStats(address string) *AddressStats {
	return &AddressStats{
		Address:           address,
		AddressDetail:     NewAddressDetail(address),
		ValueSentWei:      new(big.Int),
		ValueReceivedWei:  new(big.Int),
		TokensTransferred: new(big.Int),
	}
}

type TopAddressData struct {
	ValueReceived     []AddressStats
	ValueSent         []AddressStats
	NumTxReceived     []AddressStats
	NumTxSent         []AddressStats
	NumTokenTransfers []AddressStats
}

type AnalysisResult struct {
	StartBlockNumber    int64
	StartBlockTimestamp uint64
	EndBlockNumber      int64
	EndBlockTimestamp   uint64

	Addresses    map[string]*AddressStats `json:"-"`
	TopAddresses TopAddressData

	TxTypes       map[uint8]int
	ValueTotalWei *big.Int
	ValueTotalEth string

	NumBlocks                        int
	NumTransactions                  int
	NumTransactionsFailed            int
	NumTransactionsWithZeroValue     int
	NumTransactionsWithData          int
	NumTransactionsWithTokenTransfer int
}

func NewResult() *AnalysisResult {
	return &AnalysisResult{
		ValueTotalWei: new(big.Int),
		TxTypes:       make(map[uint8]int),
		Addresses:     make(map[string]*AddressStats),
	}
}

func (result *AnalysisResult) AddBlock(block *types.Block, client *ethclient.Client) {
	if result.StartBlockTimestamp == 0 {
		result.StartBlockTimestamp = block.Time()
	}

	result.EndBlockNumber = block.Number().Int64()
	result.EndBlockTimestamp = block.Time()

	result.NumBlocks += 1
	result.NumTransactions += len(block.Transactions())

	// Iterate over all transactions
	for _, tx := range block.Transactions() {
		result.AddTransaction(tx, client)
	}
}

func (result *AnalysisResult) AddTransaction(tx *types.Transaction, client *ethclient.Client) {
	// Count receiver stats
	var toAddrStats *AddressStats = &AddressStats{}   // might not exist
	var fromAddrStats *AddressStats = &AddressStats{} // might not exist - see EIP155Signer
	var isAddressKnown bool

	// Prepare address stats for receiver of tx
	if tx.To() != nil {
		recAddr := tx.To().String()
		toAddrStats, isAddressKnown = result.Addresses[recAddr]
		if !isAddressKnown {
			toAddrStats = NewAddressStats(recAddr)
			result.Addresses[recAddr] = toAddrStats
		}
	}

	// Prepare address stats for sender of tx, if exists
	if msg, err := tx.AsMessage(types.NewEIP155Signer(tx.ChainId())); err == nil {
		fromAddressString := msg.From().Hex()
		fromAddrStats, isAddressKnown = result.Addresses[fromAddressString]
		if !isAddressKnown {
			fromAddrStats = NewAddressStats(fromAddressString)
			result.Addresses[fromAddressString] = fromAddrStats
		}
	}

	// log.Fatal("from addr:", fromAddrStats.Address)

	if GetConfig().CheckTxStatus {
		// Check tx status
		receipt, err := client.TransactionReceipt(context.Background(), tx.Hash())
		Perror(err)
		if receipt.Status == 0 { // failed transaction. revert stats
			// DebugPrintln("- failed transaction, revert ", tx.Hash())
			result.NumTransactionsFailed += 1
			fromAddrStats.NumFailedTransfersSender += 1
			toAddrStats.NumFailedTransfersReceiver += 1
			return
		}
	}

	// Count total value
	result.ValueTotalWei = result.ValueTotalWei.Add(result.ValueTotalWei, tx.Value())

	// Count number of transactions without value
	if isBigIntZero(tx.Value()) {
		result.NumTransactionsWithZeroValue += 1
	}

	// Count tx types
	result.TxTypes[tx.Type()] += 1

	toAddrStats.NumTxReceived += 1
	toAddrStats.ValueReceivedWei = new(big.Int).Add(toAddrStats.ValueReceivedWei, tx.Value())
	toAddrStats.ValueReceivedEth = WeiToEth(toAddrStats.ValueReceivedWei).Text('f', 2)

	// Process FROM address (see https://goethereumbook.org/en/transaction-query/)
	if fromAddrStats != nil {
		fromAddrStats.NumTxSent += 1
		fromAddrStats.ValueSentWei = big.NewInt(0).Add(fromAddrStats.ValueSentWei, tx.Value())
		fromAddrStats.ValueSentEth = WeiToEth(fromAddrStats.ValueSentWei).Text('f', 2)
	}

	// Check for smart contract calls: token transfer
	data := tx.Data()
	if len(data) > 0 {
		result.NumTransactionsWithData += 1
		toAddrStats.NumTxWithData += 1

		if len(data) > 4 {
			methodId := hex.EncodeToString(data[:4])
			// fmt.Println(methodId)
			if methodId == "a9059cbb" || methodId == "23b872dd" {
				result.NumTransactionsWithTokenTransfer += 1
				toAddrStats.NumTxTokenTransfer += 1

				// Calculate and store the number of tokens transferred
				var value string
				switch methodId {
				case "a9059cbb": // transfer
					// targetAddr = hex.EncodeToString(data[4:36])
					value = hex.EncodeToString(data[36:68])
					toAddrStats.NumTxTokenMethodTransfer += 1
				case "23b872dd": // transferFrom
					// targetAddr = hex.EncodeToString(data[36:68])
					value = hex.EncodeToString(data[68:100])
					toAddrStats.NumTxTokenMethodTransferFrom += 1
				}
				valBigInt := new(big.Int)
				valBigInt.SetString(value, 16)

				// If number is too big, it is either an error or a erc-721 SC
				if len(valBigInt.String()) > 34 && toAddrStats.AddressDetail.Type == AddressTypeUnknown {
					DebugPrintf("warn: large value! tx: %s \t val: %s \t valBigInt: %v \n", tx.Hash(), value, valBigInt)

					toAddrStats.AddressDetail = GetAddressDetailFromBlockchain(toAddrStats.Address, client)
					fmt.Println("type:", toAddrStats.AddressDetail.Type)
				}

				// Increase number of tokens transferred if either ERC20 or Unknown Contract
				if toAddrStats.AddressDetail.Type == AddressTypeErc20 || toAddrStats.AddressDetail.Type == AddressTypeUnknown {
					toAddrStats.TokensTransferred = new(big.Int).Add(toAddrStats.TokensTransferred, valBigInt)
				}

				// errVal, _ := new(big.Int).SetString("1000000000000000000000000000", 10)
				// if toAddrInfo.Address == "0x0D8775F648430679A709E98d2b0Cb6250d2887EF" {
				// 	if valBigInt.Cmp(errVal) == 1 {
				// 		fmt.Printf("BAT %s \t %s \t %v \t %s \t %s \n", tx.Hash(), value, valBigInt, valBigInt.String(), toAddrInfo.TokensTransferred.String())
				// 	}
				// }
			}
		}
	}
}

// AnalysisResult.SortTopAddresses() sorts the addresses into TopAddresses after all blocks have been added
func (result *AnalysisResult) SortTopAddresses() {
	config := GetConfig()

	// Convert addresses from map into a sortable array
	_addresses := make([]AddressStats, 0, len(result.Addresses))
	for _, k := range result.Addresses {
		_addresses = append(_addresses, *k)
	}

	// token transfers
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].NumTxTokenTransfer > _addresses[j].NumTxTokenTransfer })
	for i := 0; i < len(_addresses) && i < config.NumAddressesByNumTokenTransfer; i++ {
		if _addresses[i].NumTxTokenTransfer > 0 {
			result.TopAddresses.NumTokenTransfers = append(result.TopAddresses.NumTokenTransfers, _addresses[i])
		}
	}

	// tx received
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].NumTxReceived > _addresses[j].NumTxReceived })
	for i := 0; i < len(_addresses) && i < config.NumAddressesByNumTxReceived; i++ {
		if _addresses[i].NumTxReceived > 0 {
			result.TopAddresses.NumTxReceived = append(result.TopAddresses.NumTxReceived, _addresses[i])
		}
	}

	// tx sent
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].NumTxSent > _addresses[j].NumTxSent })
	for i := 0; i < len(_addresses) && i < config.NumAddressesByNumTxSent; i++ {
		if _addresses[i].NumTxSent > 0 {
			result.TopAddresses.NumTxSent = append(result.TopAddresses.NumTxSent, _addresses[i])
		}
	}

	// value received
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].ValueReceivedWei.Cmp(_addresses[j].ValueReceivedWei) == 1 })
	for i := 0; i < len(_addresses) && i < config.NumAddressesByValueReceived; i++ {
		if _addresses[i].ValueReceivedWei.Cmp(big.NewInt(0)) == 1 { // if valueReceived > 0
			result.TopAddresses.ValueReceived = append(result.TopAddresses.ValueReceived, _addresses[i])
		}
	}

	// value sent
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].ValueSentWei.Cmp(_addresses[j].ValueSentWei) == 1 })
	for i := 0; i < len(_addresses) && i < config.NumAddressesByValueSent; i++ {
		if _addresses[i].ValueSentWei.Cmp(big.NewInt(0)) == 1 { // if valueSent > 0
			result.TopAddresses.ValueSent = append(result.TopAddresses.ValueSent, _addresses[i])
		}
	}
}

// Worker to add blocks to DB
func addBlockToDbWorker(db *sqlx.DB, jobChan <-chan *types.Block) {
	defer wg.Done()

	for block := range jobChan {
		AddBlockToDatabase(db, block)
	}
}

var wg sync.WaitGroup // for waiting until all blocks are written into DB

// Analyze blocks starting at specific block number, until a certain target timestamp
func AnalyzeBlocks(client *ethclient.Client, startBlockNumber int64, endTimestamp int64, db *sqlx.DB) *AnalysisResult {
	result := NewResult()
	result.StartBlockNumber = startBlockNumber

	currentBlockNumber := big.NewInt(startBlockNumber)

	// Setup database worker pool, for saving blocks to database
	numDbWorkers := 5
	blocksForDbQueue := make(chan *types.Block, 100)
	if db != nil {
		for w := 1; w <= numDbWorkers; w++ {
			wg.Add(1)
			go addBlockToDbWorker(db, blocksForDbQueue)
		}
	}

	timeStartBlockProcessing := time.Now()
	for {
		// fmt.Println(currentBlockNumber, "fetching...")
		currentBlock, err := client.BlockByNumber(context.Background(), currentBlockNumber)
		if err != nil {
			log.Fatal(err)
		}

		printBlock(currentBlock)

		if endTimestamp > -1 && currentBlock.Time() > uint64(endTimestamp) {
			fmt.Printf("- %d blocks processed. Skipped last block %s because it happened after endTime.\n\n", result.NumBlocks, currentBlockNumber.Text(10))
			break
		}

		if db != nil {
			blocksForDbQueue <- currentBlock
		}

		result.AddBlock(currentBlock, client)

		// Increase current block number and number of blocks processed
		currentBlockNumber.Add(currentBlockNumber, One)

		// negative endTimestamp is a helper to process only a certain number of blocks. If reached, then end now
		if endTimestamp < 0 && result.NumBlocks*-1 == int(endTimestamp) {
			break
		}
	}

	// Close DB-add-block channel and wait for all to be saved
	close(blocksForDbQueue)
	if db != nil {
		fmt.Println("Waiting for workers to finish adding blocks...")
	}
	wg.Wait()

	timeNeededBlockProcessing := time.Since(timeStartBlockProcessing)
	fmt.Printf("Reading blocks done (%.3fs). Sorting %d addresses...\n", timeNeededBlockProcessing.Seconds(), len(result.Addresses))
	timeStartSort := time.Now()
	result.SortTopAddresses()
	timeNeededSort := time.Since(timeStartSort)
	fmt.Printf("Sorting done (%.3fs)\n", timeNeededSort.Seconds())
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

// GetBlockAtTimestamp searches for a block that is within 30 seconds of the target timestamp.
// Guesses a block number, downloads it, and narrows down time gap if needed until block is found.
func GetBlockAtTimestamp(client *ethclient.Client, targetTimestamp int64) *types.Block {
	currentBlockNumber := getTargetBlocknumber(targetTimestamp)
	// TODO: check that blockNumber <= latestHeight
	var isNarrowingDownFromBelow = false

	divideSec := int64(13)                      // average block time is 13 sec
	lastSecDiffs := make(Int64RingBuffer, 0, 6) // if secDiffs keep repeating, need to adjust the divider

	fmt.Println("Finding start block:")
	for {
		// fmt.Println("Checking block:", currentBlockNumber)
		blockNumber := big.NewInt(currentBlockNumber)
		block, err := client.BlockByNumber(context.Background(), blockNumber)
		if err != nil {
			log.Println("Error fetching block at height", currentBlockNumber)
			log.Fatal(err)
		}

		secDiff := int64(block.Time()) - targetTimestamp

		if lastSecDiffs.Has(secDiff) {
			divideSec += 1
			// fmt.Println("already contains time diff. adjusting divider", divideSec)
		}
		lastSecDiffs.Add(secDiff)
		// fmt.Println(lastSecDiffs)

		fmt.Printf("%d \t blockTime: %d \t secDiff: %5d\n", currentBlockNumber, block.Time(), secDiff)
		if Abs(secDiff) < 60 {
			if secDiff < 0 {
				// still before wanted startTime. Increase by 1 from here...
				isNarrowingDownFromBelow = true
				currentBlockNumber += 1
				continue
			}

			// Only return if coming block-by-block from below, making sure to take first block after target time
			if isNarrowingDownFromBelow {
				return block
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
