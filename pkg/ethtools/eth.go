package ethtools

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type AddressInfo struct {
	Address            string
	NumTxSent          int
	NumTxReceived      int
	NumTxTokenTransfer int
	ValueSent          *big.Int
	ValueSentEth       string
	ValueReceived      *big.Int
	ValueReceivedEth   string
	tokensTransferred  *big.Int
	TokensTransferred  uint64
}

func NewAddressInfo(address string) *AddressInfo {
	return &AddressInfo{
		Address:           address,
		ValueSent:         new(big.Int),
		ValueReceived:     new(big.Int),
		tokensTransferred: new(big.Int),
	}
}

type AnalysisResult struct {
	StartBlockNumber    int64
	StartBlockTimestamp uint64
	EndBlockNumber      int64
	EndBlockTimestamp   uint64

	Addresses     map[string]*AddressInfo
	TxTypes       map[uint8]int
	ValueTotalWei *big.Int
	ValueTotalEth string

	NumBlocks                        int
	NumTransactions                  int
	NumTransactionsWithZeroValue     int
	NumTransactionsWithData          int
	NumTransactionsWithTokenTransfer int
}

func NewResult() *AnalysisResult {
	return &AnalysisResult{
		ValueTotalWei: new(big.Int),
		Addresses:     make(map[string]*AddressInfo),
		TxTypes:       make(map[uint8]int),
	}
}

// Analyze blocks starting at specific block number, until a certain target timestamp
func AnalyzeBlocks(client *ethclient.Client, startBlockNumber int64, endTimestamp int64) *AnalysisResult {
	result := NewResult()
	result.StartBlockNumber = startBlockNumber

	currentBlockNumber := big.NewInt(startBlockNumber)
	numBlocksProcessed := 0

	for {
		// fmt.Println(currentBlockNumber, "fetching...")
		currentBlock, err := client.BlockByNumber(context.Background(), currentBlockNumber)
		if err != nil {
			log.Fatal(err)
		}

		printBlock(currentBlock)

		if endTimestamp > -1 && currentBlock.Time() > uint64(endTimestamp) {
			fmt.Println("- 👆 block after end time, skipping and done")
			break
		}

		if result.StartBlockTimestamp == 0 {
			result.StartBlockTimestamp = currentBlock.Time()
		}

		result.EndBlockNumber = currentBlockNumber.Int64()
		result.EndBlockTimestamp = currentBlock.Time()

		result.NumBlocks += 1
		result.NumTransactions += len(currentBlock.Transactions())

		// Iterate over all transactions
		for _, tx := range currentBlock.Transactions() {
			// Count total value
			result.ValueTotalWei = result.ValueTotalWei.Add(result.ValueTotalWei, tx.Value())

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
					toAddrInfo = NewAddressInfo(tx.To().String())
					result.Addresses[tx.To().String()] = toAddrInfo
				}

				toAddrInfo.NumTxReceived += 1
				toAddrInfo.ValueReceived = new(big.Int).Add(toAddrInfo.ValueReceived, tx.Value())
				toAddrInfo.ValueReceivedEth = WeiToEth(toAddrInfo.ValueReceived).Text('f', 2)
			}

			// Process FROM address (see https://goethereumbook.org/en/transaction-query/)
			if msg, err := tx.AsMessage(types.NewEIP155Signer(tx.ChainId())); err == nil {
				fromHex := msg.From().Hex()
				// fmt.Println(fromHex)
				fromAddrInfo, exists := result.Addresses[fromHex]
				if !exists {
					fromAddrInfo = NewAddressInfo(fromHex)
					result.Addresses[fromHex] = fromAddrInfo
				}

				fromAddrInfo.NumTxSent += 1
				fromAddrInfo.ValueSent = big.NewInt(0).Add(fromAddrInfo.ValueSent, tx.Value())
				fromAddrInfo.ValueSentEth = WeiToEth(fromAddrInfo.ValueSent).Text('f', 2)
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

						if tx.To() != nil {
							toAddrInfo, exists := result.Addresses[tx.To().String()]
							if !exists {
								toAddrInfo = NewAddressInfo(tx.To().String())
								result.Addresses[tx.To().String()] = toAddrInfo
							}
							toAddrInfo.NumTxTokenTransfer += 1

							// Calculate and store the number of tokens transferred
							var value string
							switch methodId {
							case "a9059cbb": // transfer
								// targetAddr = hex.EncodeToString(data[4:36])
								value = hex.EncodeToString(data[36:])
							case "23b872dd": // transferFrom
								// targetAddr = hex.EncodeToString(data[36:68])
								value = hex.EncodeToString(data[68:])
							}
							valBigInt := new(big.Int)
							valBigInt.SetString(value, 16)
							toAddrInfo.tokensTransferred = new(big.Int).Add(toAddrInfo.tokensTransferred, valBigInt)
							toAddrInfo.TokensTransferred = toAddrInfo.tokensTransferred.Uint64()
							// if toAddrInfo.Address == "0xdAC17F958D2ee523a2206206994597C13D831ec7" {
							// 	fmt.Println("tt", valBigInt.String(), toAddrInfo.tokensTransferred.String(), a)
							// }
						}
					}
				}
			}
		}

		currentBlockNumber.Add(currentBlockNumber, One)
		numBlocksProcessed += 1
		if endTimestamp < 0 && numBlocksProcessed*-1 == int(endTimestamp) {
			break
		}
	}

	return result
}

// GetBlockAtTimestamp searches for a block that is within 30 seconds of the target timestamp.
// Guesses a block number, downloads it, and narrows down time gap if needed until block is found.
func GetBlockAtTimestamp(client *ethclient.Client, targetTimestamp int64) *types.Block {
	currentBlockNumber := getTargetBlocknumber(targetTimestamp)
	// TODO: check that blockNumber <= latestHeight
	var isNarrowingDownFromBelow = false

	for {
		// fmt.Println("Checking block:", currentBlockNumber)
		blockNumber := big.NewInt(currentBlockNumber)
		block, err := client.BlockByNumber(context.Background(), blockNumber)
		if err != nil {
			log.Println("Error fetching block at height", currentBlockNumber)
			log.Fatal(err)
		}

		secDiff := int64(block.Time()) - targetTimestamp
		fmt.Println("finding target block:", currentBlockNumber, "- time:", block.Time(), "- diff:", secDiff)
		if Abs(secDiff) < 60 {
			if secDiff < 0 {
				// still before wanted startTime. Increase by 1 from here...
				isNarrowingDownFromBelow = true
				currentBlockNumber += 1
				continue
			}

			// Only return if coming block-by-block from below, making sure to take first block after target time
			if isNarrowingDownFromBelow {
				return block
			} else {
				currentBlockNumber -= 1
				continue
			}
		}

		// Try for better block in big steps
		blockDiff := secDiff / 13 // average block time is 13 sec
		currentBlockNumber -= blockDiff
	}
}
