package ethstats

import (
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

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

func (stats *AddressStats) EnsureAddressDetails(client *ethclient.Client) {
	if !stats.AddressDetail.IsLoaded() && client != nil {
		stats.AddressDetail, _ = GetAddressDetail(stats.AddressDetail.Address, client)
	}
}

// Takes pointer to all addresses and builds a top-list
func GetTopAddressesForStats(allAddresses *[]AddressStats, client *ethclient.Client, key string, numItems int) (ret []AddressStats) {
	ret = make([]AddressStats, 0, numItems)
	sort.SliceStable(*allAddresses, func(i, j int) bool {
		a := (*allAddresses)[i].Get(key)
		b := (*allAddresses)[j].Get(key)
		return a.Cmp(b) == 1
	})

	for i := 0; i < len(*allAddresses) && i < numItems; i++ {
		item := (*allAddresses)[i]
		if item.Get(key).Cmp(common.Big0) == 1 {
			item.EnsureAddressDetails(client)
			ret = append(ret, item)
		}
	}

	return ret
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
