package main

import (
	"context"
	"ethstats/ethtools"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	// resetPtr := flag.Bool("reset", false, "reset database")
	// flag.Parse()

	config := ethtools.GetConfig()
	client, err := ethclient.Dial(config.EthNode)
	ethtools.Perror(err)

	txHash := common.HexToHash("0xe8e1acda795c832068c3274cc6b50d92aad761a2d4ae964fdde12e3bae4fbe39")
	tx, _, err := client.TransactionByHash(context.Background(), txHash)
	ethtools.Perror(err)
	// fmt.Println(isPending)

	// block, err := client.BlockByNumber(context.Background(), big.NewInt(12465297))

	receipt, err := client.TransactionReceipt(context.Background(), tx.Hash())
	ethtools.Perror(err)
	fmt.Println(receipt.GasUsed, tx.GasPrice())
}
