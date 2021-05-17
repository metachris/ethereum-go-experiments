package main

import (
	"bufio"
	"context"
	"ethstats/ethtools"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	// for demo
	erc20 "ethstats/ethtools/contracts/erc20"
	erc721 "ethstats/ethtools/contracts/erc721"

	// for demo

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func getToken(address string, saveToJson bool) {
	if len(address) != 42 {
		log.Fatal("Not a valid address")
	}

	info, err := ethtools.FetchTokenInfo(address)
	if err != nil {
		log.Println(err)
		return
	}

	fmt.Println(info)

	if saveToJson {
		ethtools.AddEthplorerokenToFullJson(&info)
		ethtools.AddToken(info.ToAddressDetail())
	}
}

func loadAddressesFromRawFile(filename string, saveToJson bool, skipIfExists bool) {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	addressMap := ethtools.GetAddressDetailMap(ethtools.DATASET_BOTH)

	// return
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, " ")
		if len(parts) > 0 && len(parts[0]) == 42 {
			address := strings.ToLower(parts[0])
			fmt.Println("Address:", address)

			if skipIfExists {
				if _, ok := addressMap[address]; ok {
					continue // already inside the json
				}
			}

			getToken(address, saveToJson)
			fmt.Println("")
			time.Sleep(ethtools.ETHPLORER_TIMEOUT)
		}
	}
}

func perror(err error) {
	if err != nil {
		panic(err)
	}
}

type AddressType string

const (
	AddressTypeWallet        AddressType = "Wallet"
	AddressTypeErc20         AddressType = "Erc20"
	AddressTypeErc721        AddressType = "Erc721"
	AddressTypeOtherContract AddressType = "OtherContract"
)

func IsContract(address string, client *ethclient.Client) bool {
	addr := common.HexToAddress(address)
	b, err := client.CodeAt(context.Background(), addr, nil)
	perror(err)
	return len(b) > 0
}

func IsErc721(address string, client *ethclient.Client) bool {
	addr := common.HexToAddress(address)
	erc721i, err := erc721.NewErc721(addr, client)
	perror(err)

	// INTERFACEID_ERC165 := [4]byte{1, 255, 201, 167}  // 0x01ffc9a7
	INTERFACEID_ERC721 := [4]byte{128, 172, 88, 205} // 0x80ac58cd
	// INTERFACEID_ERC721_METADATA := [4]byte{91, 94, 19, 159} // 0x5b5e139f
	// INTERFACEID_ERC721_ENUMERABLE := [4]byte{120, 14, 157, 99} // 0x780e9d63
	isErc721, err := erc721i.SupportsInterface(nil, INTERFACEID_ERC721)
	if err == nil && isErc721 {
		return true
	}
	return false
}

func IsErc20(address string, client *ethclient.Client) bool {
	addr := common.HexToAddress(address)
	instance, err := erc20.NewErc20(addr, client)
	perror(err)

	// Needs name
	name, err := instance.Name(nil)
	if err != nil || len(name) == 0 {
		return false
	}

	// Needs symbol
	symbol, err := instance.Symbol(nil)
	if err != nil || len(symbol) == 0 {
		return false
	}

	// Needs decimals
	_, err = instance.Decimals(nil)
	if err != nil {
		return false
	}

	// Needs totalSupply
	_, err = instance.TotalSupply(nil)
	return err == nil
}

func GetAddressType(address string, client *ethclient.Client) AddressType {
	if !IsContract(address, client) {
		return AddressTypeWallet
	}

	if IsErc721(address, client) {
		return AddressTypeErc721
	}

	if IsErc20(address, client) {
		return AddressTypeErc20
	}

	return AddressTypeOtherContract
}

func main() {
	filePtr := flag.String("file", "", "get addresses from specified file")
	addressPtr := flag.String("addr", "", "address")
	listPtr := flag.Bool("list", false, "only list entries")
	listEthplorerPtr := flag.Bool("listEthplorer", false, "only list entries")
	addPtr := flag.Bool("add", false, "add to json")
	fromEthPtr := flag.Bool("eth", false, "get address details from Ethereum node")
	flag.Parse()

	// List addresses
	if *listPtr {
		addressMap := ethtools.GetAddressDetailMap(ethtools.DATASET_BOTH)
		for _, v := range addressMap {
			fmt.Printf("%s \t %-10v \t %-30v %s \t %d\n", v.Address, v.Type, v.Name, v.Symbol, v.Decimals)
		}
		fmt.Printf("%d entries\n", len(addressMap))
		return
	}

	// List Ethplorer Tokens
	if *listEthplorerPtr {
		addressMap := ethtools.GetEthplorerTokenInfolMap()
		for _, v := range addressMap {
			fmt.Printf("%s \t %-30v %s \t %s\n", v.Address, v.Name, v.Symbol, v.Decimals)
		}
		fmt.Printf("%d entries\n", len(addressMap))
		return
	}

	if *filePtr != "" {
		fmt.Println("from file", *filePtr)
		loadAddressesFromRawFile(*filePtr, *addPtr, true)
		return
	}

	if *fromEthPtr {
		// TODO
		fmt.Println("Getting addr info from ethereum node")

		config := ethtools.GetConfig()
		client, err := ethclient.Dial(config.EthNode)
		perror(err)

		// address := "0xa1a55063d81696a7f3e94c84b323c792d2501bea" // wallet
		// address := "0x629a673a8242c2ac4b7b8c5d8735fbeac21a6205" // nft (sorare)
		// address := "0x3B3ee1931Dc30C1957379FAc9aba94D1C48a5405" // nft (fnd)
		// address := "0xC36442b4a4522E871399CD717aBDD847Ab11FE88" // nft (uniswap v3)
		// address := "0x57f1887a8bf19b14fc0df6fd9b2acc9af147ea85" // nft (ens)
		address := "0xdc76a2de1861ea49e8b41a1de1e461085e8f369f" // nft (terravirtua)
		// address := "0xa74476443119A942dE498590Fe1f2454d7D4aC0d" // erc20

		a := GetAddressType(address, client)
		fmt.Println(a)
		// isErc721("0x629a673a8242c2ac4b7b8c5d8735fbeac21a6205", client)
		return

		// instance, err := erc20.NewErc20(tokenAddress, client)
		// instance, err := erc721.NewErc721(address, client)
		// perror(err)

		// name, err := instance.Name(nil)
		// perror(err)
		// fmt.Println(name)

		// uri, err := instance.TokenURI(&bind.CallOpts{}, big.NewInt(1))
		// perror(err)
		// fmt.Println(uri)

		// name, err := instance.Name(&bind.CallOpts{})
		// perror(err)
		// fmt.Println(name)

		// symbol, err := instance.Symbol(&bind.CallOpts{})
		// perror(err)
		// fmt.Println(symbol)

		// decimals, err := instance.Decimals(&bind.CallOpts{})
		// perror(err)
		// fmt.Println(decimals)

	} else {
		// Get from ethplorer
		getToken(*addressPtr, *addPtr)
	}
}
