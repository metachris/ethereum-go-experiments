package ethstats

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/metachris/ethereum-go-experiments/config"
	"github.com/metachris/ethereum-go-experiments/consts"
)

// Keys for top addresses
var (
	TopAddressValueSent     string = "ValueSent"
	TopAddressValueReceived        = "ValueReceived"
)

type AnalysisResult struct {
	StartBlockNumber    int64
	StartBlockTimestamp uint64
	EndBlockNumber      int64
	EndBlockTimestamp   uint64

	Addresses          map[string]*AddressStats `json:"-"`
	TopAddresses       map[string][]AddressStats
	TopTransactions    TopTransactionData // todo: refactor for generic counters, like topaddresses
	TaggedTransactions []TxStats

	TxTypes       map[uint8]int
	ValueTotalWei *big.Int

	NumBlocks          int
	NumBlocksWithoutTx int
	GasUsed            *big.Int
	GasFeeTotal        *big.Int
	GasFeeFailedTx     *big.Int

	NumTransactions              int
	NumTransactionsFailed        int
	NumTransactionsWithZeroValue int
	NumTransactionsWithData      int

	NumTransactionsErc20Transfer  int
	NumTransactionsErc721Transfer int

	NumFlashbotsTransactionsSuccess int
	NumFlashbotsTransactionsFailed  int
}

func NewAnalysis() *AnalysisResult {
	return &AnalysisResult{
		ValueTotalWei: new(big.Int),
		TxTypes:       make(map[uint8]int),
		Addresses:     make(map[string]*AddressStats),
		TopAddresses:  make(map[string][]AddressStats),

		GasUsed:        new(big.Int),
		GasFeeTotal:    new(big.Int),
		GasFeeFailedTx: new(big.Int),

		TopTransactions: TopTransactionData{
			GasFee:   make([]TxStats, 0, config.GetConfig().NumTopTransactions),
			Value:    make([]TxStats, 0, config.GetConfig().NumTopTransactions),
			DataSize: make([]TxStats, 0, config.GetConfig().NumTopTransactions),
		},
	}
}

func (result *AnalysisResult) GetOrCreateAddressStats(address *common.Address) *AddressStats {
	if address == nil {
		return NewAddressStats("")
	}

	addr := strings.ToLower(address.String())
	addrStats, isAddressKnown := result.Addresses[addr]
	if !isAddressKnown {
		addrStats = NewAddressStats(addr)
		result.Addresses[addr] = addrStats
	}

	return addrStats
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

func (result *AnalysisResult) AddTaggedTransactionStats(txStats TxStats, tag string, client *ethclient.Client) {
	txStats.Tag = tag
	txStats.FromAddr.EnsureIsLoaded(client)
	txStats.ToAddr.EnsureIsLoaded(client)
	result.TaggedTransactions = append(result.TaggedTransactions, txStats)
}

func (result *AnalysisResult) AddTransaction(client *ethclient.Client, tx *types.Transaction, receipt *types.Receipt) {
	result.AddTxToTopList(tx, receipt)

	txToAddrStats := result.GetOrCreateAddressStats(tx.To())
	txFromAddrStats := result.GetOrCreateAddressStats(nil)
	if from, err := GetTxSender(tx); err == nil {
		txFromAddrStats = result.GetOrCreateAddressStats(&from)
	}

	txToAddrStats.Add1(consts.NumTxReceived)
	txFromAddrStats.Add1(consts.NumTxSent)

	// Gas used is paid and counted no matter if transaction failed or succeeded
	txSuccess := true        // default, used if no receipt
	txGasUsed := common.Big1 // default, used if no receipt
	if receipt != nil {
		txSuccess = receipt.Status == 1
		txGasUsed = big.NewInt(int64(receipt.GasUsed))
	}

	txGasFee := new(big.Int).Mul(txGasUsed, tx.GasPrice())
	result.GasFeeTotal = new(big.Int).Add(result.GasFeeTotal, txGasFee)
	txFromAddrStats.Add(consts.GasUsed, txGasUsed)
	txFromAddrStats.Add(consts.GasFeeTotal, txGasFee)

	if !txSuccess {
		result.NumTransactionsFailed += 1
		result.GasFeeFailedTx = new(big.Int).Add(result.GasFeeFailedTx, txGasFee)

		txFromAddrStats.Add1(consts.NumTxSentFailed)
		txFromAddrStats.Add(consts.GasFeeFailedTx, txGasFee)

		txToAddrStats.Add1(consts.NumTxReceivedFailed)

		// Count failed flashbots tx
		if len(tx.Data()) > 0 && tx.GasPrice().Uint64() == 0 {
			result.NumFlashbotsTransactionsFailed += 1
			fmt.Printf("Flashbots fail tx: https://etherscan.io/tx/%s\n", tx.Hash())
			result.AddTaggedTransactionStats(NewTxStatsFromTransactions(tx, receipt), consts.FlashBotsFailedTx, client)
			txFromAddrStats.Add1(consts.FlashBotsFailedTxSent)
		}

		return
	}

	// TX was successful...
	result.ValueTotalWei = result.ValueTotalWei.Add(result.ValueTotalWei, tx.Value())
	result.TxTypes[tx.Type()] += 1

	txFromAddrStats.Add1(consts.NumTxSentSuccess)
	txFromAddrStats.Add(consts.ValueSentWei, tx.Value())

	txToAddrStats.Add1(consts.NumTxReceivedSuccess)
	txToAddrStats.Add(consts.ValueReceivedWei, tx.Value())

	if isBigIntZero(tx.Value()) {
		result.NumTransactionsWithZeroValue += 1
	}

	// Check for smart contract calls: token transfer
	data := tx.Data()
	if len(data) > 0 {
		result.NumTransactionsWithData += 1

		txFromAddrStats.Add1(consts.NumTxWithDataSent)
		txToAddrStats.Add1(consts.NumTxWithDataReceived)

		// If no gas price, then count as MEV/Flashbots tx
		if tx.GasPrice().Uint64() == 0 {
			result.NumFlashbotsTransactionsSuccess += 1
			txFromAddrStats.Add1(consts.NumTxFlashbotsSent)
			txToAddrStats.Add1(consts.NumTxFlashbotsReceived)

			if config.GetConfig().DebugPrintFlashbotsTx {
				fmt.Printf("Flashbots ok tx: https://etherscan.io/tx/%s\n", tx.Hash())
			}
		}

		if len(data) > 4 {
			methodId := hex.EncodeToString(data[:4])
			// fmt.Println(methodId)
			if methodId == "a9059cbb" || methodId == "23b872dd" {
				txFromAddrStats.EnsureAddressDetails(client)
				txToAddrStats.EnsureAddressDetails(client)

				// Calculate and store the number of tokens transferred
				valueSenderStats := txFromAddrStats
				valueReceiverStats := txToAddrStats
				var value string
				switch methodId {
				case "a9059cbb": // transfer
					_receiverAddrString := hex.EncodeToString(data[4:36])
					_receiverAddr := common.HexToAddress(_receiverAddrString)
					// fmt.Println("transfer. to", _receiverAddr, "\t .. tx from:", txFromAddrStats.Address)
					valueReceiverStats = result.GetOrCreateAddressStats(&_receiverAddr)

					value = hex.EncodeToString(data[36:68])

				case "23b872dd": // transferFrom
					_senderAddrString := hex.EncodeToString(data[4:36])
					_senderAddr := common.HexToAddress(_senderAddrString)
					valueSenderStats = result.GetOrCreateAddressStats(&_senderAddr)

					_receiverAddrString := hex.EncodeToString(data[36:68])
					_receiverAddr := common.HexToAddress(_receiverAddrString)
					valueReceiverStats = result.GetOrCreateAddressStats(&_receiverAddr)

					value = hex.EncodeToString(data[68:100])
					// fmt.Println("transferFrom. from", _senderAddr, ", tx", txFromAddrStats.Address, "... to", _receiverAddr)
				}

				valBigInt := new(big.Int)
				valBigInt.SetString(value, 16)

				// If ERC2 SC call
				if txToAddrStats.AddressDetail.IsErc20() {
					result.NumTransactionsErc20Transfer += 1

					// Count sender
					txFromAddrStats.Add1(consts.NumTxErc20Sent)
					txFromAddrStats.Add(consts.Erc20TokensSent, valBigInt)
					if txFromAddrStats.AddressDetail.Address != valueSenderStats.AddressDetail.Address {
						valueSenderStats.Add1(consts.NumTxErc20Sent)
						valueSenderStats.Add(consts.Erc20TokensSent, valBigInt)
					}

					// Count SC transfer calls
					txToAddrStats.Add1(consts.NumTxErc20Transfer)
					txToAddrStats.Add(consts.Erc20TokensTransferred, valBigInt)

					// Count token transfer receiver
					valueReceiverStats.Add1(consts.NumTxErc20Received)
					valueReceiverStats.Add(consts.Erc20TokensReceived, valBigInt)

				} else if txToAddrStats.AddressDetail.IsErc721() {
					// fmt.Println("ERC721 transfer", tx.Hash(), "from", txFromAddrStats.AddressDetail.Address)
					result.NumTransactionsErc721Transfer += 1

					// Count sender
					txFromAddrStats.Add1(consts.NumTxErc721Sent)
					// fmt.Println(1, txFromAddrStats.AddressDetail.Address, txFromAddrStats.Stats[consts.NumTxErc721Sent])
					if txFromAddrStats.AddressDetail.Address != valueSenderStats.AddressDetail.Address {
						valueSenderStats.Add1(consts.NumTxErc721Sent)
					}

					// Count SC
					txToAddrStats.Add1(consts.NumTxErc721Transfer)

					// Count token receiver
					valueReceiverStats.Add1(consts.NumTxErc721Received)
				}
			}
		}
	}
}

// AddTxToTopList builds the top transactions list
func (result *AnalysisResult) AddTxToTopList(tx *types.Transaction, receipt *types.Receipt) {
	stats := NewTxStatsFromTransactions(tx, receipt)

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

func (analysis *AnalysisResult) EnsureTopTransactionAddressDetails(client *ethclient.Client) {
	for i := range analysis.TopTransactions.Value {
		analysis.TopTransactions.Value[i].FromAddr.EnsureIsLoaded(client)
		analysis.TopTransactions.Value[i].ToAddr.EnsureIsLoaded(client)
	}
	for i := range analysis.TopTransactions.GasFee {
		analysis.TopTransactions.GasFee[i].FromAddr.EnsureIsLoaded(client)
		analysis.TopTransactions.GasFee[i].ToAddr.EnsureIsLoaded(client)
	}
	for i := range analysis.TopTransactions.DataSize {
		analysis.TopTransactions.DataSize[i].FromAddr.EnsureIsLoaded(client)
		analysis.TopTransactions.DataSize[i].ToAddr.EnsureIsLoaded(client)
	}
}

// AnalysisResult.BuildTopAddresses() sorts the addresses into TopAddresses after all blocks have been added.
// Ensures that details for all top addresses are queried from blockchain
func (analysis *AnalysisResult) BuildTopAddresses(client *ethclient.Client) {
	numEntries := config.GetConfig().NumTopAddresses

	// Convert addresses from map into a sortable array
	addressArray := make([]AddressStats, 0, len(analysis.Addresses))
	for _, k := range analysis.Addresses {
		addressArray = append(addressArray, *k)
	}

	// Get top entries for each key
	for _, key := range consts.AddressStatsKeys {
		analysis.TopAddresses[key] = GetTopAddressesForStats(&addressArray, client, key, numEntries)
	}
}

// GetAllTopAddressStats returns a list of all top addresses across any of the statistics
func (analysis *AnalysisResult) GetAllTopAddressStats() []AddressStats {
	ret := make([]AddressStats, 0)
	for k := range analysis.TopAddresses {
		ret = append(ret, analysis.TopAddresses[k]...)
	}
	return ret
}
