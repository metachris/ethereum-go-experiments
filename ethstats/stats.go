package ethstats

import (
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/ethclient"
)

type AddressStats struct {
	Address       string
	AddressDetail AddressDetail

	NumTxSentSuccess     int
	NumTxSentFailed      int
	NumTxReceivedSuccess int
	NumTxReceivedFailed  int

	NumTxFlashbotsSent     int
	NumTxFlashbotsReceived int
	NumTxWithDataSent      int
	NumTxWithDataReceived  int

	NumTxErc20Sent      int // sender wallet
	NumTxErc721Sent     int // sender wallet
	NumTxErc20Received  int // receiver wallet (counted tx receiver and transfer receiver)
	NumTxErc721Received int // receiver wallet (counted tx receiver and transfer receiver)
	NumTxErc20Transfer  int // smart contract
	NumTxErc721Transfer int // smart contract

	ValueSentWei     *big.Int
	ValueReceivedWei *big.Int

	Erc20TokensSent        *big.Int
	Erc20TokensReceived    *big.Int
	Erc20TokensTransferred *big.Int // smart contract

	GasUsed        *big.Int
	GasFeeTotal    *big.Int
	GasFeeFailedTx *big.Int
}

func (stats *AddressStats) EnsureAddressDetails(client *ethclient.Client) {
	if !stats.AddressDetail.IsLoaded() && client != nil {
		stats.AddressDetail, _ = GetAddressDetail(stats.Address, client)
	}
}

func NewAddressStats(address string) *AddressStats {
	return &AddressStats{
		Address:                address,
		AddressDetail:          NewAddressDetail(address),
		ValueSentWei:           new(big.Int),
		ValueReceivedWei:       new(big.Int),
		Erc20TokensSent:        new(big.Int),
		Erc20TokensReceived:    new(big.Int),
		Erc20TokensTransferred: new(big.Int),
		GasUsed:                new(big.Int),
		GasFeeTotal:            new(big.Int),
		GasFeeFailedTx:         new(big.Int),
	}
}

// Takes pointer to all addresses and builds a top-list
func NewTopAddressList(allAddresses *[]AddressStats, client *ethclient.Client, numItems int, sortMethod func(i int, j int) bool, checkMethod func(a AddressStats) bool) (ret []AddressStats) {
	ret = make([]AddressStats, 0, numItems)
	sort.SliceStable(*allAddresses, sortMethod)

	for i := 0; i < len(*allAddresses) && i < numItems; i++ {
		item := &(*allAddresses)[i]
		item.EnsureAddressDetails(client)
		if checkMethod((*allAddresses)[i]) {
			ret = append(ret, *item)
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
}

func (stats TxStats) String() string {
	failedMsg := "         "
	if !stats.Success {
		failedMsg = " (failed)"
	}
	return fmt.Sprintf("%s%s \t gasfee: %8s ETH  \t  val: %14v \t datasize: %-8d \t\t %s -> %s", stats.Hash, failedMsg, WeiBigIntToEthString(stats.GasFee, 4), WeiBigIntToEthString(stats.Value, 4), stats.DataSize, stats.FromAddr.Address, stats.ToAddr.Address)
}

type TopTransactionData struct {
	GasFee   []TxStats
	Value    []TxStats
	DataSize []TxStats
}