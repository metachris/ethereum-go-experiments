package core

import (
	"fmt"
	"math/big"
	"sort"
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
type AnalysisData struct {
	StartBlockNumber    int64
	StartBlockTimestamp uint64
	EndBlockNumber      int64
	EndBlockTimestamp   uint64

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

// GetAllTopAddressStats returns a list of all top addresses across any of the statistics
func (analysis *AnalysisData) GetAllTopAddressStats() []AddressStats {
	ret := make([]AddressStats, 0)
	for k := range analysis.TopAddresses {
		ret = append(ret, analysis.TopAddresses[k]...)
	}
	return ret
}

type IAddressDetailService interface {
	EnsureIsLoaded(a *AddressDetail, client *ethclient.Client)
}

type Analysis struct {
	Data      AnalysisData
	Addresses map[string]*AddressStats `json:"-"`

	addressDetailService IAddressDetailService
	client               *ethclient.Client
}

func NewAnalysis(cfg Config, client *ethclient.Client, addressDetailsService IAddressDetailService) *Analysis {
	data := AnalysisData{
		ValueTotalWei: new(big.Int),
		TxTypes:       make(map[uint8]int),
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

	return &Analysis{
		Data:                 data,
		Addresses:            make(map[string]*AddressStats),
		addressDetailService: addressDetailsService,
		client:               client,
	}
}

func (result *Analysis) GetOrCreateAddressStats(address *common.Address) *AddressStats {
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

func (analysis *Analysis) EnsureAddressDetailIsLoaded(a *AddressDetail) {
	analysis.addressDetailService.EnsureIsLoaded(a, analysis.client)
}

func (analysis *Analysis) TagTransactionStats(txStats TxStats, tag string, client *ethclient.Client) {
	txStats.Tag = tag
	analysis.EnsureAddressDetailIsLoaded(&txStats.FromAddr)
	analysis.EnsureAddressDetailIsLoaded(&txStats.ToAddr)
	analysis.Data.TaggedTransactions = append(analysis.Data.TaggedTransactions, txStats)
}

// AnalysisResult.BuildTopAddresses() sorts the addresses into TopAddresses after all blocks have been added.
// Ensures that details for all top addresses are queried from blockchain
func (analysis *Analysis) BuildTopAddresses() {
	numEntries := GetConfig().NumTopAddresses

	// Convert addresses from map into a sortable array
	addressArray := make([]AddressStats, 0, len(analysis.Addresses))
	for _, k := range analysis.Addresses {
		addressArray = append(addressArray, *k)
	}

	// Get top entries for each key
	for _, key := range consts.AddressStatsKeys {
		analysis.BuildTopAddressesForKey(&addressArray, key, numEntries)
	}
}

// Takes pointer to all addresses and builds a top-list
func (analysis *Analysis) BuildTopAddressesForKey(allAddresses *[]AddressStats, key string, numItems int) (ret []AddressStats) {
	ret = make([]AddressStats, 0, numItems)
	sort.SliceStable(*allAddresses, func(i, j int) bool {
		a := (*allAddresses)[i].Get(key)
		b := (*allAddresses)[j].Get(key)
		return a.Cmp(b) == 1
	})

	for i := 0; i < len(*allAddresses) && i < numItems; i++ {
		item := (*allAddresses)[i]
		if item.Get(key).Cmp(common.Big0) == 1 {
			analysis.EnsureAddressDetailIsLoaded(&item.AddressDetail)
			ret = append(ret, item)
		}
	}

	analysis.Data.TopAddresses[key] = ret
	return ret
}

// AddTxToTopList builds the top transactions list
func (analysis *Analysis) AddTxToTopList(tx *types.Transaction, receipt *types.Receipt) {
	stats := NewTxStatsFromTransactions(tx, receipt)

	// Sort by Gas fee
	if len(analysis.Data.TopTransactions.GasFee) < cap(analysis.Data.TopTransactions.GasFee) { // add new item to array
		analysis.Data.TopTransactions.GasFee = append(analysis.Data.TopTransactions.GasFee, stats)
	} else if stats.GasFee.Cmp(analysis.Data.TopTransactions.GasFee[len(analysis.Data.TopTransactions.GasFee)-1].GasFee) == 1 { // replace last item
		analysis.Data.TopTransactions.GasFee[len(analysis.Data.TopTransactions.GasFee)-1] = stats
	}
	sort.SliceStable(analysis.Data.TopTransactions.GasFee, func(i, j int) bool {
		return analysis.Data.TopTransactions.GasFee[i].GasFee.Cmp(analysis.Data.TopTransactions.GasFee[j].GasFee) == 1
	})

	// Sort by value
	if len(analysis.Data.TopTransactions.Value) < cap(analysis.Data.TopTransactions.Value) { // add new item to array
		analysis.Data.TopTransactions.Value = append(analysis.Data.TopTransactions.Value, stats)
	} else if stats.Value.Cmp(analysis.Data.TopTransactions.Value[len(analysis.Data.TopTransactions.Value)-1].Value) == 1 {
		analysis.Data.TopTransactions.Value[len(analysis.Data.TopTransactions.Value)-1] = stats
	}
	sort.SliceStable(analysis.Data.TopTransactions.Value, func(i, j int) bool {
		return analysis.Data.TopTransactions.Value[i].Value.Cmp(analysis.Data.TopTransactions.Value[j].Value) == 1
	})

	// Sort by len(data)
	if len(analysis.Data.TopTransactions.DataSize) < cap(analysis.Data.TopTransactions.DataSize) { // add new item to array
		analysis.Data.TopTransactions.DataSize = append(analysis.Data.TopTransactions.DataSize, stats)
	} else if stats.DataSize > analysis.Data.TopTransactions.DataSize[len(analysis.Data.TopTransactions.DataSize)-1].DataSize {
		analysis.Data.TopTransactions.DataSize[len(analysis.Data.TopTransactions.DataSize)-1] = stats
	}
	sort.SliceStable(analysis.Data.TopTransactions.DataSize, func(i, j int) bool {
		return analysis.Data.TopTransactions.DataSize[i].DataSize > analysis.Data.TopTransactions.DataSize[j].DataSize
	})
}

func (analysis *Analysis) EnsureTopTransactionAddressDetails() {
	for i := range analysis.Data.TopTransactions.Value {
		analysis.EnsureAddressDetailIsLoaded(&analysis.Data.TopTransactions.Value[i].FromAddr)
		analysis.EnsureAddressDetailIsLoaded(&analysis.Data.TopTransactions.Value[i].ToAddr)
	}
	for i := range analysis.Data.TopTransactions.GasFee {
		analysis.EnsureAddressDetailIsLoaded(&analysis.Data.TopTransactions.GasFee[i].FromAddr)
		analysis.EnsureAddressDetailIsLoaded(&analysis.Data.TopTransactions.GasFee[i].ToAddr)
	}
	for i := range analysis.Data.TopTransactions.DataSize {
		analysis.EnsureAddressDetailIsLoaded(&analysis.Data.TopTransactions.DataSize[i].FromAddr)
		analysis.EnsureAddressDetailIsLoaded(&analysis.Data.TopTransactions.DataSize[i].ToAddr)
	}
}
