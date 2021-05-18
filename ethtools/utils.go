package ethtools

import (
	"fmt"
	"log"
	"math"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
)

var One = big.NewInt(1)

func MakeTime(dayString string, hour int, min int) time.Time {
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

	secDiff := referenceBlockTimestamp - utcTimestamp
	// fmt.Println("secDiff", secDiff)
	blocksDiff := secDiff / 13
	targetBlock := referenceBlockNumber - blocksDiff
	return targetBlock
}

func Abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

func isBigIntZero(n *big.Int) bool {
	return len(n.Bits()) == 0
}

func WeiToEth(wei *big.Int) (ethValue *big.Float) {
	// wei / 10^18
	fbalance := new(big.Float)
	fbalance.SetString(wei.String())
	ethValue = new(big.Float).Quo(fbalance, big.NewFloat(math.Pow10(18)))
	return
}

func printBlock(block *types.Block) {
	t := time.Unix(int64(block.Header().Time), 0).UTC()
	fmt.Printf("%d \t %s \t %d \t tx=%d \t %d\n", block.Header().Number, t, block.Header().Time, len(block.Transactions()), block.GasUsed())
}

// panic on error
func Perror(err error) {
	if err != nil {
		panic(err)
	}
}

func DebugPrintf(format string, a ...interface{}) {
	if GetConfig().Debug {
		_, _ = fmt.Printf(format, a...)
	}
}

func DebugPrintln(a ...interface{}) {
	if GetConfig().Debug {
		_, _ = fmt.Println(a...)
	}
}
