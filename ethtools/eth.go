package ethtools

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jmoiron/sqlx"
)

type BlockWithTxReceipts struct {
	block      *types.Block
	txReceipts map[common.Hash]*types.Receipt
}

type AddressStats struct {
	Address       string
	AddressDetail AddressDetail

	NumTxSentSuccess     int
	NumTxSentFailed      int
	NumTxReceivedSuccess int
	NumTxReceivedFailed  int

	NumTxMevSent          int
	NumTxMevReceived      int
	NumTxWithDataSent     int
	NumTxWithDataReceived int

	NumTxErc20Sent      int
	NumTxErc20Received  int
	NumTxErc721Sent     int
	NumTxErc721Received int

	ValueSentWei     *big.Int
	ValueReceivedWei *big.Int

	Erc20TokensSent     *big.Int
	Erc20TokensReceived *big.Int

	GasUsed        *big.Int
	GasFeeTotal    *big.Int
	GasFeeFailedTx *big.Int
}

// // Returns the token amount in the correct unit. Eg. USDC has 6 decimals. 12345678 -> 12.345678
// // Updates from blockchain if not already has
// func (stats *AddressStats) TokensTransferredInUnit(client *ethclient.Client) (amount *big.Float, symbol string) {
// 	tokensTransferredFloat := new(big.Float).SetInt(stats.TokensTransferred)

// 	stats.EnsureAddressDetails(client)

// 	if stats.AddressDetail.IsErc20() {
// 		decimals := uint64(stats.AddressDetail.Decimals)
// 		divider := float64(1)
// 		if decimals > 0 {
// 			divider = float64(math.Pow(10, float64(decimals)))
// 		}

// 		amount := new(big.Float).Quo(tokensTransferredFloat, big.NewFloat(divider))
// 		return amount, stats.AddressDetail.Symbol
// 	} else {
// 		return tokensTransferredFloat, ""
// 	}
// }

func (stats *AddressStats) EnsureAddressDetails(client *ethclient.Client) {
	if !stats.AddressDetail.IsLoaded() && client != nil {
		stats.AddressDetail, _ = GetAddressDetail(stats.Address, client)
	}
}

func NewAddressStats(address string) *AddressStats {
	return &AddressStats{
		Address:             address,
		AddressDetail:       NewAddressDetail(address),
		ValueSentWei:        new(big.Int),
		ValueReceivedWei:    new(big.Int),
		Erc20TokensSent:     new(big.Int),
		Erc20TokensReceived: new(big.Int),
		GasUsed:             new(big.Int),
		GasFeeTotal:         new(big.Int),
		GasFeeFailedTx:      new(big.Int),
	}
}

type TopAddressData struct {
	ValueSent     []AddressStats
	ValueReceived []AddressStats

	NumTxSentSuccess     []AddressStats
	NumTxSentFailed      []AddressStats
	NumTxReceivedSuccess []AddressStats
	NumTxReceivedFailed  []AddressStats

	NumTxErc20Sent      []AddressStats
	NumTxErc20Received  []AddressStats
	NumTxErc721Sent     []AddressStats
	NumTxErc721Received []AddressStats

	GasFeeTotal    []AddressStats
	GasFeeFailedTx []AddressStats
}

type TxStats struct {
	Hash     string
	FromAddr AddressDetail
	ToAddr   AddressDetail
	GasUsed  *big.Int
	GasFee   *big.Int
	Value    *big.Int
	DataSize int
	Success  bool
}

func (stats TxStats) String() string {
	failedMsg := "         "
	if !stats.Success {
		failedMsg = " (failed)"
	}
	return fmt.Sprintf("%s%s \t gasfee: %-8s \t val: %14v \t datasize: %-8d \t\t %s %-20s -> %s %s", stats.Hash, failedMsg, WeiBigIntToEthString(stats.GasFee, 4), WeiBigIntToEthString(stats.Value, 4), stats.DataSize, stats.FromAddr.Address, stats.FromAddr.Name, stats.ToAddr.Address, stats.ToAddr.Name)
}

type TopTransactionData struct {
	GasFee   []TxStats
	Value    []TxStats
	DataSize []TxStats
}

type AnalysisResult struct {
	StartBlockNumber    int64
	StartBlockTimestamp uint64
	EndBlockNumber      int64
	EndBlockTimestamp   uint64

	Addresses       map[string]*AddressStats `json:"-"`
	TopAddresses    TopAddressData
	TopTransactions TopTransactionData

	TxTypes       map[uint8]int
	ValueTotalWei *big.Int
	ValueTotalEth string

	NumBlocks          int
	NumBlocksWithoutTx int
	GasUsed            *big.Int
	GasFeeTotal        *big.Int

	NumTransactions              int
	NumTransactionsFailed        int
	NumTransactionsWithZeroValue int
	NumTransactionsWithData      int

	NumTransactionsErc20Transfer  int
	NumTransactionsErc721Transfer int

	NumMevTransactionsSuccess int
	NumMevTransactionsFailed  int
}

func NewResult() *AnalysisResult {
	return &AnalysisResult{
		ValueTotalWei: new(big.Int),
		TxTypes:       make(map[uint8]int),
		Addresses:     make(map[string]*AddressStats),
		GasUsed:       new(big.Int),
		GasFeeTotal:   new(big.Int),

		TopTransactions: TopTransactionData{
			GasFee:   make([]TxStats, 0, GetConfig().NumTopTransactions),
			Value:    make([]TxStats, 0, GetConfig().NumTopTransactions),
			DataSize: make([]TxStats, 0, GetConfig().NumTopTransactions),
		},
	}
}

func (result *AnalysisResult) AddTxToTopList(tx *types.Transaction, receipt *types.Receipt) {
	txGasUsed := big.NewInt(int64(receipt.GasUsed))
	txGasFee := new(big.Int).Mul(txGasUsed, tx.GasPrice())

	to := NewAddressDetail("")
	if tx.To() != nil {
		to = NewAddressDetail(tx.To().String())
	}

	from := NewAddressDetail("")
	if msg, err := tx.AsMessage(types.NewEIP155Signer(tx.ChainId())); err == nil {
		from = NewAddressDetail(msg.From().Hex())
	}

	stats := TxStats{
		Hash:     tx.Hash().Hex(),
		GasFee:   txGasFee,
		GasUsed:  txGasUsed,
		Value:    tx.Value(),
		DataSize: len(tx.Data()),
		Success:  receipt.Status == 1,
		FromAddr: from,
		ToAddr:   to,
	}

	// Sort by Gas fee
	if len(result.TopTransactions.GasFee) < cap(result.TopTransactions.GasFee) { // add new item to array
		result.TopTransactions.GasFee = append(result.TopTransactions.GasFee, stats)
	} else if stats.GasFee.Cmp(result.TopTransactions.GasFee[len(result.TopTransactions.GasFee)-1].GasFee) == 1 { // replace last item
		result.TopTransactions.GasFee[len(result.TopTransactions.GasFee)-1] = stats
	}
	sort.SliceStable(result.TopTransactions.GasFee, func(i, j int) bool {
		return result.TopTransactions.GasFee[i].GasFee.Cmp(result.TopTransactions.GasFee[j].GasFee) == 1
	})

	// Sort by value
	if len(result.TopTransactions.Value) < cap(result.TopTransactions.Value) { // add new item to array
		result.TopTransactions.Value = append(result.TopTransactions.Value, stats)
	} else if stats.Value.Cmp(result.TopTransactions.Value[len(result.TopTransactions.Value)-1].Value) == 1 {
		result.TopTransactions.Value[len(result.TopTransactions.Value)-1] = stats
	}
	sort.SliceStable(result.TopTransactions.Value, func(i, j int) bool {
		return result.TopTransactions.Value[i].Value.Cmp(result.TopTransactions.Value[j].Value) == 1
	})

	// Sort by len(data)
	if len(result.TopTransactions.DataSize) < cap(result.TopTransactions.DataSize) { // add new item to array
		result.TopTransactions.DataSize = append(result.TopTransactions.DataSize, stats)
	} else if stats.DataSize > result.TopTransactions.DataSize[len(result.TopTransactions.DataSize)-1].DataSize {
		result.TopTransactions.DataSize[len(result.TopTransactions.DataSize)-1] = stats
	}
	sort.SliceStable(result.TopTransactions.DataSize, func(i, j int) bool {
		return result.TopTransactions.DataSize[i].DataSize > result.TopTransactions.DataSize[j].DataSize
	})
}

func (result *AnalysisResult) AddBlockWithReceipts(block *BlockWithTxReceipts, client *ethclient.Client) {
	if result.StartBlockTimestamp == 0 {
		result.StartBlockTimestamp = block.block.Time()
	}

	result.EndBlockNumber = block.block.Number().Int64()
	result.EndBlockTimestamp = block.block.Time()

	result.NumBlocks += 1
	result.NumTransactions += len(block.block.Transactions())
	result.GasUsed = new(big.Int).Add(result.GasUsed, big.NewInt(int64(block.block.GasUsed())))

	// Iterate over all transactions
	for _, tx := range block.block.Transactions() {
		receipt := block.txReceipts[tx.Hash()]
		result.AddTransaction(client, tx, receipt)
	}

	// If no transactions in this block then record that
	if len(block.block.Transactions()) == 0 {
		result.NumBlocksWithoutTx += 1
	}
}

func (result *AnalysisResult) AddTransaction(client *ethclient.Client, tx *types.Transaction, receipt *types.Receipt) {
	// Count receiver stats
	var toAddrStats *AddressStats = NewAddressStats("")   // might not exist
	var fromAddrStats *AddressStats = NewAddressStats("") // might not exist - see EIP155Signer
	var isAddressKnown bool

	result.AddTxToTopList(tx, receipt)

	// Prepare address stats for receiver
	if tx.To() != nil {
		recAddr := tx.To().String()
		toAddrStats, isAddressKnown = result.Addresses[recAddr]
		if !isAddressKnown {
			toAddrStats = NewAddressStats(recAddr)
			result.Addresses[recAddr] = toAddrStats
		}
	}

	// Prepare address stats for sender
	if msg, err := tx.AsMessage(types.NewEIP155Signer(tx.ChainId())); err == nil {
		fromAddressString := msg.From().Hex()
		fromAddrStats, isAddressKnown = result.Addresses[fromAddressString]
		if !isAddressKnown {
			fromAddrStats = NewAddressStats(fromAddressString)
			result.Addresses[fromAddressString] = fromAddrStats
		}
	}

	// Gas used is paid and counted no matter if transaction failed or succeeded
	txGasUsed := big.NewInt(int64(receipt.GasUsed))
	txGasFee := new(big.Int).Mul(txGasUsed, tx.GasPrice())
	fromAddrStats.GasUsed = new(big.Int).Add(fromAddrStats.GasUsed, txGasUsed)
	fromAddrStats.GasFeeTotal = new(big.Int).Add(fromAddrStats.GasFeeTotal, txGasFee)
	result.GasFeeTotal = new(big.Int).Add(result.GasFeeTotal, txGasFee)

	txSuccessful := receipt.Status == 1
	if !txSuccessful {
		result.NumTransactionsFailed += 1
		fromAddrStats.NumTxSentFailed += 1
		toAddrStats.NumTxReceivedFailed += 1
		fromAddrStats.GasFeeFailedTx = new(big.Int).Add(fromAddrStats.GasFeeFailedTx, txGasFee)

		// Count failed MEV tx
		if len(tx.Data()) > 0 && tx.GasPrice().Uint64() == 0 {
			result.NumMevTransactionsFailed += 1
			if GetConfig().DebugPrintMevTx {
				fmt.Printf("MEV fail tx: https://etherscan.io/tx/%s\n", tx.Hash())
			}
		}

		return
	}

	// TX was successful...
	result.ValueTotalWei = result.ValueTotalWei.Add(result.ValueTotalWei, tx.Value())
	result.TxTypes[tx.Type()] += 1

	fromAddrStats.NumTxSentSuccess += 1
	fromAddrStats.ValueSentWei = new(big.Int).Add(fromAddrStats.ValueSentWei, tx.Value())

	toAddrStats.NumTxReceivedSuccess += 1
	toAddrStats.ValueReceivedWei = new(big.Int).Add(toAddrStats.ValueReceivedWei, tx.Value())

	if isBigIntZero(tx.Value()) {
		result.NumTransactionsWithZeroValue += 1
	}

	// Check for smart contract calls: token transfer
	data := tx.Data()
	if len(data) > 0 {
		result.NumTransactionsWithData += 1
		fromAddrStats.NumTxWithDataSent += 1
		toAddrStats.NumTxWithDataReceived += 1

		// If no gas price, then it's probably a MEV/Flashbots tx
		if tx.GasPrice().Uint64() == 0 {
			result.NumMevTransactionsSuccess += 1
			fromAddrStats.NumTxMevSent += 1
			toAddrStats.NumTxMevReceived += 1

			if GetConfig().DebugPrintMevTx {
				fmt.Printf("MEV ok tx: https://etherscan.io/tx/%s\n", tx.Hash())
			}
		}

		if len(data) > 4 {
			methodId := hex.EncodeToString(data[:4])
			// fmt.Println(methodId)
			if methodId == "a9059cbb" || methodId == "23b872dd" {
				fromAddrStats.EnsureAddressDetails(client)
				toAddrStats.EnsureAddressDetails(client)

				// Calculate and store the number of tokens transferred
				var value string
				switch methodId {
				case "a9059cbb": // transfer
					// targetAddr = hex.EncodeToString(data[4:36])
					value = hex.EncodeToString(data[36:68])
				case "23b872dd": // transferFrom
					// targetAddr = hex.EncodeToString(data[36:68])
					value = hex.EncodeToString(data[68:100])
				}

				valBigInt := new(big.Int)
				valBigInt.SetString(value, 16)

				if toAddrStats.AddressDetail.IsErc20() {
					result.NumTransactionsErc20Transfer += 1

					fromAddrStats.NumTxErc20Sent += 1
					fromAddrStats.Erc20TokensSent = new(big.Int).Add(fromAddrStats.Erc20TokensSent, valBigInt)

					toAddrStats.NumTxErc20Received += 1
					toAddrStats.Erc20TokensReceived = new(big.Int).Add(toAddrStats.Erc20TokensReceived, valBigInt)
				} else if toAddrStats.AddressDetail.IsErc721() {
					result.NumTransactionsErc721Transfer += 1
					fromAddrStats.NumTxErc721Sent += 1
					toAddrStats.NumTxErc721Received += 1
				}

				// If number is too big, it is either an error or a erc-721 SC
				// if len(valBigInt.String()) > 34 && toAddrStats.AddressDetail.Type == AddressTypeInit {
				// 	DebugPrintf("warn: large value! tx: %s \t val: %s \t valBigInt: %v \n", tx.Hash(), value, valBigInt)

				// 	toAddrStats.EnsureAddressDetails(client)
				// 	// fmt.Println("type:", toAddrStats.AddressDetail.Type)
				// }

				// Increase number of tokens transferred if either ERC20 or Unknown Contract
				// if toAddrStats.AddressDetail.Type == AddressTypeErc20 || toAddrStats.AddressDetail.Type == AddressTypeInit {
				// toAddrStats.TokensTransferred = new(big.Int).Add(toAddrStats.TokensTransferred, valBigInt)
				// }
			}
		}
	}
}

// AnalysisResult.SortTopAddresses() sorts the addresses into TopAddresses after all blocks have been added.
// Ensures that details for all top addresses are queried from blockchain
func (result *AnalysisResult) SortTopAddresses(client *ethclient.Client) {
	config := GetConfig()

	// Convert addresses from map into a sortable array
	_addresses := make([]AddressStats, 0, len(result.Addresses))
	for _, k := range result.Addresses {
		_addresses = append(_addresses, *k)
	}

	// erc20-tx sent
	sort.SliceStable(_addresses, func(i, j int) bool {
		return _addresses[i].NumTxErc20Sent > _addresses[j].NumTxErc20Sent
	})
	for i := 0; i < len(_addresses) && i < config.NumTopAddressesLarge; i++ {
		if _addresses[i].NumTxErc20Sent > 0 {
			_addresses[i].EnsureAddressDetails(client)
			result.TopAddresses.NumTxErc20Sent = append(result.TopAddresses.NumTxErc20Sent, _addresses[i])
		}
	}

	// erc20-tx received
	sort.SliceStable(_addresses, func(i, j int) bool {
		return _addresses[i].NumTxErc20Received > _addresses[j].NumTxErc20Received
	})
	for i := 0; i < len(_addresses) && i < config.NumTopAddressesLarge; i++ {
		if _addresses[i].NumTxErc20Received > 0 {
			_addresses[i].EnsureAddressDetails(client)
			result.TopAddresses.NumTxErc20Received = append(result.TopAddresses.NumTxErc20Received, _addresses[i])
		}
	}

	// tx sent:success
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].NumTxSentSuccess > _addresses[j].NumTxSentSuccess })
	for i := 0; i < len(_addresses) && i < config.NumTopAddresses; i++ {
		if _addresses[i].NumTxSentSuccess > 0 {
			_addresses[i].EnsureAddressDetails(client)
			result.TopAddresses.NumTxSentSuccess = append(result.TopAddresses.NumTxSentSuccess, _addresses[i])
		}
	}

	// tx sent:fail
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].NumTxSentFailed > _addresses[j].NumTxSentFailed })
	for i := 0; i < len(_addresses) && i < config.NumTopAddresses; i++ {
		if _addresses[i].NumTxSentFailed > 0 {
			_addresses[i].EnsureAddressDetails(client)
			result.TopAddresses.NumTxSentFailed = append(result.TopAddresses.NumTxSentFailed, _addresses[i])
		}
	}

	// tx received:success
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].NumTxReceivedSuccess > _addresses[j].NumTxReceivedSuccess })
	for i := 0; i < len(_addresses) && i < config.NumTopAddresses; i++ {
		if _addresses[i].NumTxReceivedSuccess > 0 {
			_addresses[i].EnsureAddressDetails(client)
			result.TopAddresses.NumTxReceivedSuccess = append(result.TopAddresses.NumTxReceivedSuccess, _addresses[i])
		}
	}

	// tx received:fail
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].NumTxSentFailed > _addresses[j].NumTxSentFailed })
	for i := 0; i < len(_addresses) && i < config.NumTopAddresses; i++ {
		if _addresses[i].NumTxSentFailed > 0 {
			_addresses[i].EnsureAddressDetails(client)
			result.TopAddresses.NumTxSentFailed = append(result.TopAddresses.NumTxSentFailed, _addresses[i])
		}
	}

	// erc721 sent
	sort.SliceStable(_addresses, func(i, j int) bool {
		return _addresses[i].NumTxErc721Sent > _addresses[j].NumTxErc721Sent
	})
	for i := 0; i < len(_addresses) && i < config.NumTopAddresses; i++ {
		if _addresses[i].NumTxErc721Sent > 0 {
			_addresses[i].EnsureAddressDetails(client)
			result.TopAddresses.NumTxErc721Sent = append(result.TopAddresses.NumTxErc721Sent, _addresses[i])
		}
	}

	// erc721 received
	sort.SliceStable(_addresses, func(i, j int) bool {
		return _addresses[i].NumTxErc721Received > _addresses[j].NumTxErc721Received
	})
	for i := 0; i < len(_addresses) && i < config.NumTopAddresses; i++ {
		if _addresses[i].NumTxErc721Received > 0 {
			_addresses[i].EnsureAddressDetails(client)
			result.TopAddresses.NumTxErc721Received = append(result.TopAddresses.NumTxErc721Sent, _addresses[i])
		}
	}

	// value received
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].ValueReceivedWei.Cmp(_addresses[j].ValueReceivedWei) == 1 })
	for i := 0; i < len(_addresses) && i < config.NumTopAddresses; i++ {
		if _addresses[i].ValueReceivedWei.Cmp(big.NewInt(0)) == 1 { // if valueReceived > 0
			_addresses[i].EnsureAddressDetails(client)
			result.TopAddresses.ValueReceived = append(result.TopAddresses.ValueReceived, _addresses[i])
		}
	}

	// value sent
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].ValueSentWei.Cmp(_addresses[j].ValueSentWei) == 1 })
	for i := 0; i < len(_addresses) && i < config.NumTopAddresses; i++ {
		if _addresses[i].ValueSentWei.Cmp(big.NewInt(0)) == 1 { // if valueSent > 0
			_addresses[i].EnsureAddressDetails(client)
			result.TopAddresses.ValueSent = append(result.TopAddresses.ValueSent, _addresses[i])
		}
	}
}

func (result *AnalysisResult) UpdateTxAddressDetails(client *ethclient.Client) {
	for i, v := range result.TopTransactions.Value {
		result.TopTransactions.Value[i].FromAddr, _ = GetAddressDetail(v.FromAddr.Address, client)
		result.TopTransactions.Value[i].ToAddr, _ = GetAddressDetail(v.ToAddr.Address, client)
	}
	for i, v := range result.TopTransactions.GasFee {
		result.TopTransactions.GasFee[i].FromAddr, _ = GetAddressDetail(v.FromAddr.Address, client)
		result.TopTransactions.GasFee[i].ToAddr, _ = GetAddressDetail(v.ToAddr.Address, client)
	}
	for i, v := range result.TopTransactions.DataSize {
		result.TopTransactions.DataSize[i].FromAddr, _ = GetAddressDetail(v.FromAddr.Address, client)
		result.TopTransactions.DataSize[i].ToAddr, _ = GetAddressDetail(v.ToAddr.Address, client)
	}
}

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
