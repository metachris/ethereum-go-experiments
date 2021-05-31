package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jmoiron/sqlx"
	"github.com/metachris/ethereum-go-experiments/ethstats"
)

// Raw AnalysisResult extended with a few interesting fields
// type ExportData struct {
// 	Data ethstats.AnalysisResult

// 	TopAddressData TopAddressData
// }

// func NewExportData(result ethstats.AnalysisResult) *ExportData {
// 	topAddressData := TopAddressData{
// 		NumTxReceived:  make([]ethstats.AddressStats, TOP_ADDRESS_COUNT),
// 		NumTxSent:      make([]ethstats.AddressStats, TOP_ADDRESS_COUNT),
// 		ValueReceived:  make([]ethstats.AddressStats, TOP_ADDRESS_COUNT),
// 		ValueSent:      make([]ethstats.AddressStats, TOP_ADDRESS_COUNT),
// 		TokenTransfers: make([]ethstats.AddressStats, TOP_ADDRESS_TOKEN_TRANSFER_COUNT),
// 	}

// 	return &ExportData{
// 		Data:           result,
// 		TopAddressData: topAddressData,
// 	}
// }

func main() {
	timestampMainStart := time.Now() // for measuring execution time

	datePtr := flag.String("date", "", "date (yyyy-mm-dd or -1d)")
	hourPtr := flag.Int("hour", 0, "hour (UTC)")
	minPtr := flag.Int("min", 0, "hour (UTC)")
	lenPtr := flag.String("len", "", "num blocks or timespan (4s, 5m, 1h, ...)")
	outJsonPtr := flag.String("out", "", "filename to store JSON output")
	blockHeightPtr := flag.Int("block", 0, "specific block to check")
	addToDbPtr := flag.Bool("addDb", false, "add to database")
	flag.Parse()

	if len(*datePtr) == 0 && *blockHeightPtr == 0 {
		log.Fatal("Date or block missing, add with -date <yyyy-mm-dd> or -block <blockNum>")
	}

	numBlocks := 0
	timespanSec := 0
	switch {
	case strings.HasSuffix(*lenPtr, "s"):
		timespanSec, _ = strconv.Atoi(strings.TrimSuffix(*lenPtr, "s"))
	case strings.HasSuffix(*lenPtr, "m"):
		timespanSec, _ = strconv.Atoi(strings.TrimSuffix(*lenPtr, "m"))
		timespanSec *= 60
	case strings.HasSuffix(*lenPtr, "h"):
		timespanSec, _ = strconv.Atoi(strings.TrimSuffix(*lenPtr, "h"))
		timespanSec *= 60 * 60
	case strings.HasSuffix(*lenPtr, "d"):
		timespanSec, _ = strconv.Atoi(strings.TrimSuffix(*lenPtr, "d"))
		timespanSec *= 60 * 60 * 24
	case len(*lenPtr) == 0:
		numBlocks = 1
	default:
		// No suffix: number of blocks
		numBlocks, _ = strconv.Atoi(*lenPtr)
	}

	config := ethstats.GetConfig()
	// fmt.Println(config)

	var db *sqlx.DB
	if *addToDbPtr { // try to open DB at the beginning, to fail before the analysis
		db = ethstats.NewDatabaseConnection(config.Database)
	}

	_ = numBlocks
	_ = db

	fmt.Println("Connecting to Ethereum node at", config.EthNode)
	fmt.Println("")
	client, err := ethclient.Dial(config.EthNode)
	ethstats.Perror(err)

	// var result *ethstats.AnalysisResult
	date := *datePtr
	hour := *hourPtr
	min := *minPtr
	sec := 0
	_ = sec
	_ = outJsonPtr

	startBlockHeight := int64(*blockHeightPtr)
	var startTimestamp int64
	var startTime time.Time

	if *blockHeightPtr > 0 { // start at timestamp
		startBlockHeader, err := client.HeaderByNumber(context.Background(), big.NewInt(int64(*blockHeightPtr)))
		ethstats.Perror(err)
		startTimestamp = int64(startBlockHeader.Time)
		startTime = time.Unix(startTimestamp, 0)

	} else {
		// Negative date prefix (-1d, -2m, -1y)
		if strings.HasPrefix(*datePtr, "-") {
			if strings.HasSuffix(*datePtr, "d") {
				t := time.Now().AddDate(0, 0, -1)
				startTime = t.Truncate(24 * time.Hour)
			} else if strings.HasSuffix(*datePtr, "m") {
				t := time.Now().AddDate(0, -1, 0)
				startTime = t.Truncate(24 * time.Hour)
			} else if strings.HasSuffix(*datePtr, "y") {
				t := time.Now().AddDate(-1, 0, 0)
				startTime = t.Truncate(24 * time.Hour)
			} else {
				panic(fmt.Sprintf("Not a valid date: %s", *datePtr))
			}
		} else {
			startTime = ethstats.MakeTime(date, hour, min)
		}

		startTimestamp = startTime.Unix()
		fmt.Printf("startTime: %d / %v ... ", startTimestamp, time.Unix(startTimestamp, 0).UTC())
		startBlockHeader, err := ethstats.GetBlockHeaderAtTimestamp(client, startTimestamp, config.Debug)
		ethstats.Perror(err)
		startBlockHeight = startBlockHeader.Number.Int64()
		date = startTime.Format("2006-01-02")
	}

	fmt.Printf("startBlock: %d \n", startBlockHeight)

	var endBlockHeight int64
	if numBlocks > 0 {
		endBlockHeight = startBlockHeight + int64(numBlocks-1)
	} else if timespanSec > 0 {
		endTimestamp := startTimestamp + int64(timespanSec)
		fmt.Printf("endTime:   %d / %v ... ", endTimestamp, time.Unix(endTimestamp, 0).UTC())
		endBlockHeader, _ := ethstats.GetBlockHeaderAtTimestamp(client, endTimestamp, config.Debug)
		endBlockHeight = endBlockHeader.Number.Int64() - 1
	} else {
		panic("No valid block range")
	}

	if endBlockHeight < startBlockHeight {
		endBlockHeight = startBlockHeight
	}

	fmt.Printf("  endBlock: %d \n", endBlockHeight)
	fmt.Println("")

	result := ethstats.AnalyzeBlocks(client, db, startBlockHeight, endBlockHeight)
	if !ethstats.GetConfig().HideOutput {
		fmt.Printf("\n===================\n  ANALYSIS RESULT  \n===================\n\n")
		printResult(result)
	}

	timeNeeded := time.Since(timestampMainStart)
	fmt.Printf("\nAnalysis finished (%.2fs)\n", timeNeeded.Seconds())

	// result.ToHtml("/tmp/x.html")
	if *addToDbPtr {
		// Add to database
		fmt.Printf("\nSaving to database...\n")
		timeStartAddToDb := time.Now()
		ethstats.AddAnalysisResultToDatabase(db, client, date, hour, min, sec, timespanSec, result)
		timeNeededAddToDb := time.Since(timeStartAddToDb)
		fmt.Printf("Saved to database (%.2fs)\n", timeNeededAddToDb.Seconds())
	}

	// if len(*outJsonPtr) > 0 {
	// 	j, err := json.MarshalIndent(exportData, "", " ")
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	outFn := *outJsonPtr
	// 	_ = ioutil.WriteFile(outFn, j, 0644)
	// 	fmt.Println("Saved to " + *outJsonPtr)
	// }
}

func printH1(msg string) {
	m := strings.Trim(msg, "\n")
	fmt.Println(strings.Repeat("=", utf8.RuneCountInString(m)))
	fmt.Println(m)
	fmt.Println(strings.Repeat("=", utf8.RuneCountInString(m)))
}

func printH2(msg string) {
	m := strings.Trim(msg, "\n")
	fmt.Println(msg)
	fmt.Println(strings.Repeat("-", utf8.RuneCountInString(m)))
}

func printTopTx(msg string, txList []ethstats.TxStats) {
	printH2(msg)
	for _, v := range txList {
		fmt.Println(v)
	}
}

func addressWithName(address string) string {
	detail, _ := ethstats.GetAddressDetail(address, nil)
	return fmt.Sprintf("%s %-28s", detail.Address, detail.Name)
}

func printTopAddr(msg string, list []ethstats.AddressStats, max int) {
	printH2(msg)
	for i, v := range list {
		fmt.Printf("%-66v txInOk: %7d  \t  txOutOk: %7d \t txInFail: %5d \t txOutFail: %5d \t %10v ETH received \t %10v ETH sent \t gasFee %v ETH / for failed: %v ETH \n", addressWithName(v.Address), v.NumTxReceivedSuccess, v.NumTxSentSuccess, v.NumTxReceivedFailed, v.NumTxSentFailed, ethstats.WeiBigIntToEthString(v.ValueReceivedWei, 2), ethstats.WeiBigIntToEthString(v.ValueSentWei, 2), ethstats.WeiBigIntToEthString(v.GasFeeTotal, 2), ethstats.WeiBigIntToEthString(v.GasFeeFailedTx, 2))
		if max > 0 && i == max {
			break
		}
	}
}

// Processes a raw result into the export data structure, and prints the stats to stdout
func printResult(result *ethstats.AnalysisResult) {
	fmt.Println("Total blocks:", ethstats.NumberToHumanReadableString(result.NumBlocks, 0))
	fmt.Println("- without tx:", ethstats.NumberToHumanReadableString(result.NumBlocksWithoutTx, 0))
	fmt.Println("")
	fmt.Println("Total transactions:", ethstats.NumberToHumanReadableString(result.NumTransactions, 0), "\t tx-types:", result.TxTypes)
	fmt.Printf("- failed:         %7s \t %.2f%%\n", ethstats.NumberToHumanReadableString(result.NumTransactionsFailed, 0), (float64(result.NumTransactionsFailed)/float64(result.NumTransactions))*100)
	fmt.Printf("- with value:     %7s \t %.2f%%\n", ethstats.NumberToHumanReadableString(result.NumTransactions-result.NumTransactionsWithZeroValue, 0), (float64((result.NumTransactions-result.NumTransactionsWithZeroValue))/float64(result.NumTransactions))*100)
	fmt.Printf("- zero value:     %7s \t %.2f%%\n", ethstats.NumberToHumanReadableString(result.NumTransactionsWithZeroValue, 0), (float64(result.NumTransactionsWithZeroValue)/float64(result.NumTransactions))*100)
	fmt.Printf("- with data:      %7s \t %.2f%%\n", ethstats.NumberToHumanReadableString(result.NumTransactionsWithData, 0), (float64(result.NumTransactionsWithData)/float64(result.NumTransactions))*100)
	fmt.Printf("- erc20 transfer: %7s \t %.2f%%\n", ethstats.NumberToHumanReadableString(result.NumTransactionsErc20Transfer, 0), (float64(result.NumTransactionsErc20Transfer)/float64(result.NumTransactions))*100)
	fmt.Printf("- erc721 transfer:%7s \t %.2f%%\n", ethstats.NumberToHumanReadableString(result.NumTransactionsErc721Transfer, 0), (float64(result.NumTransactionsErc721Transfer)/float64(result.NumTransactions))*100)
	fmt.Printf("- flashbots:       %s ok, %s failed \n", ethstats.NumberToHumanReadableString(result.NumFlashbotsTransactionsSuccess, 0), ethstats.NumberToHumanReadableString(result.NumFlashbotsTransactionsFailed, 0))
	fmt.Println("")

	fmt.Println("Total addresses:", ethstats.NumberToHumanReadableString(len(result.Addresses), 0))
	fmt.Println("Total value transferred:", ethstats.WeiBigIntToEthString(result.ValueTotalWei, 2), "ETH")
	fmt.Println("Total gas fees:", ethstats.WeiBigIntToEthString(result.GasFeeTotal, 2), "ETH")
	fmt.Println("Gas for failed tx:", ethstats.WeiBigIntToEthString(result.GasFeeFailedTx, 2), "ETH")

	if ethstats.GetConfig().HideOutput {
		fmt.Println("End because of config.HideOutput")
		return
	}

	fmt.Println("")
	printH1("Transactions")
	printTopTx("\nTop transactions by GAS FEE", result.TopTransactions.GasFee)
	printTopTx("\nTop transactions by ETH VALUE", result.TopTransactions.Value)
	printTopTx("\nTop transactions by MOST DATA", result.TopTransactions.DataSize)

	fmt.Println("")
	printH1("\nSmart Contracts")

	printH2("\nERC20: most token tranfers")
	for _, v := range result.TopAddresses["NumTxErc20Transfer"] {
		tokensTransferredInUnit, tokenSymbol := ethstats.GetErc20TokensInUnit(v.Erc20TokensTransferred, v.AddressDetail)
		tokenAmount := fmt.Sprintf("%s %-5v", formatBigFloat(tokensTransferredInUnit), tokenSymbol)
		fmt.Printf("%s \t %8d erc20-tx \t %8d tx \t %32v\n", addressWithName(v.Address), v.NumTxErc20Transfer, v.NumTxReceivedSuccess, tokenAmount)
	}

	// fmt.Println("")
	// fmt.Printf("Top %d addresses by erc20 sent\n", len(result.TopAddresses.NumTxErc20Sent))
	// for _, v := range result.TopAddresses.NumTxErc20Sent {
	// 	fmt.Printf("%s \t %8d erc20-tx \t %8d tx \n", addressWithName(v.Address), v.NumTxErc20Sent, v.NumTxSentSuccess)
	// }

	printH2("\nERC721: most token transfers")
	for _, v := range result.TopAddresses["NumTxErc721Transfer"] {
		fmt.Printf("%-100s \t %8d erc721-tx \t %8d tx \t %s\n", addressWithName(v.Address), v.NumTxErc721Transfer, v.NumTxReceivedSuccess, v.AddressDetail.Type)
	}

	fmt.Println("")
	printH1("\nAddresses")

	interesting := [...]string{"NumTxReceivedSuccess", "NumTxSentSuccess", ethstats.TopAddressValueReceived, ethstats.TopAddressValueSent, "NumTxReceivedFailed", "NumTxSentFailed"}
	for _, key := range interesting {
		fmt.Println("")
		printTopAddr(key, result.TopAddresses[key], 0)
	}
	// printTopAddr("\nTop addresses by number of transactions received (success)", result.TopAddresses.NumTxReceivedSuccess, 0)
	// printTopAddr("\nTop addresses by number of transactions sent (success)", result.TopAddresses.NumTxSentSuccess, 0)
	// printTopAddr("\nTop addresses by value received", result.TopAddresses.ValueReceived, 0)
	// printTopAddr("\nTop addresses by value sent", result.TopAddresses.ValueSent, 0)
	// printTopAddr("\nTop addresses by failed tx received", result.TopAddresses.NumTxReceivedFailed, 0)
	// printTopAddr("\nTop addresses by failed tx sent", result.TopAddresses.NumTxSentFailed, 0)
}

func formatBigFloat(number *big.Float) string {
	output := number.Text('f', 0)
	startOffset := 3
	if number.Cmp(big.NewFloat(0)) == -1 { // negative number, starts with -
		startOffset++
	}
	for outputIndex := len(output); outputIndex > startOffset; {
		outputIndex -= 3
		output = output[:outputIndex] + "," + output[outputIndex:]
	}
	return output
}
