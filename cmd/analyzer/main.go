package main

import (
	"encoding/json"
	"ethstats/ethtools"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jmoiron/sqlx"
)

const NODE_URI = "http://95.217.145.161:8545"

// const NODE_URI = "/server/geth.ipc"

// const NODE_URI = "https://mainnet.infura.io/v3/e03fe41147d548a8a8f55ecad18378fb"

const TOP_ADDRESS_COUNT = 30
const TOP_ADDRESS_TOKEN_TRANSFER_COUNT = 100

type TopAddressData struct {
	NumTxReceived  []ethtools.AddressInfo
	NumTxSent      []ethtools.AddressInfo
	ValueReceived  []ethtools.AddressInfo
	ValueSent      []ethtools.AddressInfo
	TokenTransfers []ethtools.AddressInfo // of a contract address, how many times a transfer/transferFrom method was called
}

// Raw AnalysisResult extended with a few interesting fields
type ExportData struct {
	Data ethtools.AnalysisResult

	TopAddressData TopAddressData
}

func NewExportData(result ethtools.AnalysisResult) *ExportData {
	topAddressData := TopAddressData{
		NumTxReceived:  make([]ethtools.AddressInfo, TOP_ADDRESS_COUNT),
		NumTxSent:      make([]ethtools.AddressInfo, TOP_ADDRESS_COUNT),
		ValueReceived:  make([]ethtools.AddressInfo, TOP_ADDRESS_COUNT),
		ValueSent:      make([]ethtools.AddressInfo, TOP_ADDRESS_COUNT),
		TokenTransfers: make([]ethtools.AddressInfo, TOP_ADDRESS_TOKEN_TRANSFER_COUNT),
	}

	return &ExportData{
		Data:           result,
		TopAddressData: topAddressData,
	}
}

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

	var db *sqlx.DB
	if *addToDbPtr { // try to open DB at the beginning, to fail before the analysis
		db = ethtools.OpenDatabase(ethtools.GetDbConfig())
	}

	fmt.Println("Connecting to Ethereum node at", NODE_URI)
	fmt.Println("")
	client, err := ethclient.Dial(NODE_URI)
	if err != nil {
		log.Fatal(err)
	}

	var result *ethtools.AnalysisResult
	date := *datePtr
	hour := *hourPtr
	min := *minPtr
	sec := 0

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

	fmt.Println("")
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

	if *addToDbPtr {
		ethtools.AddAnalysisResultToDatabase(db, date, hour, min, sec, timespanSec, result)
		fmt.Println("Saved to database ✔")
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
	_addresses := make([]ethtools.AddressInfo, 0, len(result.Addresses))
	for _, k := range result.Addresses {
		_addresses = append(_addresses, *k)
	}

	exportData := NewExportData(*result)

	/* SORT BY NUM_TX_RECEIVED */
	fmt.Println("")
	fmt.Printf("Top %d addresses by num-tx-received\n", TOP_ADDRESS_COUNT)
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].NumTxReceived > _addresses[j].NumTxReceived })
	copy(exportData.TopAddressData.NumTxReceived, _addresses[:TOP_ADDRESS_COUNT])
	for _, v := range exportData.TopAddressData.NumTxReceived {
		fmt.Printf("%-66v %7d %7d\t%10v ETH\n", AddressWithName(v.Address), v.NumTxReceived, v.NumTxSent, ethtools.WeiToEth(v.ValueReceivedWei).Text('f', 2))
	}

	/* SORT BY NUM_TX_SENT */
	fmt.Println("")
	fmt.Printf("Top %d addresses by num-tx-sent\n", TOP_ADDRESS_COUNT)
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].NumTxSent > _addresses[j].NumTxSent })
	copy(exportData.TopAddressData.NumTxSent, _addresses[:TOP_ADDRESS_COUNT])
	for _, v := range exportData.TopAddressData.NumTxSent {
		fmt.Printf("%-66v %7d %7d\t%10v ETH\n", AddressWithName(v.Address), v.NumTxReceived, v.NumTxSent, ethtools.WeiToEth(v.ValueReceivedWei).Text('f', 2))
	}

	/* SORT BY VALUE_RECEIVED */
	fmt.Println("")
	fmt.Printf("Top %d addresses by value-received\n", TOP_ADDRESS_COUNT)
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].ValueReceivedWei.Cmp(_addresses[j].ValueReceivedWei) == 1 })
	copy(exportData.TopAddressData.ValueReceived, _addresses[:TOP_ADDRESS_COUNT])
	for _, v := range exportData.TopAddressData.ValueReceived {
		fmt.Printf("%-66v %7d %7d\t%10v ETH\n", AddressWithName(v.Address), v.NumTxReceived, v.NumTxSent, ethtools.WeiToEth(v.ValueReceivedWei).Text('f', 2))
	}

	/* SORT BY VALUE_SENT */
	fmt.Println("")
	fmt.Printf("Top %d addresses by value-sent\n", TOP_ADDRESS_COUNT)
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].ValueSentWei.Cmp(_addresses[j].ValueSentWei) == 1 })
	copy(exportData.TopAddressData.ValueSent, _addresses[:TOP_ADDRESS_COUNT])
	for _, v := range exportData.TopAddressData.ValueSent {
		fmt.Printf("%-66v %7d %7d\t%10v ETH\n", AddressWithName(v.Address), v.NumTxReceived, v.NumTxSent, ethtools.WeiToEth(v.ValueSentWei).Text('f', 2))
	}

	/* SORT BY TOKEN_TRANSFERS */
	fmt.Println("")
	fmt.Printf("Top %d addresses by token-transfers\n", TOP_ADDRESS_TOKEN_TRANSFER_COUNT)
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].NumTxTokenTransfer > _addresses[j].NumTxTokenTransfer })
	copy(exportData.TopAddressData.TokenTransfers, _addresses[:TOP_ADDRESS_TOKEN_TRANSFER_COUNT])
	for _, v := range exportData.TopAddressData.TokenTransfers {
		if v.NumTxReceived == 0 {
			continue
		}
		tokensTransferredInUnit, tokenSymbol, found := ethtools.TokenAmountToUnit(v.TokensTransferred, v.Address)
		tokenAmount := fmt.Sprintf("%s %-5v", formatInt(tokensTransferredInUnit), tokenSymbol)
		if !found {
			tokenAmount = fmt.Sprintf("%s ?    ", tokensTransferredInUnit.Text('f', 0))
		}
		fmt.Printf("%-66v %8d token transfers \t %8d tx \t %32v\n", AddressWithName(v.Address), v.NumTxTokenTransfer, v.NumTxReceived, tokenAmount)
	}

	return exportData
}

func AddressWithName(address string) string {
	detail, ok := ethtools.AllAddressesFromJson[strings.ToLower(address)]
	if ok {
		return fmt.Sprintf("%s %s", address, detail.Name)
	} else {
		return address
	}
}

func formatInt(number *big.Float) string {
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
