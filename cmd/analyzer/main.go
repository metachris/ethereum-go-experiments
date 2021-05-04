package main

import (
	"encoding/json"
	"ethtools"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
)

// const NODE_URI = "http://95.217.145.161:8545"

const NODE_URI = "/server/geth.ipc"

const TOP_ADDRESS_COUNT = 30
const TOP_ADDRESS_TOKEN_TRANSFER_COUNT = 100

var AddressDetails = ethtools.GetAddressDetailMap(ethtools.DATASET_BOTH)

type TopAddressData struct {
	NumTxReceived  []*ethtools.AddressInfo
	NumTxSent      []*ethtools.AddressInfo
	ValueReceived  []*ethtools.AddressInfo
	ValueSent      []*ethtools.AddressInfo
	TokenTransfers []*ethtools.AddressInfo // of a contract address, how many times a transfer/transferFrom method was called
}

// Raw AnalysisResult extended with a few interesting fields
type ExportData struct {
	Data *ethtools.AnalysisResult

	TopAddressData TopAddressData
}

func NewExportData(result ethtools.AnalysisResult) *ExportData {
	topAddressData := TopAddressData{
		NumTxReceived:  make([]*ethtools.AddressInfo, TOP_ADDRESS_COUNT),
		NumTxSent:      make([]*ethtools.AddressInfo, TOP_ADDRESS_COUNT),
		ValueReceived:  make([]*ethtools.AddressInfo, TOP_ADDRESS_COUNT),
		ValueSent:      make([]*ethtools.AddressInfo, TOP_ADDRESS_COUNT),
		TokenTransfers: make([]*ethtools.AddressInfo, TOP_ADDRESS_TOKEN_TRANSFER_COUNT),
	}

	return &ExportData{
		Data:           &result,
		TopAddressData: topAddressData,
	}
}

func main() {
	datePtr := flag.String("date", "", "date (yyyy-mm-dd)")
	hourPtr := flag.Int("hour", 0, "hour (UTC)")
	minPtr := flag.Int("min", 0, "hour (UTC)")
	timespanPtr := flag.String("len", "", "timespan (4s, 5m, 1h, ...)")
	outJsonPtr := flag.String("out", "", "filename to store JSON output")
	flag.Parse()

	if len(*datePtr) == 0 {
		log.Fatal("Date missing, add with -date <yyyy-mm-dd>")
	}

	if len(*timespanPtr) == 0 {
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
	}

	// Start of analysis (UTC)
	// dayStr := "2021-04-29"
	// hour := 16
	// min := 0
	startTime := ethtools.MakeTime(*datePtr, *hourPtr, *minPtr)
	startTimestamp := startTime.Unix()
	// startTimestamp := time.Now().UTC().Unix() - 180*60
	fmt.Println("startTime:", startTimestamp, "/", time.Unix(startTimestamp, 0).UTC())

	// End of analysis
	// var oneHourInSec int64 = 60 * 60
	// endTimestamp := startTimestamp + 60*5
	endTimestamp := startTimestamp + int64(timespanSec)
	fmt.Println("endTime:  ", endTimestamp, "/", time.Unix(endTimestamp, 0).UTC())

	fmt.Println("Connecting to Ethereum node at", NODE_URI)
	fmt.Println("")
	client, err := ethclient.Dial(NODE_URI)
	if err != nil {
		log.Fatal(err)
	}

	block := ethtools.GetBlockAtTimestamp(client, startTimestamp)
	fmt.Println("\nStarting block found:", block.Number(), "- time:", block.Time(), "/", time.Unix(int64(block.Time()), 0).UTC())
	result := ethtools.AnalyzeBlocks(client, block.Number().Int64(), endTimestamp)
	// analyzeBlocks(client, 12332609, -2)

	exportData := processResultAndPrint(result)

	if len(*outJsonPtr) > 0 {
		j, err := json.MarshalIndent(exportData, "", " ")
		if err != nil {
			panic(err)
		}

		// outFn := "/tmp/test.json"
		outFn := *outJsonPtr
		_ = ioutil.WriteFile(outFn, j, 0644)
		fmt.Println("Saved to " + *outJsonPtr)
	}
}

// Processes a raw result into the export data structure, and prints the stats to stdout
func processResultAndPrint(result *ethtools.AnalysisResult) *ExportData {
	fmt.Println("Total blocks:", result.NumBlocks)
	fmt.Println("Total transactions:", result.NumTransactions, "/ types:", result.TxTypes)
	fmt.Println("- with value:", result.NumTransactions-result.NumTransactionsWithZeroValue)
	fmt.Println("- zero value:", result.NumTransactionsWithZeroValue)
	fmt.Println("- with data: ", result.NumTransactionsWithData)
	fmt.Printf("- with token transfer: %d (%.2f%%)\n", result.NumTransactionsWithTokenTransfer, (float64(result.NumTransactionsWithTokenTransfer)/float64(result.NumTransactions))*100)

	// ETH value transferred
	ethValue := ethtools.WeiToEth(result.ValueTotalWei)
	result.ValueTotalEth = ethValue.Text('f', 2)
	// check(err)

	fmt.Println("Total value transferred:", ethValue.Text('f', 2), "ETH")

	// Address details
	fmt.Println("Total addresses:", len(result.Addresses))

	// Create addresses array for sorting
	_addresses := make([]*ethtools.AddressInfo, 0, len(result.Addresses))
	for _, k := range result.Addresses {
		_addresses = append(_addresses, k)
	}

	exportData := NewExportData(*result)

	/* SORT BY NUM_TX_RECEIVED */
	fmt.Println("")
	fmt.Printf("Top %d addresses by num-tx-received\n", TOP_ADDRESS_COUNT)
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].NumTxReceived > _addresses[j].NumTxReceived })
	copy(exportData.TopAddressData.NumTxReceived, _addresses[:TOP_ADDRESS_COUNT])
	for _, v := range exportData.TopAddressData.NumTxReceived {
		fmt.Printf("%-66v %7d %7d\t%10v ETH\n", AddressWithName(v.Address), v.NumTxReceived, v.NumTxSent, ethtools.WeiToEth(v.ValueReceived).Text('f', 2))
	}

	/* SORT BY NUM_TX_SENT */
	fmt.Println("")
	fmt.Printf("Top %d addresses by num-tx-sent\n", TOP_ADDRESS_COUNT)
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].NumTxSent > _addresses[j].NumTxSent })
	copy(exportData.TopAddressData.NumTxSent, _addresses[:TOP_ADDRESS_COUNT])
	for _, v := range exportData.TopAddressData.NumTxSent {
		fmt.Printf("%-66v %7d %7d\t%10v ETH\n", AddressWithName(v.Address), v.NumTxReceived, v.NumTxSent, ethtools.WeiToEth(v.ValueReceived).Text('f', 2))
	}

	/* SORT BY VALUE_RECEIVED */
	fmt.Println("")
	fmt.Printf("Top %d addresses by value-received\n", TOP_ADDRESS_COUNT)
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].ValueReceived.Cmp(_addresses[j].ValueReceived) == 1 })
	copy(exportData.TopAddressData.ValueReceived, _addresses[:TOP_ADDRESS_COUNT])
	for _, v := range exportData.TopAddressData.ValueReceived {
		fmt.Printf("%-66v %7d %7d\t%10v ETH\n", AddressWithName(v.Address), v.NumTxReceived, v.NumTxSent, ethtools.WeiToEth(v.ValueReceived).Text('f', 2))
	}

	/* SORT BY VALUE_SENT */
	fmt.Println("")
	fmt.Printf("Top %d addresses by value-sent\n", TOP_ADDRESS_COUNT)
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].ValueSent.Cmp(_addresses[j].ValueSent) == 1 })
	copy(exportData.TopAddressData.ValueSent, _addresses[:TOP_ADDRESS_COUNT])
	for _, v := range exportData.TopAddressData.ValueSent {
		fmt.Printf("%-66v %7d %7d\t%10v ETH\n", AddressWithName(v.Address), v.NumTxReceived, v.NumTxSent, ethtools.WeiToEth(v.ValueSent).Text('f', 2))
	}

	/* SORT BY TOKEN_TRANSFERS */
	fmt.Println("")
	fmt.Printf("Top %d addresses by token-transfers\n", TOP_ADDRESS_TOKEN_TRANSFER_COUNT)
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].NumTxTokenTransfer > _addresses[j].NumTxTokenTransfer })
	copy(exportData.TopAddressData.TokenTransfers, _addresses[:TOP_ADDRESS_TOKEN_TRANSFER_COUNT])
	for _, v := range exportData.TopAddressData.TokenTransfers {
		fmt.Printf("%-66v %8d token transfers \t %8d tx \t %30v \n", AddressWithName(v.Address), v.NumTxTokenTransfer, v.NumTxReceived, NumTokensWithDecimals(v.TokensTransferred, v.Address))
	}

	return exportData
}

func AddressWithName(address string) string {
	detail, ok := AddressDetails[strings.ToLower(address)]
	if ok {
		return fmt.Sprintf("%s %s", address, detail.Name)
	} else {
		return address
	}
}

func NumTokensWithDecimals(numTokens uint64, address string) string {
	// fmt.Println(address)
	detail, ok := AddressDetails[strings.ToLower(address)]
	if ok {
		decimals := uint64(detail.Decimals)
		divider := float64(1)
		if decimals > 0 {
			divider = math.Pow(10, float64(decimals))
		}
		return fmt.Sprintf("%.2f %-4v", float64(numTokens)/divider, detail.Symbol)
	} else {
		return fmt.Sprintf("%d ?   ", numTokens)
	}
}