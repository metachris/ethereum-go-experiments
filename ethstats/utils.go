package ethstats

import (
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var One = big.NewInt(1)

func MakeTime(dayString string, hour int, min int) time.Time {
	dateString := fmt.Sprintf("%sT%02d:%02d:00Z", dayString, hour, min)
	t, err := time.Parse(time.RFC3339, dateString)
	Perror(err)
	return t
}

// Roughly estimate a block number by target timestamp (might be off by a lot)
func EstimateTargetBlocknumber(utcTimestamp int64) int64 {
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
	ethValue = new(big.Float).Quo(fbalance, big.NewFloat(1e18))
	return
}

func WeiUintToEth(wei uint64) (ethValue float64) {
	// wei / 10^18
	return float64(wei) / 1e18
}

// Returns bigint wei amount as comma-separated, human-readable string (eg. 1,435,332.71)
func WeiBigIntToEthString(wei *big.Int, decimals int) string {
	return BigFloatToHumanNumberString(WeiToEth(wei), decimals)
}

func BigIntToHumanNumberString(i *big.Int, decimals int) string {
	return BigFloatToHumanNumberString(new(big.Float).SetInt(i), decimals)
}

func BigFloatToHumanNumberString(f *big.Float, decimals int) string {
	output := f.Text('f', decimals)
	dotIndex := strings.Index(output, ".")
	if dotIndex == -1 {
		dotIndex = len(output)
	}
	for outputIndex := dotIndex; outputIndex > 3; {
		outputIndex -= 3
		output = output[:outputIndex] + "," + output[outputIndex:]
	}
	return output
}

func NumberToHumanReadableString(value interface{}, decimals int) string {
	switch v := value.(type) {
	case int:
		i := big.NewInt(int64(v))
		return BigIntToHumanNumberString(i, decimals)
	case big.Int:
		return BigIntToHumanNumberString(&v, decimals)
	case big.Float:
		return BigFloatToHumanNumberString(&v, decimals)
	case string:
		f, ok := new(big.Float).SetString(v)
		if !ok {
			return v
		}
		return BigFloatToHumanNumberString(f, decimals)
	default:
		return "XXX"
	}
}

func printBlock(block *types.Block) {
	t := time.Unix(int64(block.Header().Time), 0).UTC()
	fmt.Printf("%d \t %s \t %d \t tx=%-4d \t gas=%d\n", block.Header().Number, t, block.Header().Time, len(block.Transactions()), block.GasUsed())
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

// // Returns the token amount in the correct unit. Eg. USDC has 6 decimals. 12345678 -> 12.345678
// // Updates from blockchain if not already has
func GetErc20TokensInUnit(numTokens *big.Int, addrDetail AddressDetail) (amount *big.Float, symbol string) {
	tokensFloat := new(big.Float).SetInt(numTokens)

	decimals := int(addrDetail.Decimals)
	divider := math.Pow10(decimals)

	amount = new(big.Float).Quo(tokensFloat, big.NewFloat(divider))
	return amount, addrDetail.Symbol
}

func GetTxSender(tx *types.Transaction) (from common.Address, err error) {
	from, err = types.Sender(types.NewEIP155Signer(tx.ChainId()), tx)
	if err != nil {
		from, err = types.Sender(types.HomesteadSigner{}, tx)
	}
	return from, err
}
