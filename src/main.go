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

	// Start of analysis
	dayStr := "2020-04-27"
	hour := 17 // 0:00 - 0:59
	startTime := makeTime(dayStr, hour, 0)
	startTimestamp := startTime.Unix()

	// End of analysis
	endTimestamp := startTimestamp + 10 // 10 sec

	fmt.Println("Connecting to Ethereum node at", nodeAddr)
	client, err := ethclient.Dial(nodeAddr)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("target time:", startTimestamp, "/", startTime)

	block := getBlockAtTimestamp(client, startTimestamp)
	tm := time.Unix(int64(block.Time()), 0)
	fmt.Println("block:", block.Number(), block.Time(), tm)

	analyzeBlocks(block.Number(), endTimestamp)
}

// Analyze blocks starting at specific block number, until a certain target timestamp
func analyzeBlocks(startBlockNumber *big.Int, endTimestamp int64) {

}

// getBlockAtTimestamp searches for a block that is within 30 seconds of the target timestamp.
// Guesses a block number, downloads it, and narrows down time gap if needed until block is found.
//
// TODO: After block found, check one block further, and use that if even better match
func getBlockAtTimestamp(client *ethclient.Client, targetTimestamp int64) *types.Block {
	currentBlockNumber := getTargetBlocknumber(targetTimestamp)

	for {
		fmt.Println("Checking block:", currentBlockNumber)
		blockNumber := big.NewInt(currentBlockNumber)
		block, err := client.BlockByNumber(context.Background(), blockNumber)
		if err != nil {
			log.Fatal(err)
		}

		secDiff := int64(block.Time()) - targetTimestamp
		fmt.Println("block", currentBlockNumber, "time", block.Time(), "diff", secDiff)
		if Abs(secDiff) < 30 {
			return block
		}

		// Not found. Check for better block
		blockDiff := secDiff / 13
		currentBlockNumber -= blockDiff
	}
}
