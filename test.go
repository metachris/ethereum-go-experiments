package main

import (
	"context"
	"fmt"
	"log"
	"math"
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
	// nodeAddr := "http://localhost:7545"  // ganache

	fmt.Println("Connecting to Ethereum node at", nodeAddr)
	client, err := ethclient.Dial(nodeAddr)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Connected")

	// address := common.HexToAddress("0x160cD5F7ab0DdaBe67A3e393e86F39c5329c4622")
	// fmt.Println(address.Hex())        // 0x71C7656EC7ab88b098defB751B7401B5f6d8976F
	// fmt.Println(address.Hash().Hex()) // 0x00000000000000000000000071c7656ec7ab88b098defb751b7401b5f6d8976f
	// fmt.Println(address.Bytes())      // [113 199 101 110 199 171 136 176 152 222 251 117 27 116 1 181 246 216 151 111]

	// Get balance
	// balance, err := client.BalanceAt(context.Background(), address, nil)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// // fmt.Println(balance) // 25893180161173005034

	// fbalance := new(big.Float)
	// fbalance.SetString(balance.String())
	// ethValue := new(big.Float).Quo(fbalance, big.NewFloat(math.Pow10(18)))
	// fmt.Println(ethValue) // 25.729324269165216041

	// GET BLOCK HEADER
	header, err := client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Latest block", header.Number.String())

	// GET TRANSACTION COUNT
	// count, err := client.TransactionCount(context.Background(), header.Hash())
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println(count)

	// GET FULL BLOCK
	// block, err := client.BlockByNumber(context.Background(), header.Number)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// printBlock(block)

	// Go over last x blocks
	var one = big.NewInt(1)

	var latestBlockHeight = header.Number

	var numBlocks int64 = 3
	var end = big.NewInt(0).Sub(latestBlockHeight, big.NewInt(numBlocks))

	// var transactions []*types.Transaction // list of transactions
	addresses := make(map[string]*AddressInfo)
	txTypes := make(map[uint8]int)
	valueTotal := big.NewInt(0)
	numTransactions := 0
	numTransactionsWithZeroValue := 0
	numTransactionsWithData := 0

	// Collect all blocks and transactions
	for blockHeight := new(big.Int).Set(latestBlockHeight); blockHeight.Cmp(end) > 0; blockHeight.Sub(blockHeight, one) {
		// fmt.Println(blockHeight, "fetching...")
		currentBlock, err := client.BlockByNumber(context.Background(), blockHeight)
		if err != nil {
			log.Fatal(err)
		}
		printBlock(currentBlock)
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

			txData := tx.Data()
			if len(txData) > 0 {
				numTransactionsWithData += 1
			}
		}
	}

	// fmt.Println("total transactions:", numTransactions, "- zero value:", numTransactionsWithZeroValue)
	fmt.Println("tx types:", txTypes)
	fmt.Println("total transactions:", numTransactions)
	fmt.Println("- with value:", numTransactions-numTransactionsWithZeroValue)
	fmt.Println("- zero value:", numTransactionsWithZeroValue)
	fmt.Println("- with data: ", numTransactionsWithData)

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

	/* SORT BY AMOUNT_RECEIVED */
	fmt.Println("")
	fmt.Println("Top 10 addresses by value-received")
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].valueReceived.Cmp(&_addresses[j].valueReceived) == 1 })
	for _, v := range _addresses[:10] {
		fmt.Printf("%s \t %d \t %d \t %s\n", v.address, v.numTxReceived, v.numTxSent, weiToEth(&v.valueReceived).String())
	}
}

func isBigIntZero(n *big.Int) bool {
	return len(n.Bits()) == 0
}

func weiToEth(wei *big.Int) (ethValue *big.Float) {
	// wei / 10^18
	fbalance := new(big.Float)
	fbalance.SetString(wei.String())
	ethValue = new(big.Float).Quo(fbalance, big.NewFloat(math.Pow10(18)))
	return
}

func printBlock(block *types.Block) {
	t := time.Unix(int64(block.Header().Time), 0)
	fmt.Printf("%d \t %s \t tx=%d\n", block.Header().Number, t, len(block.Transactions()))
}
