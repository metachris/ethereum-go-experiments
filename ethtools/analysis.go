package ethtools

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

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
			}
		}
	}
}

// AddTxToTopList builds the top transactions list
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
	for i := 0; i < len(_addresses) && i < config.NumTopAddresses; i++ {
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
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].NumTxReceivedFailed > _addresses[j].NumTxReceivedFailed })
	for i := 0; i < len(_addresses) && i < config.NumTopAddresses; i++ {
		if _addresses[i].NumTxReceivedFailed > 0 {
			_addresses[i].EnsureAddressDetails(client)
			result.TopAddresses.NumTxReceivedFailed = append(result.TopAddresses.NumTxReceivedFailed, _addresses[i])
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
	for i := 0; i < len(_addresses) && i < config.NumTopAddressesLarge; i++ {
		if _addresses[i].NumTxErc721Received > 0 {
			_addresses[i].EnsureAddressDetails(client)
			result.TopAddresses.NumTxErc721Received = append(result.TopAddresses.NumTxErc721Received, _addresses[i])
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

// UpdateTxAddressDetails gets the address details from cache for each top transaction
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
