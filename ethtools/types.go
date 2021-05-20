package ethtools

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
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
	return fmt.Sprintf("%s%s \t gasfee: %8s ETH  \t  val: %14v \t datasize: %-8d \t\t %s %-20s -> %s %s", stats.Hash, failedMsg, WeiBigIntToEthString(stats.GasFee, 4), WeiBigIntToEthString(stats.Value, 4), stats.DataSize, stats.FromAddr.Address, stats.FromAddr.Name, stats.ToAddr.Address, stats.ToAddr.Name)
}

type TopTransactionData struct {
	GasFee   []TxStats
	Value    []TxStats
	DataSize []TxStats
}
