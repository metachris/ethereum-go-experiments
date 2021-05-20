package main

import (
	"ethstats/ethtools"
	"flag"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jmoiron/sqlx"
)

// Raw AnalysisResult extended with a few interesting fields
// type ExportData struct {
// 	Data ethtools.AnalysisResult

// 	TopAddressData TopAddressData
// }

// func NewExportData(result ethtools.AnalysisResult) *ExportData {
// 	topAddressData := TopAddressData{
// 		NumTxReceived:  make([]ethtools.AddressStats, TOP_ADDRESS_COUNT),
// 		NumTxSent:      make([]ethtools.AddressStats, TOP_ADDRESS_COUNT),
// 		ValueReceived:  make([]ethtools.AddressStats, TOP_ADDRESS_COUNT),
// 		ValueSent:      make([]ethtools.AddressStats, TOP_ADDRESS_COUNT),
// 		TokenTransfers: make([]ethtools.AddressStats, TOP_ADDRESS_TOKEN_TRANSFER_COUNT),
// 	}

// 	return &ExportData{
// 		Data:           result,
// 		TopAddressData: topAddressData,
// 	}
// }

func main() {
	datePtr := flag.String("date", "", "date (yyyy-mm-dd)")
	hourPtr := flag.Int("hour", 0, "hour (UTC)")
	minPtr := flag.Int("min", 0, "hour (UTC)")
	timespanPtr := flag.String("len", "", "timespan (4s, 5m, 1h, ...)")
	outJsonPtr := flag.String("out", "", "filename to store JSON output")
	blockNumPtr := flag.Int("block", 0, "specific block to check")
	addToDbPtr := flag.Bool("addDb", false, "add to database")
	flag.Parse()

	if len(*datePtr) == 0 && *blockNumPtr == 0 {
		log.Fatal("Date or block missing, add with -date <yyyy-mm-dd> or -block <blockNum>")
	}

	if *blockNumPtr == 0 && len(*timespanPtr) == 0 {
		log.Fatal("Timespan missing, -len")
	}

	timespanSec := 0
	switch {
	case strings.HasSuffix(*timespanPtr, "s"):
		timespanSec, _ = strconv.Atoi(strings.TrimSuffix(*timespanPtr, "s"))
	case strings.HasSuffix(*timespanPtr, "m"):
		timespanSec, _ = strconv.Atoi(strings.TrimSuffix(*timespanPtr, "m"))
		timespanSec *= 60
	case strings.HasSuffix(*timespanPtr, "h"):
		timespanSec, _ = strconv.Atoi(strings.TrimSuffix(*timespanPtr, "h"))
		timespanSec *= 60 * 60
	case strings.HasSuffix(*timespanPtr, "d"):
		timespanSec, _ = strconv.Atoi(strings.TrimSuffix(*timespanPtr, "d"))
		timespanSec *= 60 * 60 * 24
	default:
		timespanSec, _ = strconv.Atoi(*timespanPtr)

		// If we analyze by starting at a specific block, timespanSec is used for the number of blocks
		if *blockNumPtr > 0 {
			if timespanSec == 0 {
				timespanSec = -1
			} else if timespanSec > 0 {
				timespanSec *= -1
			}
		}
	}

	config := ethtools.GetConfig()
	// fmt.Println(config)

	var db *sqlx.DB
	if *addToDbPtr { // try to open DB at the beginning, to fail before the analysis
		db = ethtools.NewDatabaseConnection(config.Database)
	}

	fmt.Println("Connecting to Ethereum node at", config.EthNode)
	fmt.Println("")
	client, err := ethclient.Dial(config.EthNode)
	if err != nil {
		panic(err)
	}

	var result *ethtools.AnalysisResult
	date := *datePtr
	hour := *hourPtr
	min := *minPtr
	sec := 0
	_ = sec
	_ = outJsonPtr

	if *blockNumPtr > 0 {
		result = ethtools.AnalyzeBlocks(client, int64(*blockNumPtr), int64(timespanSec), db)
		t1 := time.Unix(int64(result.StartBlockTimestamp), 0).UTC()
		date = t1.Format("2006-01-02")
		hour = t1.Hour()
		min = t1.Minute()
		sec = t1.Second()
		timespanSec = int(result.EndBlockTimestamp - result.StartBlockTimestamp)

	} else {
		// Start of analysis (UTC)
		startTime := ethtools.MakeTime(date, hour, min)
		startTimestamp := startTime.Unix()
		fmt.Println("startTime:", startTimestamp, "/", time.Unix(startTimestamp, 0).UTC())

		// End of analysis
		endTimestamp := startTimestamp + int64(timespanSec)
		fmt.Println("endTime:  ", endTimestamp, "/", time.Unix(endTimestamp, 0).UTC())

		fmt.Println("")
		block := ethtools.GetBlockAtTimestamp(client, startTimestamp)
		fmt.Println("Starting block found:", block.Number(), "- time:", block.Time(), "/", time.Unix(int64(block.Time()), 0).UTC())
		fmt.Println("")
		result = ethtools.AnalyzeBlocks(client, block.Number().Int64(), endTimestamp, db)
	}

	fmt.Printf("\n===================\n  ANALYSIS RESULT  \n===================\n\n")
	printResult(result)

	if *addToDbPtr {
		// Add to database (and log execution time)
		fmt.Printf("\nAdding analysis to database...\n")
		timeStartAddToDb := time.Now()
		ethtools.AddAnalysisResultToDatabase(db, client, date, hour, min, sec, timespanSec, result)
		timeNeededAddToDb := time.Since(timeStartAddToDb)
		fmt.Printf("Saved to database (%.3fs)\n", timeNeededAddToDb.Seconds())
	}

	// if len(*outJsonPtr) > 0 {
	// 	j, err := json.MarshalIndent(exportData, "", " ")
	// 	if err != nil {
	// 		panic(err)
	// 	}

	// 	// outFn := "/tmp/test.json"
	// 	outFn := *outJsonPtr
	// 	_ = ioutil.WriteFile(outFn, j, 0644)
	// 	fmt.Println("Saved to " + *outJsonPtr)
	// }

}

// Processes a raw result into the export data structure, and prints the stats to stdout
func printResult(result *ethtools.AnalysisResult) {
	fmt.Println("Total blocks:", result.NumBlocks)
	fmt.Println("- with 0 tx:", result.NumBlocksWithoutTx)
	fmt.Println("- gas fee:", ethtools.WeiBigIntToEthString(result.GasFeeTotal, 2), "ETH")
	fmt.Println("")
	fmt.Println("Total transactions:", result.NumTransactions, "/ types:", result.TxTypes)
	fmt.Printf("- failed:         %7d \t %.2f%%\n", result.NumTransactionsFailed, (float64(result.NumTransactionsFailed)/float64(result.NumTransactions))*100)
	fmt.Printf("- with value:     %7d \t %.2f%%\n", result.NumTransactions-result.NumTransactionsWithZeroValue, (float64((result.NumTransactions-result.NumTransactionsWithZeroValue))/float64(result.NumTransactions))*100)
	fmt.Printf("- zero value:     %7d \t %.2f%%\n", result.NumTransactionsWithZeroValue, (float64(result.NumTransactionsWithZeroValue)/float64(result.NumTransactions))*100)
	fmt.Printf("- with data:      %7d \t %.2f%%\n", result.NumTransactionsWithData, (float64(result.NumTransactionsWithData)/float64(result.NumTransactions))*100)
	fmt.Printf("- token transfer: %7d \t %.2f%%\n", result.NumTransactionsWithTokenTransfer, (float64(result.NumTransactionsWithTokenTransfer)/float64(result.NumTransactions))*100)
	fmt.Printf("- MEV: ok %d \t fail %d\n", result.NumMevTransactionsSuccess, result.NumMevTransactionsFailed)
	fmt.Println("")

	// ETH value transferred
	ethValue := ethtools.WeiToEth(result.ValueTotalWei)
	result.ValueTotalEth = ethValue.Text('f', 2)
	// check(err)

	fmt.Println("Total value transferred:", ethValue.Text('f', 2), "ETH")

	// Address details
	fmt.Println("Total addresses:", len(result.Addresses))

	addressWithName := func(detail ethtools.AddressDetail) string {
		return fmt.Sprintf("%s %-28s", detail.Address, detail.Name)
	}

	/* SORT BY TOKEN_TRANSFERS */
	fmt.Println("")
	fmt.Printf("Top %d addresses by token-transfers\n", len(result.TopAddresses.NumTokenTransfers))
	for _, v := range result.TopAddresses.NumTokenTransfers {
		tokensTransferredInUnit, tokenSymbol := v.TokensTransferredInUnit(nil)
		tokenAmount := fmt.Sprintf("%s %-5v", formatBigFloat(tokensTransferredInUnit), tokenSymbol)
		fmt.Printf("%s \t %8d token transfers \t %8d tx \t %32v \t %s\n", addressWithName(v.AddressDetail), v.NumTxTokenTransfer, v.NumTxReceived, tokenAmount, v.AddressDetail.Type)
	}

	if ethtools.GetConfig().HideOutput {
		fmt.Println("End because of config.HideOutput")
		return
	}

	fmt.Println("")
	fmt.Printf("Top %d addresses by num-tx-received\n", len(result.TopAddresses.NumTxReceived))
	for _, v := range result.TopAddresses.NumTxReceived {
		fmt.Printf("%-66v %7d %7d\t%10v ETH\n", addressWithName(v.AddressDetail), v.NumTxReceived, v.NumTxSent, ethtools.WeiToEth(v.ValueReceivedWei).Text('f', 2))
	}

	fmt.Println("")
	fmt.Printf("Top %d addresses by num-tx-sent\n", len(result.TopAddresses.NumTxSent))
	for _, v := range result.TopAddresses.NumTxSent {
		fmt.Printf("%-66v %7d %7d\t%10v ETH\n", addressWithName(v.AddressDetail), v.NumTxReceived, v.NumTxSent, ethtools.WeiToEth(v.ValueReceivedWei).Text('f', 2))
	}

	fmt.Println("")
	fmt.Printf("Top %d addresses by value-received\n", len(result.TopAddresses.ValueReceived))
	for _, v := range result.TopAddresses.ValueReceived {
		fmt.Printf("%-66v %7d %7d\t%10v ETH\n", addressWithName(v.AddressDetail), v.NumTxReceived, v.NumTxSent, ethtools.WeiToEth(v.ValueReceivedWei).Text('f', 2))
	}

	fmt.Println("")
	fmt.Printf("Top %d addresses by value-sent\n", len(result.TopAddresses.ValueSent))
	for _, v := range result.TopAddresses.ValueSent {
		fmt.Printf("%-66v %7d %7d\t%10v ETH\n", addressWithName(v.AddressDetail), v.NumTxReceived, v.NumTxSent, ethtools.WeiToEth(v.ValueSentWei).Text('f', 2))
	}

	fmt.Println("")
	fmt.Printf("Top %d addresses by failed-tx-received\n", len(result.TopAddresses.NumFailedTxReceived))
	for _, v := range result.TopAddresses.NumFailedTxReceived {
		fmt.Printf("%-66v %7d tx-rec %7d tx-sent \t %4d failed-tx-rec \t %4d failed-tx-sent\n", addressWithName(v.AddressDetail), v.NumTxReceived, v.NumTxSent, v.NumFailedTxReceived, v.NumFailedTxSent)
	}

	fmt.Println("")
	fmt.Printf("Top %d addresses by failed-tx-sent\n", len(result.TopAddresses.NumFailedTxSent))
	for _, v := range result.TopAddresses.NumFailedTxSent {
		fmt.Printf("%-66v %7d tx-rec %7d tx-sent \t %4d failed-tx-rec \t %4d failed-tx-sent \t %4d mev \t %s ETH gasFee \n", addressWithName(v.AddressDetail), v.NumTxReceived, v.NumTxSent, v.NumFailedTxReceived, v.NumFailedTxSent, v.NumTxMev, ethtools.WeiBigIntToEthString(v.GasFeeTotal, 2))
	}
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
