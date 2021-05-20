package main

import (
	"context"
	"ethstats/ethtools"
	"fmt"
	"math"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	// resetPtr := flag.Bool("reset", false, "reset database")
	// flag.Parse()

	config := ethtools.GetConfig()
	client, err := ethclient.Dial(config.EthNode)
	ethtools.Perror(err)

	txHash := common.HexToHash("0x3308ca87b00911f3b4aac572f526d41d7786c8b4d845950e83020ac8596353c0")
	tx, _, err := client.TransactionByHash(context.Background(), txHash)
	ethtools.Perror(err)
	// fmt.Println(isPending)

	// block, err := client.BlockByNumber(context.Background(), big.NewInt(12465297))

	receipt, err := client.TransactionReceipt(context.Background(), tx.Hash())
	ethtools.Perror(err)
	fee := float64((receipt.GasUsed * tx.GasPrice().Uint64())) / math.Pow10(18)
	fmt.Println(receipt.GasUsed, tx.GasPrice(), fee)
}
