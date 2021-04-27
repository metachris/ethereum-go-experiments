package main

import (
	"fmt"
	"log"
	"math"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
)

var one = big.NewInt(1)

func makeTime(dayString string, hour int, min int) time.Time {
	dateString := fmt.Sprintf("%sT%02d:%02d:00Z", dayString, hour, min)
	// fmt.Println(ts)
	t, err := time.Parse(time.RFC3339, dateString)
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Println(t)
	return t
}

func getTargetBlocknumber(utcTimestamp int64) int64 {
	// Create a time.Time instance

	// Calculate block-difference from reference block
	referenceBlockNumber := int64(12323940)
	referenceBlockTimestamp := int64(1619546404) // 2021-04-27 19:49:13 +0200 CEST
	avgBlockSec := int64(13)

	secDiff := referenceBlockTimestamp - utcTimestamp
	blocksDiff := secDiff / avgBlockSec
	targetBlock := referenceBlockNumber - blocksDiff
	return targetBlock
}

// func testTimestamp() {
// 	dayStr := "2021-01-27"
// 	hour := 17 // 0:00 - 0:59

// 	targetBlockNumber := getTargetBlocknumber(dayStr, hour)
// 	fmt.Println("target block", targetBlockNumber)
// }

func Abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

func isBigIntZero(n *big.Int) bool {
	return len(n.Bits()) == 0
}

func weiToEth(wei *big.Int) (ethValue *big.Float) {
	// wei / 10^18
	fbalance := new(big.Float)
	fbalance.SetString(wei.String())
	ethValue = new(big.Float).Quo(fbalance, big.NewFloat(math.Pow10(18)))
	return
}

func printBlock(block *types.Block) {
	t := time.Unix(int64(block.Header().Time), 0)
	fmt.Printf("%d \t %s \t %d \t tx=%d\n", block.Header().Number, t, block.Header().Time, len(block.Transactions()))
}

type MyBigInt big.Int

func (i MyBigInt) MarshalJSON() ([]byte, error) {
	i2 := big.Int(i)
	return []byte(fmt.Sprintf(`"%s"`, i2.String())), nil
}
