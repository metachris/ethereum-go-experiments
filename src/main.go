package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

const TOP_ADDRESS_COUNT = 20
const TOP_ADDRESS_TOKEN_TRANSFER_COUNT = 50

type AddressInfo struct {
	Address           string
	NumTxSent         int
	NumTxReceived     int
	NumTokenTransfers int
	valueSent         *big.Int
	ValueSentEth      string
	valueReceived     *big.Int
	ValueReceivedEth  string
}

func NewAddressInfo(address string) *AddressInfo {
	return &AddressInfo{
		Address:       address,
		valueSent:     new(big.Int),
		valueReceived: new(big.Int),
	}
}

type AnalysisResult struct {
	StartBlockNumber int64
	EndBlockNumber   int64

	addresses     map[string]*AddressInfo
	TxTypes       map[uint8]int
	valueTotalWei *big.Int
	ValueTotalEth string

	NumBlocks                        int
	NumTransactions                  int
	NumTransactionsWithZeroValue     int
	NumTransactionsWithData          int
	NumTransactionsWithTokenTransfer int

	AddressesTopNumTxReceived  []*AddressInfo
	AddressesTopNumTxSent      []*AddressInfo
	AddressesTopValueReceived  []*AddressInfo
	AddressesTopValueSent      []*AddressInfo
	AddressesTopTokenTransfers []*AddressInfo
}

func NewResult() *AnalysisResult {
	return &AnalysisResult{
		valueTotalWei:              new(big.Int),
		addresses:                  make(map[string]*AddressInfo),
		TxTypes:                    make(map[uint8]int),
		AddressesTopNumTxReceived:  make([]*AddressInfo, TOP_ADDRESS_COUNT),
		AddressesTopNumTxSent:      make([]*AddressInfo, TOP_ADDRESS_COUNT),
		AddressesTopValueReceived:  make([]*AddressInfo, TOP_ADDRESS_COUNT),
		AddressesTopValueSent:      make([]*AddressInfo, TOP_ADDRESS_COUNT),
		AddressesTopTokenTransfers: make([]*AddressInfo, TOP_ADDRESS_TOKEN_TRANSFER_COUNT),
	}
}

func main() {
	nodeAddr := "http://95.217.145.161:8545"
	// nodeAddr := "/server/geth.ipc"

	// Start of analysis (UTC)
	// dayStr := "2021-04-27"
	// hour := 17
	// min := 49
	// startTime := makeTime(dayStr, hour, min)
	// startTimestamp := startTime.Unix()
	startTimestamp := time.Now().UTC().Unix() - 60*60
	fmt.Println("startTime:", startTimestamp, "/", time.Unix(startTimestamp, 0).UTC())

	// End of analysis
	endTimestamp := startTimestamp + 60*5
	fmt.Println("endTime:  ", endTimestamp, "/", time.Unix(endTimestamp, 0).UTC())

	fmt.Println("Connecting to Ethereum node at", nodeAddr)
	client, err := ethclient.Dial(nodeAddr)
	if err != nil {
		log.Fatal(err)
	}

	block := getBlockAtTimestamp(client, startTimestamp)
	fmt.Println("Starting block found:", block.Number(), "- time:", block.Time(), "/", time.Unix(int64(block.Time()), 0).UTC())

	analyzeBlocks(client, block.Number().Int64(), uint64(endTimestamp))
}

// Analyze blocks starting at specific block number, until a certain target timestamp
func analyzeBlocks(client *ethclient.Client, startBlockNumber int64, endTimestamp uint64) {
	result := NewResult()
	result.StartBlockNumber = startBlockNumber

	currentBlockNumber := big.NewInt(startBlockNumber)

	for {
		// fmt.Println(currentBlockNumber, "fetching...")
		currentBlock, err := client.BlockByNumber(context.Background(), currentBlockNumber)
		if err != nil {
			log.Fatal(err)
		}

		if currentBlock.Time() > endTimestamp {
			break
		}

		printBlock(currentBlock)

		result.EndBlockNumber = currentBlockNumber.Int64()
		result.NumBlocks += 1
		result.NumTransactions += len(currentBlock.Transactions())

		// Iterate over all transactions
		for _, tx := range currentBlock.Transactions() {
			// Count total value
			result.valueTotalWei = result.valueTotalWei.Add(result.valueTotalWei, tx.Value())

			// Count number of transactions without value
			if isBigIntZero(tx.Value()) {
				result.NumTransactionsWithZeroValue += 1
			}

			// Count tx types
			result.TxTypes[tx.Type()] += 1

			// Count sender addresses
			// println(tx.To().String())
			if tx.To() != nil {
				toAddrInfo, exists := result.addresses[tx.To().String()]
				if !exists {
					// toAddrInfo = &AddressInfo{address: tx.To().String()}
					toAddrInfo = NewAddressInfo(tx.To().String())
					result.addresses[tx.To().String()] = toAddrInfo
				}

				toAddrInfo.NumTxReceived += 1
				// toAddrInfo.ValueReceived = new(BigInt).Add(toAddrInfo.ValueReceived, tx.Value())
				toAddrInfo.valueReceived = new(big.Int).Add(toAddrInfo.valueReceived, tx.Value())
				toAddrInfo.ValueReceivedEth = weiToEth(toAddrInfo.valueReceived).Text('f', 2)
			}

			// Process FROM address (see https://goethereumbook.org/en/transaction-query/)
			if msg, err := tx.AsMessage(types.NewEIP155Signer(tx.ChainId())); err == nil {
				fromHex := msg.From().Hex()
				// fmt.Println(fromHex)
				fromAddrInfo, exists := result.addresses[fromHex]
				if !exists {
					// fromAddrInfo = &AddressInfo{address: fromHex}
					fromAddrInfo = NewAddressInfo(fromHex)
					result.addresses[fromHex] = fromAddrInfo
				}

				fromAddrInfo.NumTxSent += 1
				fromAddrInfo.valueSent = big.NewInt(0).Add(fromAddrInfo.valueSent, tx.Value())
				fromAddrInfo.ValueSentEth = weiToEth(fromAddrInfo.valueSent).Text('f', 2)
			}

			// Check for token transfer
			data := tx.Data()
			// fmt.Println(data)
			if len(data) > 0 {
				result.NumTransactionsWithData += 1

				if len(data) > 4 {
					methodId := hex.EncodeToString(data[:4])
					// fmt.Println(methodId)
					if methodId == "a9059cbb" || methodId == "23b872dd" {
						result.NumTransactionsWithTokenTransfer += 1

						toAddrInfo, exists := result.addresses[tx.To().String()]
						if !exists {
							// toAddrInfo = &AddressInfo{address: tx.To().String()}
							toAddrInfo = NewAddressInfo(tx.To().String())
							result.addresses[tx.To().String()] = toAddrInfo
						}
						toAddrInfo.NumTokenTransfers += 1
					}
				}
			}
		}

		currentBlockNumber.Add(currentBlockNumber, one)
	}

	// fmt.Println("total transactions:", numTransactions, "- zero value:", numTransactionsWithZeroValue)
	fmt.Println("total blocks:", result.NumBlocks)
	fmt.Println("total transactions:", result.NumTransactions, "/ types:", result.TxTypes)
	fmt.Println("- with value:", result.NumTransactions-result.NumTransactionsWithZeroValue)
	fmt.Println("- zero value:", result.NumTransactionsWithZeroValue)
	fmt.Println("- with data: ", result.NumTransactionsWithData)
	fmt.Printf("- with token transfer: %d (%.2f%%)\n", result.NumTransactionsWithTokenTransfer, (float64(result.NumTransactionsWithTokenTransfer)/float64(result.NumTransactions))*100)

	// ETH value transferred
	ethValue := weiToEth(result.valueTotalWei)
	result.ValueTotalEth = ethValue.Text('f', 2)
	// check(err)

	fmt.Println("total value transferred:", ethValue.Text('f', 2), "ETH")

	// Address details
	fmt.Println("total addresses:", len(result.addresses))

	// Create addresses array for sorting
	_addresses := make([]*AddressInfo, 0, len(result.addresses))
	for _, k := range result.addresses {
		_addresses = append(_addresses, k)
	}

	/* SORT BY NUM_TX_RECEIVED */
	fmt.Println("")
	fmt.Printf("Top %d addresses by num-tx-received\n", TOP_ADDRESS_COUNT)
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].NumTxReceived > _addresses[j].NumTxReceived })
	copy(result.AddressesTopNumTxReceived, _addresses[:TOP_ADDRESS_COUNT])
	for _, v := range result.AddressesTopNumTxReceived {
		fmt.Printf("%s \t %d \t %d \t %s\n", v.Address, v.NumTxReceived, v.NumTxSent, weiToEth(v.valueReceived).String())
	}

	/* SORT BY NUM_TX_SENT */
	fmt.Println("")
	fmt.Printf("Top %d addresses by num-tx-sent\n", TOP_ADDRESS_COUNT)
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].NumTxSent > _addresses[j].NumTxSent })
	copy(result.AddressesTopNumTxSent, _addresses[:TOP_ADDRESS_COUNT])
	for _, v := range result.AddressesTopNumTxSent {
		fmt.Printf("%s \t %d \t %d \t %s\n", v.Address, v.NumTxReceived, v.NumTxSent, weiToEth(v.valueReceived).String())
	}

	/* SORT BY VALUE_RECEIVED */
	fmt.Println("")
	fmt.Printf("Top %d addresses by value-received\n", TOP_ADDRESS_COUNT)
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].valueReceived.Cmp(_addresses[j].valueReceived) == 1 })
	copy(result.AddressesTopValueReceived, _addresses[:TOP_ADDRESS_COUNT])
	for _, v := range result.AddressesTopValueReceived {
		fmt.Printf("%s \t %d \t %d \t %s\n", v.Address, v.NumTxReceived, v.NumTxSent, weiToEth(v.valueReceived).String())
	}

	/* SORT BY VALUE_SENT */
	fmt.Println("")
	fmt.Printf("Top %d addresses by value-sent\n", TOP_ADDRESS_COUNT)
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].valueSent.Cmp(_addresses[j].valueSent) == 1 })
	copy(result.AddressesTopValueSent, _addresses[:TOP_ADDRESS_COUNT])
	for _, v := range result.AddressesTopValueSent {
		fmt.Printf("%s \t %d \t %d \t %s\n", v.Address, v.NumTxReceived, v.NumTxSent, weiToEth(v.valueSent).String())
	}

	/* SORT BY TOKEN_TRANSFERS */
	fmt.Println("")
	fmt.Printf("Top %d addresses by token-transfers\n", TOP_ADDRESS_TOKEN_TRANSFER_COUNT)
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].NumTokenTransfers > _addresses[j].NumTokenTransfers })
	copy(result.AddressesTopTokenTransfers, _addresses[:TOP_ADDRESS_TOKEN_TRANSFER_COUNT])
	for _, v := range result.AddressesTopTokenTransfers {
		fmt.Printf("%s \t %d token transfers \t %d tx received\n", v.Address, v.NumTokenTransfers, v.NumTxReceived)
	}

	j, err := json.MarshalIndent(result, "", " ")
	if err != nil {
		log.Fatal(err)
	}
	// err := ioutil.WriteFile("/tmp/dat1", d1, 0644)
	_ = ioutil.WriteFile("test.json", j, 0644)
	fmt.Println("Saved to test.json")
	// fmt.Println(string(j))

}

// getBlockAtTimestamp searches for a block that is within 30 seconds of the target timestamp.
// Guesses a block number, downloads it, and narrows down time gap if needed until block is found.
//
// TODO: After block found, check one block further, and use that if even better match
func getBlockAtTimestamp(client *ethclient.Client, targetTimestamp int64) *types.Block {
	currentBlockNumber := getTargetBlocknumber(targetTimestamp)

	for {
		// fmt.Println("Checking block:", currentBlockNumber)
		blockNumber := big.NewInt(currentBlockNumber)
		block, err := client.BlockByNumber(context.Background(), blockNumber)
		if err != nil {
			log.Fatal(err)
		}

		secDiff := int64(block.Time()) - targetTimestamp
		fmt.Println("finding target block | height:", currentBlockNumber, "time:", block.Time(), "diff:", secDiff)
		if Abs(secDiff) < 30 {
			return block
		}

		// Not found. Check for better block
		blockDiff := secDiff / 13
		currentBlockNumber -= blockDiff
	}
}
