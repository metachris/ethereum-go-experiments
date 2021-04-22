package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	nodeAddr := "http://95.217.145.161:8545"
	// nodeAddr := "http://localhost:7545"  // ganache

	fmt.Println("Connecting to Ethereum node at", nodeAddr)
	client, err := ethclient.Dial(nodeAddr)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Connected")
	_ = client // we'll use this in the upcoming sections

	// address := common.HexToAddress("0x160cD5F7ab0DdaBe67A3e393e86F39c5329c4622")
	// fmt.Println(address.Hex())        // 0x71C7656EC7ab88b098defB751B7401B5f6d8976F
	// fmt.Println(address.Hash().Hex()) // 0x00000000000000000000000071c7656ec7ab88b098defb751b7401b5f6d8976f
	// fmt.Println(address.Bytes())      // [113 199 101 110 199 171 136 176 152 222 251 117 27 116 1 181 246 216 151 111]

	// Get balance
	// balance, err := client.BalanceAt(context.Background(), address, nil)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// // fmt.Println(balance) // 25893180161173005034

	// fbalance := new(big.Float)
	// fbalance.SetString(balance.String())
	// ethValue := new(big.Float).Quo(fbalance, big.NewFloat(math.Pow10(18)))
	// fmt.Println(ethValue) // 25.729324269165216041

	// GET BLOCK HEADER
	header, err := client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Latest block", header.Number.String())

	// GET TRANSACTION COUNT
	// count, err := client.TransactionCount(context.Background(), header.Hash())
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println(count)

	// GET FULL BLOCK
	// block, err := client.BlockByNumber(context.Background(), header.Number)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// printBlock(block)

	// Go over last x blocks
	var latestBlockHeight = header.Number
	var one = big.NewInt(1)

	var numBlocks int64 = 3
	var end = big.NewInt(0).Sub(latestBlockHeight, big.NewInt(numBlocks))

	// transactions := make([]types.Transaction, 1)
	var transactions types.Transactions = types.Transactions{}

	for blockHeight := new(big.Int).Set(latestBlockHeight); blockHeight.Cmp(end) > 0; blockHeight.Sub(blockHeight, one) {
		// fmt.Println(blockHeight, "fetching...")
		blockX, err := client.BlockByNumber(context.Background(), blockHeight)
		if err != nil {
			log.Fatal(err)
		}
		printBlock(blockX)
		transactions = append(transactions, blockX.Transactions()...)
	}
	fmt.Println("tx total", len(transactions))
}

func printBlock(block *types.Block) {
	t := time.Unix(int64(block.Header().Time), 0)
	fmt.Printf("%d \t %s \t tx=%d\n", block.Header().Number, t, len(block.Transactions()))
}
