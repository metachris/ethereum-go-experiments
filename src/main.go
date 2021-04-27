package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type AddressInfo struct {
	address       string
	numTxSent     int
	numTxReceived int
	valueSent     *big.Int
	valueReceived *big.Int
}

func NewAddressInfo(address string) *AddressInfo {
	return &AddressInfo{
		address:       address,
		valueSent:     big.NewInt(0),
		valueReceived: big.NewInt(0),
	}
}

type AnalysisResult struct {
	startBlockNumber *big.Int
	endBlockNumber   *big.Int

	addresses                        map[string]*AddressInfo
	txTypes                          map[uint8]int
	valueTotal                       *big.Int
	numBlocks                        int
	numTransactions                  int
	numTransactionsWithZeroValue     int
	numTransactionsWithData          int
	numTransactionsWithTokenTransfer int
}

func NewResult() *AnalysisResult {
	addresses := make(map[string]*AddressInfo)
	txTypes := make(map[uint8]int)

	return &AnalysisResult{
		addresses:        addresses,
		txTypes:          txTypes,
		valueTotal:       big.NewInt(0),
		startBlockNumber: big.NewInt(0),
		endBlockNumber:   big.NewInt(0),
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

		result.endBlockNumber = currentBlockNumber
		result.numBlocks += 1
		result.numTransactions += len(currentBlock.Transactions())

		// Iterate over all transactions
		for _, tx := range currentBlock.Transactions() {
			// Count total value
			// result.valueTotal = big.NewInt(0).Add(result.valueTotal, tx.Value())

			// Count number of transactions without value
			if isBigIntZero(tx.Value()) {
				result.numTransactionsWithZeroValue += 1
			}

			// Count tx types
			result.txTypes[tx.Type()] += 1

			// Count sender addresses
			// println(tx.To().String())
			if tx.To() != nil {
				toAddrInfo, exists := result.addresses[tx.To().String()]
				if !exists {
					// toAddrInfo = &AddressInfo{address: tx.To().String()}
					toAddrInfo = NewAddressInfo(tx.To().String())
					result.addresses[tx.To().String()] = toAddrInfo
				}

				toAddrInfo.numTxReceived += 1
				toAddrInfo.valueReceived = big.NewInt(0).Add(toAddrInfo.valueReceived, tx.Value())
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

				fromAddrInfo.numTxSent += 1
				fromAddrInfo.valueSent = big.NewInt(0).Add(fromAddrInfo.valueSent, tx.Value())
			}

			// Check for token transfer
			data := tx.Data()
			// fmt.Println(data)
			if len(data) > 0 {
				result.numTransactionsWithData += 1

				if len(data) > 4 {
					methodId := hex.EncodeToString(data[:4])
					// fmt.Println(methodId)
					if methodId == "a9059cbb" || methodId == "23b872dd" {
						result.numTransactionsWithTokenTransfer += 1
					}
				}
			}
		}

		currentBlockNumber.Add(currentBlockNumber, one)
	}

	// fmt.Println("total transactions:", numTransactions, "- zero value:", numTransactionsWithZeroValue)
	fmt.Println("total blocks:", result.numBlocks)
	fmt.Println("total transactions:", result.numTransactions, "/ types:", result.txTypes)
	fmt.Println("- with value:", result.numTransactions-result.numTransactionsWithZeroValue)
	fmt.Println("- zero value:", result.numTransactionsWithZeroValue)
	fmt.Println("- with data: ", result.numTransactionsWithData)
	fmt.Printf("- with token transfer: %d (%.2f%%)\n", result.numTransactionsWithTokenTransfer, (float64(result.numTransactionsWithTokenTransfer)/float64(result.numTransactions))*100)

	// ETH value transferred
	// fmt.Println("total value transferred:", weiToEth(result.valueTotal).Text('f', 2), "ETH")

	// Address details
	fmt.Println("total addresses:", len(result.addresses))

	// Create addresses array, for sorting
	_addresses := make([]*AddressInfo, 0, len(result.addresses))
	for _, k := range result.addresses {
		_addresses = append(_addresses, k)
	}

	/* SORT BY NUM_TX_RECEIVED */
	fmt.Println("")
	fmt.Println("Top 10 addresses by num-tx-received")
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].numTxReceived > _addresses[j].numTxReceived })
	for _, v := range _addresses[:10] {
		fmt.Printf("%s \t %d \t %d \t %s\n", v.address, v.numTxReceived, v.numTxSent, weiToEth(v.valueReceived).String())
	}

	/* SORT BY NUM_TX_SENT */
	fmt.Println("")
	fmt.Println("Top 10 addresses by num-tx-sent")
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].numTxSent > _addresses[j].numTxSent })
	for _, v := range _addresses[:10] {
		fmt.Printf("%s \t %d \t %d \t %s\n", v.address, v.numTxReceived, v.numTxSent, weiToEth(v.valueReceived).String())
	}

	/* SORT BY VALUE_RECEIVED */
	fmt.Println("")
	fmt.Println("Top 10 addresses by value-received")
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].valueReceived.Cmp(_addresses[j].valueReceived) == 1 })
	for _, v := range _addresses[:10] {
		fmt.Printf("%s \t %d \t %d \t %s\n", v.address, v.numTxReceived, v.numTxSent, weiToEth(v.valueReceived).String())
	}

	/* SORT BY VALUE_SENT */
	fmt.Println("")
	fmt.Println("Top 10 addresses by value-sent")
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].valueSent.Cmp(_addresses[j].valueSent) == 1 })
	for _, v := range _addresses[:10] {
		fmt.Printf("%s \t %d \t %d \t %s\n", v.address, v.numTxReceived, v.numTxSent, weiToEth(v.valueSent).String())
	}
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
