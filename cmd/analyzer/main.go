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
	timespanPtr := flag.String("len", "", "num blocks or timespan (4s, 5m, 1h, ...)")
	outJsonPtr := flag.String("out", "", "filename to store JSON output")
	blockHeightPtr := flag.Int("block", 0, "specific block to check")
	addToDbPtr := flag.Bool("addDb", false, "add to database")
	flag.Parse()

	if len(*datePtr) == 0 && *blockHeightPtr == 0 {
		log.Fatal("Date or block missing, add with -date <yyyy-mm-dd> or -block <blockNum>")
	}

	if *blockHeightPtr == 0 && len(*timespanPtr) == 0 {
		log.Fatal("Timespan missing, -len")
	}

	numBlocks := 0
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
		// No suffix: number of blocks
		numBlocks, _ = strconv.Atoi(*timespanPtr)
	}

	config := ethtools.GetConfig()
	// fmt.Println(config)

	var db *sqlx.DB
	if *addToDbPtr { // try to open DB at the beginning, to fail before the analysis
		db = ethtools.NewDatabaseConnection(config.Database)
	}

	_ = numBlocks
	_ = db

	fmt.Println("Connecting to Ethereum node at", config.EthNode)
	fmt.Println("")
	client, err := ethclient.Dial(config.EthNode)
	ethtools.Perror(err)

	// var result *ethtools.AnalysisResult
	date := *datePtr
	hour := *hourPtr
	min := *minPtr
	sec := 0
	_ = sec
	_ = outJsonPtr

	startBlockHeight := int64(*blockHeightPtr)
	if *blockHeightPtr == 0 { // start at timestamp
		startTime := ethtools.MakeTime(date, hour, min)
		startTimestamp := startTime.Unix()
		fmt.Printf("startTime: %d / %v ... ", startTimestamp, time.Unix(startTimestamp, 0).UTC())
		startBlockHeader, _ := ethtools.GetBlockHeaderAtTimestamp(client, startTimestamp, false)
		startBlockHeight = startBlockHeader.Number.Int64()
	}

	fmt.Printf("startBlock: %d \n", startBlockHeight)

	var endBlockHeight int64
	if numBlocks > 0 {
		endBlockHeight = startBlockHeight + int64(numBlocks)
	} else if numBlocks == 0 && timespanSec > 0 {
		startTime := ethtools.MakeTime(date, hour, min)
		startTimestamp := startTime.Unix()
		endTimestamp := startTimestamp + int64(timespanSec)
		fmt.Printf("endTime:   %d / %v ... ", endTimestamp, time.Unix(endTimestamp, 0).UTC())
		endBlockHeader, _ := ethtools.GetBlockHeaderAtTimestamp(client, endTimestamp, false)
		endBlockHeight = endBlockHeader.Number.Int64() - 1
	} else {
		endBlockHeight = startBlockHeight
	}

	fmt.Printf("  endBlock: %d \n", endBlockHeight)
	fmt.Println("")

	result := ethtools.AnalyzeBlocks(client, db, startBlockHeight, endBlockHeight)
	fmt.Printf("\n===================\n  ANALYSIS RESULT  \n===================\n\n")
	printResult(result)

	// if *addToDbPtr {
	// 	// Add to database (and log execution time)
	// 	fmt.Printf("\nAdding analysis to database...\n")
	// 	timeStartAddToDb := time.Now()
	// 	ethtools.AddAnalysisResultToDatabase(db, client, date, hour, min, sec, timespanSec, result)
	// 	timeNeededAddToDb := time.Since(timeStartAddToDb)
	// 	fmt.Printf("Saved to database (%.3fs)\n", timeNeededAddToDb.Seconds())
	// }

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

func printTopTx(msg string, txList []ethtools.TxStats) {
	fmt.Println(msg)
	for _, v := range txList {
		fmt.Println(v)
	}
}

// Processes a raw result into the export data structure, and prints the stats to stdout
func printResult(result *ethtools.AnalysisResult) {
	fmt.Println("Total blocks:", result.NumBlocks)
	fmt.Println("- without tx:", result.NumBlocksWithoutTx)
	fmt.Println("- gas fee:", ethtools.WeiBigIntToEthString(result.GasFeeTotal, 2), "ETH")
	fmt.Println("")
	fmt.Println("Total transactions:", result.NumTransactions, "/ types:", result.TxTypes)
	fmt.Printf("- failed:         %7d \t %.2f%%\n", result.NumTransactionsFailed, (float64(result.NumTransactionsFailed)/float64(result.NumTransactions))*100)
	fmt.Printf("- with value:     %7d \t %.2f%%\n", result.NumTransactions-result.NumTransactionsWithZeroValue, (float64((result.NumTransactions-result.NumTransactionsWithZeroValue))/float64(result.NumTransactions))*100)
	fmt.Printf("- zero value:     %7d \t %.2f%%\n", result.NumTransactionsWithZeroValue, (float64(result.NumTransactionsWithZeroValue)/float64(result.NumTransactions))*100)
	fmt.Printf("- with data:      %7d \t %.2f%%\n", result.NumTransactionsWithData, (float64(result.NumTransactionsWithData)/float64(result.NumTransactions))*100)
	fmt.Printf("- erc20 transfer: %7d \t %.2f%%\n", result.NumTransactionsErc20Transfer, (float64(result.NumTransactionsErc20Transfer)/float64(result.NumTransactions))*100)
	fmt.Printf("- erc721 transfer:%7d \t %.2f%%\n", result.NumTransactionsErc721Transfer, (float64(result.NumTransactionsErc721Transfer)/float64(result.NumTransactions))*100)
	fmt.Printf("- MEV:         ok %d \t fail %d\n", result.NumMevTransactionsSuccess, result.NumMevTransactionsFailed)
	fmt.Println("")

	fmt.Println("Total addresses:", len(result.Addresses))
	fmt.Println("Total value transferred:", ethtools.WeiBigIntToEthString(result.ValueTotalWei, 2), "ETH")

	printTopTx("\nTop transactions by gas-fee", result.TopTransactions.GasFee)
	printTopTx("\nTop transactions by value", result.TopTransactions.Value)
	printTopTx("\nTop transactions by most data", result.TopTransactions.DataSize)

	addressWithName := func(address string) string {
		detail, _ := ethtools.GetAddressDetail(address, nil)
		return fmt.Sprintf("%s %-28s", detail.Address, detail.Name)
	}

	/* SORT BY TOKEN_TRANSFERS */
	fmt.Println("")
	fmt.Printf("Top %d addresses by erc20 received\n", len(result.TopAddresses.NumErc20TransfersReceived))
	for _, v := range result.TopAddresses.NumErc20TransfersReceived {
		tokensTransferredInUnit, tokenSymbol := ethtools.GetErc20TokensInUnit(v.Erc20TokensReceived, v.AddressDetail)
		tokenAmount := fmt.Sprintf("%s %-5v", formatBigFloat(tokensTransferredInUnit), tokenSymbol)
		fmt.Printf("%s \t %8d erc20-tx \t %8d tx \t %32v \t %s\n", addressWithName(v.Address), v.NumTxErc20TransferReceived, v.NumTxReceivedSuccess, tokenAmount, v.AddressDetail.Type)
	}

	fmt.Println("")
	fmt.Printf("Top %d addresses by erc20 sent\n", len(result.TopAddresses.NumErc20TransfersSent))
	for _, v := range result.TopAddresses.NumErc20TransfersSent {
		fmt.Printf("%s \t %8d erc20-tx \t %8d tx \n", addressWithName(v.Address), v.NumTxErc20TransferSent, v.NumTxSentSuccess)
	}

	if ethtools.GetConfig().HideOutput {
		fmt.Println("End because of config.HideOutput")
		return
	}

	// fmt.Println("")
	// fmt.Printf("Top %d addresses by num-tx-received\n", len(result.TopAddresses.NumTxReceived))
	// for _, v := range result.TopAddresses.NumTxReceived {
	// 	fmt.Printf("%-66v %7d %7d\t%10v ETH\n", addressWithName(v.AddressDetail), v.NumTxReceivedSuccess, v.NumTxSentSuccess, ethtools.WeiToEth(v.ValueReceivedWei).Text('f', 2))
	// }

	// fmt.Println("")
	// fmt.Printf("Top %d addresses by num-tx-sent\n", len(result.TopAddresses.NumTxSent))
	// for _, v := range result.TopAddresses.NumTxSent {
	// 	fmt.Printf("%-66v %7d %7d\t%10v ETH\n", addressWithName(v.AddressDetail), v.NumTxReceivedSuccess, v.NumTxSentSuccess, ethtools.WeiToEth(v.ValueReceivedWei).Text('f', 2))
	// }

	// fmt.Println("")
	// fmt.Printf("Top %d addresses by value-received\n", len(result.TopAddresses.ValueReceived))
	// for _, v := range result.TopAddresses.ValueReceived {
	// 	fmt.Printf("%-66v %7d %7d\t%10v ETH\n", addressWithName(v.AddressDetail), v.NumTxReceivedSuccess, v.NumTxSentSuccess, ethtools.WeiToEth(v.ValueReceivedWei).Text('f', 2))
	// }

	// fmt.Println("")
	// fmt.Printf("Top %d addresses by value-sent\n", len(result.TopAddresses.ValueSent))
	// for _, v := range result.TopAddresses.ValueSent {
	// 	fmt.Printf("%-66v %7d %7d\t%10v ETH\n", addressWithName(v.AddressDetail), v.NumTxReceivedSuccess, v.NumTxSentSuccess, ethtools.WeiToEth(v.ValueSentWei).Text('f', 2))
	// }

	// fmt.Println("")
	// fmt.Printf("Top %d addresses by failed-tx-received\n", len(result.TopAddresses.NumFailedTxReceived))
	// for _, v := range result.TopAddresses.NumFailedTxReceived {
	// 	fmt.Printf("%-66v %7d tx-rec %7d tx-sent \t %4d failed-tx-rec \t %4d failed-tx-sent\n", addressWithName(v.AddressDetail), v.NumTxReceivedSuccess, v.NumTxSentSuccess, v.NumFailedTxReceived, v.NumTxSentFailed)
	// }

	// fmt.Println("")
	// fmt.Printf("Top %d addresses by failed-tx-sent\n", len(result.TopAddresses.NumFailedTxSent))
	// for _, v := range result.TopAddresses.NumFailedTxSent {
	// 	fmt.Printf("%-66v %7d tx-rec %7d tx-sent \t %4d failed-tx-rec \t %4d failed-tx-sent \t %4d mev \t %s ETH gasFee \n", addressWithName(v.AddressDetail), v.NumTxReceivedSuccess, v.NumTxSentSuccess, v.NumFailedTxReceived, v.NumTxSentFailed, v.NumTxMev, ethtools.WeiBigIntToEthString(v.GasFeeTotal, 2))
	// }
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
