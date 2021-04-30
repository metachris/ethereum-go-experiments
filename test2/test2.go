package xxx

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {

	nodeAddr := "http://95.217.145.161:8545"
	// nodeAddr := "https://mainnet.infura.io/v3/e03fe41147d548a8a8f55ecad18378fb"

	fmt.Println("Connecting to Ethereum node at", nodeAddr)
	client, err := ethclient.Dial(nodeAddr)
	if err != nil {
		log.Fatal(err)
	}

	// chainConfig := params.ChainConfig{ChainID: big.NewInt(1), EIP155Block: }

	// blockNumber := big.NewInt(5671744)
	// block, err := client.BlockByNumber(context.Background(), blockNumber)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// for _, tx := range block.Transactions() {
	// 	fmt.Println(tx.Hash().Hex()) // 0x5d49fcaa394c97ec8a9c3e7bd9e8388d420fb050a52083ca52ff24b3b65bc9c2
	// 	// fmt.Println(tx.Value().String())    // 10000000000000000
	// 	// fmt.Println(tx.Gas())               // 105000
	// 	// fmt.Println(tx.GasPrice().Uint64()) // 102000000000
	// 	// fmt.Println(tx.Nonce())             // 110644
	// 	// fmt.Println(tx.Data())              // []
	// 	// fmt.Println(tx.To().Hex())          // 0x55fE59D8Ad77035154dDd0AD0388D09Dd4047A8e

	// 	// if msg, err := tx.AsMessage(types.HomesteadSigner{}); err != nil {
	// 	if msg, err := tx.AsMessage(types.NewEIP155Signer(big.NewInt(1))); err == nil {
	// 		fmt.Println("x", msg.From().Hex()) // 0x0fD081e3Bb178dc45c0cb23202069ddA57064258
	// 	}
	// }

	// txHex := "0xb36d6fdbda67ee6eb9316b4c269e103f1fcdd0d145599f90f9acdbd2a0665d95" // ERC20 transfer
	// txHex := "0x08cc6cf2b6ad598458c1fc71026f4c55d9ffa1c91baf1354ee74204406205b8c" // ERC721 transfer
	txHex := "0xc4bbf3633cd5fc760ebe3eab4b2d54cad4220125997bb696cf17f0e5e48466a9" // ERC721 transferFrom
	txHash := common.HexToHash(txHex)
	tx, _, err := client.TransactionByHash(context.Background(), txHash)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(tx.Hash().Hex()) // 0x5d49fcaa394c97ec8a9c3e7bd9e8388d420fb050a52083ca52ff24b3b65bc9c2
	data := tx.Data()
	if len(data) > 4 {
		// fmt.Println(len(data))
		// fmt.Println("data", data)
		methodId := hex.EncodeToString(data[:4])

		var value string
		var targetAddr string

		switch methodId {
		case "a9059cbb": // transfer
			fmt.Println("transfer")
			targetAddr = hex.EncodeToString(data[4:36])
			value = hex.EncodeToString(data[36:])
		case "23b872dd": // transferFrom
			fmt.Println("transferFrom")
			targetAddr = hex.EncodeToString(data[36:68])
			value = hex.EncodeToString(data[68:])
		}

		fmt.Println("methodId  ", methodId)
		fmt.Println("targetAddr", targetAddr)
		fmt.Println("value     ", value)

		n2 := new(big.Int)
		n2.SetString(value, 16)
		fmt.Println(n2.Text(10))
	}
}
