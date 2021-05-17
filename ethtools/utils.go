package ethtools

import (
	"fmt"
	"log"
	"math"
	"math/big"
	"strings"
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

// Takes a token amount, looks up the contract and returns it as float with correct decimals
// eg. USDC has 6 decimals. 12345678 -> 12.345678
func TokenAmountToUnit(numTokens *big.Int, address string) (res *big.Float, symbol string, success bool) {
	// fmt.Println(address)
	detail, ok := AllAddressesFromJson[strings.ToLower(address)]
	if ok {
		decimals := uint64(detail.Decimals)
		divider := float64(1)
		if decimals > 0 {
			divider = float64(math.Pow(10, float64(decimals)))
		}

		res = new(big.Float).Quo(new(big.Float).SetInt(numTokens), big.NewFloat(divider))
		return res, detail.Symbol, true

		// resInt := new(big.Int).Div(numTokens, big.NewInt(int64(divider)))
		// formatted := formatInt(int(resInt.Uint64()))
		// return fmt.Sprintf("%s %-5v", formatted, detail.Symbol), true
	} else {
		res = new(big.Float).SetInt(numTokens)
		return res, "", false
	}
}
