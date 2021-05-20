package main

import (
	"context"
	"ethstats/ethtools"
	"fmt"
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

func TxTest() {
	config := ethtools.GetConfig()
	client, err := ethclient.Dial(config.EthNode)
	ethtools.Perror(err)

	txHash := common.HexToHash("0x3308ca87b00911f3b4aac572f526d41d7786c8b4d845950e83020ac8596353c0")
	tx, _, err := client.TransactionByHash(context.Background(), txHash)
	ethtools.Perror(err)
	// fmt.Println(isPending)

	receipt, err := client.TransactionReceipt(context.Background(), tx.Hash())
	ethtools.Perror(err)
	fee := float64((receipt.GasUsed * tx.GasPrice().Uint64())) / math.Pow10(18)
	fmt.Println(receipt.GasUsed, tx.GasPrice(), fee)
}

type BlockWithTxReceipts struct {
	block      *types.Block
	txReceipts map[common.Hash]*types.Receipt
}

var wg sync.WaitGroup // for waiting until all blocks are written into DB

func GetBlockWithTxReceipts(client *ethclient.Client, height *big.Int) (res BlockWithTxReceipts) {
	fmt.Println("getBlockWithTxReceipts", height)
	defer wg.Done()

	var err error
	if client == nil {
		client, err = ethclient.Dial(ethtools.GetConfig().EthNode)
		ethtools.Perror(err)
	}
	res.block, err = client.BlockByNumber(context.Background(), height)
	ethtools.Perror(err)

	res.txReceipts = make(map[common.Hash]*types.Receipt)
	for _, tx := range res.block.Transactions() {
		receipt, err := client.TransactionReceipt(context.Background(), tx.Hash())
		ethtools.Perror(err)
		res.txReceipts[tx.Hash()] = receipt
	}

	return res
}

func TestBlockWithTxReceipts() {
	// test speed difference:
	// (a) get block and receipt for all tx in one function
	// (b) do a, but for several blocks at once (with one and multiple geth connections)

	blockHeight := big.NewInt(12465297)
	numBlocks := 3

	timeStart := time.Now()
	for i := 0; i < numBlocks; i++ {
		wg.Add(1)
		go GetBlockWithTxReceipts(nil, blockHeight)
		blockHeight = new(big.Int).Add(blockHeight, common.Big1)
	}

	wg.Wait()
	timeNeeded := time.Since(timeStart)
	fmt.Printf("Finished in (%.3fs)\n", timeNeeded.Seconds())
}

func main() {
	// resetPtr := flag.Bool("reset", false, "reset database")
	// flag.Parse()
	TestBlockWithTxReceipts()
}
