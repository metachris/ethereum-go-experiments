package core

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/metachris/ethereum-go-experiments/consts"
)

type AddressType string

const (
	AddressTypeInit AddressType = "" // Init value

	// After detection
	AddressTypeErc20         AddressType = "Erc20"
	AddressTypeErc721        AddressType = "Erc721"
	AddressTypeErcToken      AddressType = "ErcToken" // Just recognized transfer call. Either ERC20 or ERC721. Will be updated on next sync.
	AddressTypeOtherContract AddressType = "OtherContract"
	AddressTypePubkey        AddressType = "Wallet" // nothing else detected, probably just a wallet
)

type AddressDetail struct {
	Address  string      `json:"address"`
	Type     AddressType `json:"type"`
	Name     string      `json:"name"`
	Symbol   string      `json:"symbol"`
	Decimals uint8       `json:"decimals"`
}

// Returns a new unknown address detail
func NewAddressDetail(address string) AddressDetail {
	return AddressDetail{Address: address, Type: AddressTypeInit}
}

func (a AddressDetail) String() string {
	return fmt.Sprintf("%s [%s] name=%s, symbol=%s, decimals=%d", a.Address, a.Type, a.Name, a.Symbol, a.Decimals)
}

func (a AddressDetail) AsAddressWithName(brackets bool) string {
	namePart := ""
	if len(a.Name) > 0 {
		if brackets {
			namePart = fmt.Sprintf(" (%s)", a.Name)
		} else {
			namePart = fmt.Sprintf(" %s", a.Name)
		}
	}
	return fmt.Sprintf("%s%s", a.Address, namePart)
}

func (a *AddressDetail) IsLoaded() bool {
	return a.Type != AddressTypeInit
}

func (a *AddressDetail) IsErc20() bool {
	return a.Type == AddressTypeErc20
}

func (a *AddressDetail) IsErc721() bool {
	return a.Type == AddressTypeErc721
}

func (a *AddressDetail) EnsureIsLoaded(client *ethclient.Client) {
	if a.IsLoaded() {
		return
	}

	// // b, _ := GetAddressDetail(a.Address, client)
	// a.Address = b.Address
	// a.Type = b.Type
	// a.Name = b.Name
	// a.Symbol = b.Symbol
	// a.Decimals = b.Decimals
}

//
// Address Stats
//
// AddressStats represents one address and accumulates statistics
type AddressStats struct {
	AddressDetail AddressDetail
	Stats         map[string]*big.Int
}

func NewAddressStats(address string) *AddressStats {
	return &AddressStats{
		AddressDetail: NewAddressDetail(address),
		Stats:         make(map[string]*big.Int),
	}
}

func (stats *AddressStats) Add(key string, val *big.Int) {
	if stats.Stats[key] == nil {
		stats.Stats[key] = new(big.Int)
	}
	stats.Stats[key] = new(big.Int).Add(stats.Stats[key], val)
}

func (stats *AddressStats) Add1(key string) {
	stats.Add(key, common.Big1)
}

// Get is guaranteed to return a non-nil *big.Int for the stat key
func (stats *AddressStats) Get(key string) *big.Int {
	if v, found := stats.Stats[key]; found && v != nil {
		return v
	} else {
		return new(big.Int)
	}
}

//
// TxStats
//
type TxStats struct {
	Hash     string
	FromAddr AddressDetail
	ToAddr   AddressDetail
	GasUsed  *big.Int
	GasFee   *big.Int
	Value    *big.Int
	DataSize int
	Success  bool
	Tag      string // internally used to mark specific txs
}

func NewTxStatsFromTransactions(tx *types.Transaction, receipt *types.Receipt) TxStats {
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

	return TxStats{
		Hash:     tx.Hash().Hex(),
		GasFee:   txGasFee,
		GasUsed:  txGasUsed,
		Value:    tx.Value(),
		DataSize: len(tx.Data()),
		Success:  txSuccess,
		FromAddr: from,
		ToAddr:   to,
	}
}

func (stats TxStats) String() string {
	failedMsg := "         "
	if !stats.Success {
		failedMsg = " (failed)"
	}
	tagMsg := ""
	if len(stats.Tag) > 0 {
		tagMsg = fmt.Sprintf("%s\t", stats.Tag)
	}
	return fmt.Sprintf("%s%s%s\tgasfee: %8s ETH\tval: %14v \t datasize: %-8d \t\t %s -> %s", tagMsg, stats.Hash, failedMsg, WeiBigIntToEthString(stats.GasFee, 4), WeiBigIntToEthString(stats.Value, 4), stats.DataSize, stats.FromAddr.AsAddressWithName(true), stats.ToAddr.AsAddressWithName(true))
}

type TopTransactionData struct {
	GasFee   []TxStats
	Value    []TxStats
	DataSize []TxStats
}

//
// Analysis
//
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

func NewAnalysis(cfg Config) *AnalysisResult {
	return &AnalysisResult{
		ValueTotalWei: new(big.Int),
		TxTypes:       make(map[uint8]int),
		Addresses:     make(map[string]*AddressStats),
		TopAddresses:  make(map[string][]AddressStats),

		GasUsed:        new(big.Int),
		GasFeeTotal:    new(big.Int),
		GasFeeFailedTx: new(big.Int),

		TopTransactions: TopTransactionData{
			GasFee:   make([]TxStats, 0, cfg.NumTopTransactions),
			Value:    make([]TxStats, 0, cfg.NumTopTransactions),
			DataSize: make([]TxStats, 0, cfg.NumTopTransactions),
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

func (result *AnalysisResult) TagTransactionStats(txStats TxStats, tag string, client *ethclient.Client) {
	txStats.Tag = tag
	txStats.FromAddr.EnsureIsLoaded(client)
	txStats.ToAddr.EnsureIsLoaded(client)
	result.TaggedTransactions = append(result.TaggedTransactions, txStats)
}

// AnalysisResult.BuildTopAddresses() sorts the addresses into TopAddresses after all blocks have been added.
// Ensures that details for all top addresses are queried from blockchain
func (analysis *AnalysisResult) BuildTopAddresses(client *ethclient.Client) {
	numEntries := GetConfig().NumTopAddresses

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
