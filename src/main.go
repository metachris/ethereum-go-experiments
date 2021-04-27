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
	// nodeAddr := "https://mainnet.infura.io/v3/e03fe41147d548a8a8f55ecad18378fb"

	fmt.Println("Connecting to Ethereum node at", nodeAddr)
	client, err := ethclient.Dial(nodeAddr)
	if err != nil {
		log.Fatal(err)
	}

	// chainConfig := params.ChainConfig{ChainID: big.NewInt(1), EIP155Block: }

	// blockNumber := big.NewInt(12323940)
	// block, err := client.BlockByNumber(context.Background(), blockNumber)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println(block.Time())
	// tm := time.Unix(int64(block.Time()), 0)
	// fmt.Println(tm)

	// for _, tx := range block.Transactions() {
	// 	fmt.Println(tx.Hash().Hex()) // 0x5d49fcaa394c97ec8a9c3e7bd9e8388d420fb050a52083ca52ff24b3b65bc9c2
	// 	// fmt.Println(tx.Value().String())    // 10000000000000000
	// 	// fmt.Println(tx.Gas())               // 105000
	// 	// fmt.Println(tx.GasPrice().Uint64()) // 102000000000
	// 	// fmt.Println(tx.Nonce())             // 110644
	// 	// fmt.Println(tx.Data())              // []
	// 	// fmt.Println(tx.To().Hex())          // 0x55fE59D8Ad77035154dDd0AD0388D09Dd4047A8e

	// 	// if msg, err := tx.AsMessage(types.HomesteadSigner{}); err != nil {
	// 	if msg, err := tx.AsMessage(types.NewEIP155Signer(big.NewInt(1))); err == nil {
	// 		fmt.Println("x", msg.From().Hex()) // 0x0fD081e3Bb178dc45c0cb23202069ddA57064258
	// 	}
	// }

	/* TX Playground */
	// // txHex := "0x9e9d82e23fbab6161495cb068387e96ba67200676cbafea9a422b0450f59083b"
	// txHex := "0xf240459183000a8bc6cc1e3bfd3b05b76c7c594e72f896ae0b1b4eb442b7a298"
	// txHash := common.HexToHash(txHex)
	// tx, _, err := client.TransactionByHash(context.Background(), txHash)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// fmt.Println(tx.Hash().Hex()) // 0x5d49fcaa394c97ec8a9c3e7bd9e8388d420fb050a52083ca52ff24b3b65bc9c2
	// data := tx.Data()
	// if len(data) > 0 {
	// 	methodId := hex.EncodeToString(data[:4])
	// 	fmt.Println(data)
	// 	fmt.Println(methodId)
	// }
	// fmt.Printf("%x", data[:4])

	dayStr := "2020-04-27"
	hour := 17 // 0:00 - 0:59

	targetTime := makeTime(dayStr, hour)
	targetTimestamp := targetTime.Unix()
	fmt.Println("target time:", targetTimestamp, "/", targetTime)

	// targetBlockNumber := getTargetBlocknumber(timestamp)
	// fmt.Println("target block:", targetBlockNumber)

	// blockNumber := big.NewInt(targetBlockNumber)
	// block, err := client.BlockByNumber(context.Background(), blockNumber)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// tm := time.Unix(int64(block.Time()), 0)
	// fmt.Println("block timestamp:", block.Time(), "/", tm)
	// // fmt.Println(tm)
	// secDiff := int64(block.Time()) - timestamp
	// fmt.Println(secDiff)
	// if Abs(secDiff) > 30 {

	// }
	// fmt.Println("timestamp differs by", secDiff)
	block := getBlockAtTimestamp(client, targetTimestamp)
	fmt.Println("block:", block.Number())

}

// getBlockAtTimestamp searches for a block that is within 30 seconds of the target timestamp.
// Guesses a block number, downloads it, and narrows down time gap if needed until block is found.
func getBlockAtTimestamp(client *ethclient.Client, targetTimestamp int64) *types.Block {
	currentBlockNumber := getTargetBlocknumber(targetTimestamp)

	for {
		fmt.Println("target block:", currentBlockNumber)

		blockNumber := big.NewInt(currentBlockNumber)
		block, err := client.BlockByNumber(context.Background(), blockNumber)
		if err != nil {
			log.Fatal(err)
		}
		tm := time.Unix(int64(block.Time()), 0)
		fmt.Println("block timestamp:", block.Time(), "/", tm)
		// fmt.Println(tm)
		secDiff := int64(block.Time()) - targetTimestamp
		fmt.Println("block", currentBlockNumber, "time", block.Time(), "diff", secDiff)
		if Abs(secDiff) < 30 {
			fmt.Println("found match")
			return block
		}

		// Not found. Check for better block
		blockDiff := secDiff / 13
		currentBlockNumber -= blockDiff
		println("secDiff", secDiff, "blockDiff", blockDiff)
	}
}
