package ethtools

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"sort"
	"strings"
	"text/template"

	"github.com/ethereum/go-ethereum/common"
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

func NewResult() *AnalysisResult {
	return &AnalysisResult{
		ValueTotalWei:  new(big.Int),
		TxTypes:        make(map[uint8]int),
		Addresses:      make(map[string]*AddressStats),
		GasUsed:        new(big.Int),
		GasFeeTotal:    new(big.Int),
		GasFeeFailedTx: new(big.Int),

		TopTransactions: TopTransactionData{
			GasFee:   make([]TxStats, 0, GetConfig().NumTopTransactions),
			Value:    make([]TxStats, 0, GetConfig().NumTopTransactions),
			DataSize: make([]TxStats, 0, GetConfig().NumTopTransactions),
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

func (result *AnalysisResult) AddTransaction(client *ethclient.Client, tx *types.Transaction, receipt *types.Receipt) {
	result.AddTxToTopList(tx, receipt)

	// Get the address stats instance for receiver and sender
	txToAddrStats := result.GetOrCreateAddressStats(tx.To())
	txFromAddrStats := result.GetOrCreateAddressStats(nil)
	if from, err := GetTxSender(tx); err == nil {
		txFromAddrStats = result.GetOrCreateAddressStats(&from)
	}

	// Gas used is paid and counted no matter if transaction failed or succeeded
	txSuccess := true        // default, used if no receipt
	txGasUsed := common.Big1 // default, used if no receipt
	if receipt != nil {
		txSuccess = receipt.Status == 1
		txGasUsed = big.NewInt(int64(receipt.GasUsed))
	}

	txGasFee := new(big.Int).Mul(txGasUsed, tx.GasPrice())
	txFromAddrStats.GasUsed = new(big.Int).Add(txFromAddrStats.GasUsed, txGasUsed)
	txFromAddrStats.GasFeeTotal = new(big.Int).Add(txFromAddrStats.GasFeeTotal, txGasFee)
	result.GasFeeTotal = new(big.Int).Add(result.GasFeeTotal, txGasFee)

	if !txSuccess {
		result.NumTransactionsFailed += 1
		txFromAddrStats.NumTxSentFailed += 1
		txToAddrStats.NumTxReceivedFailed += 1
		result.GasFeeFailedTx = new(big.Int).Add(result.GasFeeFailedTx, txGasFee)
		txFromAddrStats.GasFeeFailedTx = new(big.Int).Add(txFromAddrStats.GasFeeFailedTx, txGasFee)

		// Count failed flashbots tx
		if len(tx.Data()) > 0 && tx.GasPrice().Uint64() == 0 {
			result.NumFlashbotsTransactionsFailed += 1
			if GetConfig().DebugPrintFlashbotsTx {
				fmt.Printf("Flashbots fail tx: https://etherscan.io/tx/%s\n", tx.Hash())
			}
		}

		return
	}

	// TX was successful...
	result.ValueTotalWei = result.ValueTotalWei.Add(result.ValueTotalWei, tx.Value())
	result.TxTypes[tx.Type()] += 1

	txFromAddrStats.NumTxSentSuccess += 1
	txFromAddrStats.ValueSentWei = new(big.Int).Add(txFromAddrStats.ValueSentWei, tx.Value())

	txToAddrStats.NumTxReceivedSuccess += 1
	txToAddrStats.ValueReceivedWei = new(big.Int).Add(txToAddrStats.ValueReceivedWei, tx.Value())

	if isBigIntZero(tx.Value()) {
		result.NumTransactionsWithZeroValue += 1
	}

	// Check for smart contract calls: token transfer
	data := tx.Data()
	if len(data) > 0 {
		result.NumTransactionsWithData += 1
		txFromAddrStats.NumTxWithDataSent += 1
		txToAddrStats.NumTxWithDataReceived += 1

		// If no gas price, then it's probably a MEV/Flashbots tx
		if tx.GasPrice().Uint64() == 0 {
			result.NumFlashbotsTransactionsSuccess += 1
			txFromAddrStats.NumTxFlashbotsSent += 1
			txToAddrStats.NumTxFlashbotsReceived += 1

			if GetConfig().DebugPrintFlashbotsTx {
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
					txFromAddrStats.NumTxErc20Sent += 1
					txFromAddrStats.Erc20TokensSent = new(big.Int).Add(txFromAddrStats.Erc20TokensSent, valBigInt)
					if txFromAddrStats.Address != valueSenderStats.Address {
						valueSenderStats.NumTxErc20Sent += 1
						valueSenderStats.Erc20TokensSent = new(big.Int).Add(valueSenderStats.Erc20TokensSent, valBigInt)
					}

					// Count SC transfer calls
					txToAddrStats.NumTxErc20Transfer += 1
					txToAddrStats.Erc20TokensTransferred = new(big.Int).Add(txToAddrStats.Erc20TokensTransferred, valBigInt)

					// Count token transfer receiver
					valueReceiverStats.NumTxErc20Received += 1
					valueReceiverStats.Erc20TokensReceived = new(big.Int).Add(valueReceiverStats.Erc20TokensReceived, valBigInt)

				} else if txToAddrStats.AddressDetail.IsErc721() {
					result.NumTransactionsErc721Transfer += 1

					// Count sender
					txFromAddrStats.NumTxErc721Sent += 1
					if txFromAddrStats.Address != valueSenderStats.Address {
						valueSenderStats.NumTxErc721Sent += 1
					}

					// Count SC
					txToAddrStats.NumTxErc721Transfer += 1

					// Count token receiver
					valueReceiverStats.NumTxErc721Received += 1
				}
			}
		}
	}
}

// AddTxToTopList builds the top transactions list
func (result *AnalysisResult) AddTxToTopList(tx *types.Transaction, receipt *types.Receipt) {
	txSuccess := true
	txGasUsed := common.Big1
	if receipt != nil {
		txSuccess = receipt.Status == 1
		txGasUsed = big.NewInt(int64(receipt.GasUsed))
	}

	txGasFee := new(big.Int).Mul(txGasUsed, tx.GasPrice())

	to := NewAddressDetail("")
	if tx.To() != nil {
		to = NewAddressDetail(tx.To().String())
	}

	from := NewAddressDetail("")
	if fromAddr, err := GetTxSender(tx); err == nil {
		from = NewAddressDetail(fromAddr.Hex())
	}

	stats := TxStats{
		Hash:     tx.Hash().Hex(),
		GasFee:   txGasFee,
		GasUsed:  txGasUsed,
		Value:    tx.Value(),
		DataSize: len(tx.Data()),
		Success:  txSuccess,
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

	// erc20-tx transferred
	sort.SliceStable(_addresses, func(i, j int) bool {
		return _addresses[i].NumTxErc20Transfer > _addresses[j].NumTxErc20Transfer
	})
	for i := 0; i < len(_addresses) && i < config.NumTopAddressesLarge; i++ {
		if _addresses[i].NumTxErc20Transfer > 0 {
			_addresses[i].EnsureAddressDetails(client)
			result.TopAddresses.NumTxErc20Transfers = append(result.TopAddresses.NumTxErc20Transfers, _addresses[i])
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

	// erc721 transferred
	sort.SliceStable(_addresses, func(i, j int) bool {
		return _addresses[i].NumTxErc721Transfer > _addresses[j].NumTxErc721Transfer
	})
	for i := 0; i < len(_addresses) && i < config.NumTopAddressesLarge; i++ {
		if _addresses[i].NumTxErc721Transfer > 0 {
			_addresses[i].EnsureAddressDetails(client)
			result.TopAddresses.NumTxErc721Transfers = append(result.TopAddresses.NumTxErc721Transfers, _addresses[i])
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

// UpdateTxAddressDetails gets the address details from cache for each top transaction
func (result *AnalysisResult) ToHtml(filename string) {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0600)
	Perror(err)
	err = f.Truncate(0)
	Perror(err)
	defer f.Close()

	t, _ := template.ParseFiles("templates/stats.html")
	t.Execute(f, result)
}
