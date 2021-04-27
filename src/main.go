package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type AddressInfo struct {
	Address       string
	NumTxSent     int
	NumTxReceived int
	ValueSent     *big.Int
	ValueReceived *big.Int
}

func NewAddressInfo(address string) *AddressInfo {
	return &AddressInfo{
		Address:       address,
		ValueSent:     big.NewInt(0),
		ValueReceived: big.NewInt(0),
	}
}

type AnalysisResult struct {
	StartBlockNumber *big.Int
	EndBlockNumber   *big.Int

	Addresses                        map[string]*AddressInfo
	TxTypes                          map[uint8]int
	ValueTotal                       *big.Int
	NumBlocks                        int
	NumTransactions                  int
	NumTransactionsWithZeroValue     int
	NumTransactionsWithData          int
	NumTransactionsWithTokenTransfer int
}

func NewResult() *AnalysisResult {
	addresses := make(map[string]*AddressInfo)
	txTypes := make(map[uint8]int)

	return &AnalysisResult{
		Addresses:        addresses,
		TxTypes:          txTypes,
		ValueTotal:       big.NewInt(0),
		StartBlockNumber: big.NewInt(0),
		EndBlockNumber:   big.NewInt(0),
	}
}

func main() {
	nodeAddr := "http://95.217.145.161:8545"
	// nodeAddr := "/server/geth.ipc"

	// Start of analysis (UTC)
	dayStr := "2021-04-27"
	hour := 17
	min := 49
	startTime := makeTime(dayStr, hour, min)
	startTimestamp := startTime.Unix()
	fmt.Println("startTime:", startTimestamp, "/", startTime)

	// End of analysis
	endTimestamp := startTimestamp + 30
	fmt.Println("endTime:  ", endTimestamp, "/", time.Unix(endTimestamp, 0).UTC())

	fmt.Println("Connecting to Ethereum node at", nodeAddr)
	client, err := ethclient.Dial(nodeAddr)
	if err != nil {
		log.Fatal(err)
	}

	block := getBlockAtTimestamp(client, startTimestamp)
	fmt.Println("Starting block found:", block.Number(), "- time:", block.Time(), "/", time.Unix(int64(block.Time()), 0))

	analyzeBlocks(client, block.Number(), uint64(endTimestamp))
}

// Analyze blocks starting at specific block number, until a certain target timestamp
func analyzeBlocks(client *ethclient.Client, startBlockNumber *big.Int, endTimestamp uint64) {
	result := NewResult()

	currentBlockNumber := new(big.Int).Set(startBlockNumber)

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

		result.EndBlockNumber = currentBlockNumber
		result.NumBlocks += 1
		result.NumTransactions += len(currentBlock.Transactions())

		// Iterate over all transactions
		for _, tx := range currentBlock.Transactions() {
			// Count total value
			// result.valueTotal = big.NewInt(0).Add(result.valueTotal, tx.Value())

			// Count number of transactions without value
			if isBigIntZero(tx.Value()) {
				result.NumTransactionsWithZeroValue += 1
			}

			// Count tx types
			result.TxTypes[tx.Type()] += 1

			// Count sender addresses
			// println(tx.To().String())
			if tx.To() != nil {
				toAddrInfo, exists := result.Addresses[tx.To().String()]
				if !exists {
					// toAddrInfo = &AddressInfo{address: tx.To().String()}
					toAddrInfo = NewAddressInfo(tx.To().String())
					result.Addresses[tx.To().String()] = toAddrInfo
				}

				toAddrInfo.NumTxReceived += 1
				toAddrInfo.ValueReceived = big.NewInt(0).Add(toAddrInfo.ValueReceived, tx.Value())
			}

			// Process FROM address (see https://goethereumbook.org/en/transaction-query/)
			if msg, err := tx.AsMessage(types.NewEIP155Signer(tx.ChainId())); err == nil {
				fromHex := msg.From().Hex()
				// fmt.Println(fromHex)
				fromAddrInfo, exists := result.Addresses[fromHex]
				if !exists {
					// fromAddrInfo = &AddressInfo{address: fromHex}
					fromAddrInfo = NewAddressInfo(fromHex)
					result.Addresses[fromHex] = fromAddrInfo
				}

				fromAddrInfo.NumTxSent += 1
				fromAddrInfo.ValueSent = big.NewInt(0).Add(fromAddrInfo.ValueSent, tx.Value())
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
	fmt.Println("total value transferred:", weiToEth(result.ValueTotal).Text('f', 2), "ETH")

	// Address details
	fmt.Println("total addresses:", len(result.Addresses))

	// Create addresses array, for sorting
	_addresses := make([]*AddressInfo, 0, len(result.Addresses))
	for _, k := range result.Addresses {
		_addresses = append(_addresses, k)
	}

	/* SORT BY NUM_TX_RECEIVED */
	fmt.Println("")
	fmt.Println("Top 10 addresses by num-tx-received")
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].NumTxReceived > _addresses[j].NumTxReceived })
	for _, v := range _addresses[:10] {
		fmt.Printf("%s \t %d \t %d \t %s\n", v.Address, v.NumTxReceived, v.NumTxSent, weiToEth(v.ValueReceived).String())
	}

	/* SORT BY NUM_TX_SENT */
	fmt.Println("")
	fmt.Println("Top 10 addresses by num-tx-sent")
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].NumTxSent > _addresses[j].NumTxSent })
	for _, v := range _addresses[:10] {
		fmt.Printf("%s \t %d \t %d \t %s\n", v.Address, v.NumTxReceived, v.NumTxSent, weiToEth(v.ValueReceived).String())
	}

	/* SORT BY VALUE_RECEIVED */
	fmt.Println("")
	fmt.Println("Top 10 addresses by value-received")
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].ValueReceived.Cmp(_addresses[j].ValueReceived) == 1 })
	for _, v := range _addresses[:10] {
		fmt.Printf("%s \t %d \t %d \t %s\n", v.Address, v.NumTxReceived, v.NumTxSent, weiToEth(v.ValueReceived).String())
	}

	/* SORT BY VALUE_SENT */
	fmt.Println("")
	fmt.Println("Top 10 addresses by value-sent")
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].ValueSent.Cmp(_addresses[j].ValueSent) == 1 })
	for _, v := range _addresses[:10] {
		fmt.Printf("%s \t %d \t %d \t %s\n", v.Address, v.NumTxReceived, v.NumTxSent, weiToEth(v.ValueSent).String())
	}

	j, err := json.Marshal(result)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(j))

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
