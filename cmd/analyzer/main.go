package main

import (
	"encoding/json"
	"ethtools"
	"fmt"
	"io/ioutil"
	"log"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
)

const TOP_ADDRESS_COUNT = 20
const TOP_ADDRESS_TOKEN_TRANSFER_COUNT = 50

var AddressDetails = ethtools.GetAddressDetailMap()

type ResultCondensed struct {
	AddressesTopNumTxReceived  []*ethtools.AddressInfo
	AddressesTopNumTxSent      []*ethtools.AddressInfo
	AddressesTopValueReceived  []*ethtools.AddressInfo
	AddressesTopValueSent      []*ethtools.AddressInfo
	AddressesTopTokenTransfers []*ethtools.AddressInfo // of a contract address, how many times a transfer/transferFrom method was called
}

func NewResultCondensed() *ResultCondensed {
	return &ResultCondensed{
		AddressesTopNumTxReceived:  make([]*ethtools.AddressInfo, TOP_ADDRESS_COUNT),
		AddressesTopNumTxSent:      make([]*ethtools.AddressInfo, TOP_ADDRESS_COUNT),
		AddressesTopValueReceived:  make([]*ethtools.AddressInfo, TOP_ADDRESS_COUNT),
		AddressesTopValueSent:      make([]*ethtools.AddressInfo, TOP_ADDRESS_COUNT),
		AddressesTopTokenTransfers: make([]*ethtools.AddressInfo, TOP_ADDRESS_TOKEN_TRANSFER_COUNT),
	}
}

func main() {
	nodeAddr := "http://95.217.145.161:8545"
	// nodeAddr := "/server/geth.ipc"

	// Start of analysis (UTC)
	dayStr := "2021-04-29"
	hour := 8
	min := 0
	startTime := ethtools.MakeTime(dayStr, hour, min)
	startTimestamp := startTime.Unix()
	// startTimestamp := time.Now().UTC().Unix() - 180*60
	fmt.Println("startTime:", startTimestamp, "/", time.Unix(startTimestamp, 0).UTC())

	// End of analysis
	// var oneHourInSec int64 = 60 * 60
	endTimestamp := startTimestamp + 4*60
	fmt.Println("endTime:  ", endTimestamp, "/", time.Unix(endTimestamp, 0).UTC())

	fmt.Println("Connecting to Ethereum node at", nodeAddr)
	client, err := ethclient.Dial(nodeAddr)
	if err != nil {
		log.Fatal(err)
	}

	block := ethtools.GetBlockAtTimestamp(client, startTimestamp)
	fmt.Println("Starting block found:", block.Number(), "- time:", block.Time(), "/", time.Unix(int64(block.Time()), 0).UTC())
	result := ethtools.AnalyzeBlocks(client, block.Number().Int64(), endTimestamp)
	printResult(result)

	// analyzeBlocks(client, 12332609, -2)
}

func printResult(result *ethtools.AnalysisResult) {
	// fmt.Println("total transactions:", numTransactions, "- zero value:", numTransactionsWithZeroValue)
	fmt.Println("total blocks:", result.NumBlocks)
	fmt.Println("total transactions:", result.NumTransactions, "/ types:", result.TxTypes)
	fmt.Println("- with value:", result.NumTransactions-result.NumTransactionsWithZeroValue)
	fmt.Println("- zero value:", result.NumTransactionsWithZeroValue)
	fmt.Println("- with data: ", result.NumTransactionsWithData)
	fmt.Printf("- with token transfer: %d (%.2f%%)\n", result.NumTransactionsWithTokenTransfer, (float64(result.NumTransactionsWithTokenTransfer)/float64(result.NumTransactions))*100)

	// ETH value transferred
	ethValue := ethtools.WeiToEth(result.ValueTotalWei)
	result.ValueTotalEth = ethValue.Text('f', 2)
	// check(err)

	fmt.Println("total value transferred:", ethValue.Text('f', 2), "ETH")

	// Address details
	fmt.Println("total addresses:", len(result.Addresses))

	// Create addresses array for sorting
	_addresses := make([]*ethtools.AddressInfo, 0, len(result.Addresses))
	for _, k := range result.Addresses {
		_addresses = append(_addresses, k)
	}

	resultCondensed := NewResultCondensed()

	/* SORT BY NUM_TX_RECEIVED */
	fmt.Println("")
	fmt.Printf("Top %d addresses by num-tx-received\n", TOP_ADDRESS_COUNT)
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].NumTxReceived > _addresses[j].NumTxReceived })
	copy(resultCondensed.AddressesTopNumTxReceived, _addresses[:TOP_ADDRESS_COUNT])
	for _, v := range resultCondensed.AddressesTopNumTxReceived {
		fmt.Printf("%-64v %4d \t %4d \t %s\n", AddressWithName(v.Address), v.NumTxReceived, v.NumTxSent, ethtools.WeiToEth(v.ValueReceived).Text('f', 2))
	}

	/* SORT BY NUM_TX_SENT */
	fmt.Println("")
	fmt.Printf("Top %d addresses by num-tx-sent\n", TOP_ADDRESS_COUNT)
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].NumTxSent > _addresses[j].NumTxSent })
	copy(resultCondensed.AddressesTopNumTxSent, _addresses[:TOP_ADDRESS_COUNT])
	for _, v := range resultCondensed.AddressesTopNumTxSent {
		fmt.Printf("%-64v %4d \t %4d \t %s\n", AddressWithName(v.Address), v.NumTxReceived, v.NumTxSent, ethtools.WeiToEth(v.ValueReceived).Text('f', 2))
	}

	/* SORT BY VALUE_RECEIVED */
	fmt.Println("")
	fmt.Printf("Top %d addresses by value-received\n", TOP_ADDRESS_COUNT)
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].ValueReceived.Cmp(_addresses[j].ValueReceived) == 1 })
	copy(resultCondensed.AddressesTopValueReceived, _addresses[:TOP_ADDRESS_COUNT])
	for _, v := range resultCondensed.AddressesTopValueReceived {
		fmt.Printf("%-64v %4d \t %4d \t %s\n", AddressWithName(v.Address), v.NumTxReceived, v.NumTxSent, ethtools.WeiToEth(v.ValueReceived).Text('f', 2))
	}

	/* SORT BY VALUE_SENT */
	fmt.Println("")
	fmt.Printf("Top %d addresses by value-sent\n", TOP_ADDRESS_COUNT)
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].ValueSent.Cmp(_addresses[j].ValueSent) == 1 })
	copy(resultCondensed.AddressesTopValueSent, _addresses[:TOP_ADDRESS_COUNT])
	for _, v := range resultCondensed.AddressesTopValueSent {
		fmt.Printf("%-64v %4d \t %4d \t %s\n", AddressWithName(v.Address), v.NumTxReceived, v.NumTxSent, ethtools.WeiToEth(v.ValueSent).Text('f', 2))
	}

	/* SORT BY TOKEN_TRANSFERS */
	fmt.Println("")
	fmt.Printf("Top %d addresses by token-transfers\n", TOP_ADDRESS_TOKEN_TRANSFER_COUNT)
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].NumTxTokenTransfer > _addresses[j].NumTxTokenTransfer })
	copy(resultCondensed.AddressesTopTokenTransfers, _addresses[:TOP_ADDRESS_TOKEN_TRANSFER_COUNT])
	for _, v := range resultCondensed.AddressesTopTokenTransfers {
		fmt.Printf("%-64v %4d token transfers \t %4d tx received \t %d \n", AddressWithName(v.Address), v.NumTxTokenTransfer, v.NumTxReceived, v.TokensTransferred)
	}

	j, err := json.MarshalIndent(resultCondensed, "", " ")
	if err != nil {
		log.Fatal(err)
	}

	outFn := "/tmp/test.json"
	_ = ioutil.WriteFile(outFn, j, 0644)
	fmt.Println("Saved to " + outFn)
}

func AddressWithName(address string) string {
	detail, ok := AddressDetails[address]
	if ok {
		return fmt.Sprintf("%s (%s)", address, detail.Name)
	} else {
		return address
	}
}
