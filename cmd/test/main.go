package main

import (
	"context"
	"ethstats/ethtools"
	"fmt"
	"math"
	"sort"

	"github.com/ethereum/go-ethereum/common"
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
	fmt.Println(tx.Hash().Hex())
}

// func GetBlockWithTxReceipts(client *ethclient.Client, wg *sync.WaitGroup, height *big.Int) (res ethtools.BlockWithTxReceipts) {
// 	fmt.Println("getBlockWithTxReceipts", height)
// 	defer wg.Done()

// 	var err error
// 	if client == nil {
// 		client, err = ethclient.Dial(ethtools.GetConfig().EthNode)
// 		ethtools.Perror(err)
// 	}
// 	res.block, err = client.BlockByNumber(context.Background(), height)
// 	ethtools.Perror(err)

// 	res.txReceipts = make(map[common.Hash]*types.Receipt)
// 	for _, tx := range res.block.Transactions() {
// 		receipt, err := client.TransactionReceipt(context.Background(), tx.Hash())
// 		ethtools.Perror(err)
// 		res.txReceipts[tx.Hash()] = receipt
// 	}

// 	fmt.Println(height, len(res.txReceipts))
// 	return res
// }

// func TestBlockWithTxReceipts() {
// 	// test speed difference:
// 	// (a) get block and receipt for all tx in one function
// 	// (b) do a, but for several blocks at once (with one and multiple geth connections)

// 	blockHeight := big.NewInt(12464297)
// 	numBlocks := 10

// 	var wg sync.WaitGroup // for waiting until all blocks are written into DB
// 	// client, _ := ethclient.Dial(ethtools.GetConfig().EthNode)

// 	timeStart := time.Now()
// 	for i := 0; i < numBlocks; i++ {
// 		wg.Add(1)
// 		go GetBlockWithTxReceipts(nil, &wg, blockHeight)
// 		blockHeight = new(big.Int).Add(blockHeight, common.Big1)
// 	}

// 	wg.Wait()
// 	timeNeeded := time.Since(timeStart)
// 	fmt.Printf("Finished in (%.3fs)\n", timeNeeded.Seconds())
// }

const maxEntries = 5

type Test struct {
	lists map[string][]int
}

func (test *Test) Add1(key string, val int) { // test lists are already sorted
	if _, ok := test.lists[key]; !ok {
		test.lists[key] = make([]int, 5)
	}

	if test.lists[key][len(test.lists[key])-1] < val {
		test.lists[key][len(test.lists[key])-1] = val
	}

	sort.Sort(sort.Reverse(sort.IntSlice((test.lists[key]))))
	fmt.Println("3", test.lists[key])

	// test.lists[key][0] = val
	// list, _ := test.lists[key]

}

func main() {
	// resetPtr := flag.Bool("reset", false, "reset database")
	// flag.Parse()
	// TestBlockWithTxReceipts()

	TxTest()

	// fmt.Println(math.Pow10(0))
	// t := Test{
	// 	lists: make(map[string][]int),
	// }

	// for i := 0; i < 10; i++ {
	// 	t.Add1("foo", i)
	// }

	// fmt.Println(t.lists["foo"])
}
