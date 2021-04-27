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
	valueSent     big.Int
	valueReceived big.Int
}

func main() {
	nodeAddr := "http://95.217.145.161:8545"

	// Start of analysis
	dayStr := "2020-04-27"
	hour := 17 // 0:00 - 0:59
	startTime := makeTime(dayStr, hour, 0)
	startTimestamp := startTime.Unix()
	fmt.Println("startTime:", startTimestamp, "/", startTime)

	// End of analysis
	endTimestamp := startTimestamp + 60*5 // 10 sec
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
	addresses := make(map[string]*AddressInfo)
	txTypes := make(map[uint8]int)
	valueTotal := big.NewInt(0)
	numTransactions := 0
	numTransactionsWithZeroValue := 0
	numTransactionsWithData := 0
	numTransactionsWithTokenTransfer := 0

	currentBlockNumber := new(big.Int).Set(startBlockNumber)

	for {
		// fmt.Println(currentBlockNumber, "fetching...")
		currentBlock, err := client.BlockByNumber(context.Background(), currentBlockNumber)
		if err != nil {
			log.Fatal(err)
		}
		printBlock(currentBlock)
		if currentBlock.Time() > endTimestamp {
			fmt.Println("Block after end. Finished")
			break
		}

		// transactions = append(transactions, currentBlock.Transactions()...)
		numTransactions += len(currentBlock.Transactions())

		// Iterate over all transactions
		for _, tx := range currentBlock.Transactions() {
			// Count total value
			valueTotal = big.NewInt(0).Add(valueTotal, tx.Value())

			// Count number of transactions without value
			if isBigIntZero(tx.Value()) {
				numTransactionsWithZeroValue += 1
			}

			// Count tx types
			txTypes[tx.Type()] += 1

			// Count sender addresses
			// println(tx.To().String())
			if tx.To() != nil {
				toAddrInfo, exists := addresses[tx.To().String()]
				if !exists {
					toAddrInfo = &AddressInfo{address: tx.To().String()}
					addresses[tx.To().String()] = toAddrInfo
				}

				toAddrInfo.numTxReceived += 1
				toAddrInfo.valueReceived = *big.NewInt(0).Add(&toAddrInfo.valueReceived, tx.Value())
			}

			// Process FROM address (see https://goethereumbook.org/en/transaction-query/)
			if msg, err := tx.AsMessage(types.NewEIP155Signer(tx.ChainId())); err == nil {
				fromHex := msg.From().Hex()
				// fmt.Println(fromHex)
				fromAddrInfo, exists := addresses[fromHex]
				if !exists {
					fromAddrInfo = &AddressInfo{address: fromHex}
					addresses[fromHex] = fromAddrInfo
				}

				fromAddrInfo.numTxSent += 1
				fromAddrInfo.valueSent = *big.NewInt(0).Add(&fromAddrInfo.valueSent, tx.Value())
			}

			// Check for token transfer
			data := tx.Data()
			// fmt.Println(data)
			if len(data) > 0 {
				numTransactionsWithData += 1

				if len(data) > 4 {
					methodId := hex.EncodeToString(data[:4])
					// fmt.Println(methodId)
					if methodId == "a9059cbb" || methodId == "23b872dd" {
						numTransactionsWithTokenTransfer += 1
					}
				}
			}
		}

		currentBlockNumber.Add(currentBlockNumber, one)
	}

	print("AFTER THE LOOP")

	// fmt.Println("total transactions:", numTransactions, "- zero value:", numTransactionsWithZeroValue)
	fmt.Println("tx types:", txTypes)
	fmt.Println("total transactions:", numTransactions)
	fmt.Println("- with value:", numTransactions-numTransactionsWithZeroValue)
	fmt.Println("- zero value:", numTransactionsWithZeroValue)
	fmt.Println("- with data: ", numTransactionsWithData)
	fmt.Println("- with token transfer: ", numTransactionsWithTokenTransfer)

	// ETH value transferred
	fmt.Println("total value transferred:", weiToEth(valueTotal).Text('f', 2), "ETH")

	// Address details
	fmt.Println("total addresses:", len(addresses))

	// Create addresses array, for sorting
	_addresses := make([]*AddressInfo, 0, len(addresses))
	for _, k := range addresses {
		_addresses = append(_addresses, k)
	}

	/* SORT BY NUM_TX_RECEIVED */
	fmt.Println("")
	fmt.Println("Top 10 addresses by num-tx-received")
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].numTxReceived > _addresses[j].numTxReceived })
	for _, v := range _addresses[:10] {
		fmt.Printf("%s \t %d \t %d \t %s\n", v.address, v.numTxReceived, v.numTxSent, weiToEth(&v.valueReceived).String())
	}

	/* SORT BY NUM_TX_SENT */
	fmt.Println("")
	fmt.Println("Top 10 addresses by num-tx-sent")
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].numTxSent > _addresses[j].numTxSent })
	for _, v := range _addresses[:10] {
		fmt.Printf("%s \t %d \t %d \t %s\n", v.address, v.numTxReceived, v.numTxSent, weiToEth(&v.valueReceived).String())
	}

	/* SORT BY VALUE_RECEIVED */
	fmt.Println("")
	fmt.Println("Top 10 addresses by value-received")
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].valueReceived.Cmp(&_addresses[j].valueReceived) == 1 })
	for _, v := range _addresses[:10] {
		fmt.Printf("%s \t %d \t %d \t %s\n", v.address, v.numTxReceived, v.numTxSent, weiToEth(&v.valueReceived).String())
	}

	/* SORT BY VALUE_SENT */
	fmt.Println("")
	fmt.Println("Top 10 addresses by value-sent")
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].valueSent.Cmp(&_addresses[j].valueSent) == 1 })
	for _, v := range _addresses[:10] {
		fmt.Printf("%s \t %d \t %d \t %s\n", v.address, v.numTxReceived, v.numTxSent, weiToEth(&v.valueSent).String())
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
