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

	erc20 "ethstats/ethtools/contracts/erc20"
	erc721 "ethstats/ethtools/contracts/erc721"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func getTokenFromEthplorerAndSaveToJson(address string, saveToJson bool) {
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
		ethtools.AddTokenToJson(info.ToAddressDetail())
	}
}

// Helper to check multiple addresses from the run output
func loadAddressesFromRawFile(filename string, saveToJson bool, skipIfExists bool) {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	addressMap := ethtools.GetAddressDetailMap(ethtools.DATASET_BOTH)

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

			getTokenFromEthplorerAndSaveToJson(address, saveToJson)
			fmt.Println("")
			time.Sleep(ethtools.ETHPLORER_TIMEOUT)
		}
	}
}

func IsContract(address string, client *ethclient.Client) bool {
	addr := common.HexToAddress(address)
	b, err := client.CodeAt(context.Background(), addr, nil)
	ethtools.Perror(err)
	return len(b) > 0
}

func IsErc721(address string, client *ethclient.Client) (isErc721 bool, detail ethtools.AddressDetail) {
	detail.Address = address

	addr := common.HexToAddress(address)
	instance, err := erc721.NewErc721(addr, client)
	ethtools.Perror(err)

	isErc721, err = instance.SupportsInterface(nil, ethtools.INTERFACEID_ERC721)
	if err != nil || !isErc721 {
		return false, detail
	}

	detail.Type = ethtools.AddressTypeErc721

	// Check if has metadata
	implementsMetadata, err := instance.SupportsInterface(nil, ethtools.INTERFACEID_ERC721_METADATA)
	if err == nil && implementsMetadata {
		detail.Name, err = instance.Name(nil)
		ethtools.Perror(err)

		detail.Symbol, err = instance.Symbol(nil)
		ethtools.Perror(err)
	}

	return true, detail
}

func IsErc20(address string, client *ethclient.Client) (isErc20 bool, detail ethtools.AddressDetail) {
	detail.Address = address
	addr := common.HexToAddress(address)
	instance, err := erc20.NewErc20(addr, client)
	ethtools.Perror(err)

	detail.Name, err = instance.Name(nil)
	if err != nil || len(detail.Name) == 0 {
		return false, detail
	}

	// Needs symbol
	detail.Symbol, err = instance.Symbol(nil)
	if err != nil || len(detail.Symbol) == 0 {
		return false, detail
	}

	// Needs decimals
	detail.Decimals, err = instance.Decimals(nil)
	if err != nil {
		return false, detail
	}

	// Needs totalSupply
	_, err = instance.TotalSupply(nil)
	if err != nil {
		return false, detail
	}

	detail.Type = ethtools.AddressTypeErc20
	return true, detail
}

func GetAddressType(address string, client *ethclient.Client) ethtools.AddressType {
	if !IsContract(address, client) {
		return ethtools.AddressTypeWallet
	}

	if isErc721, _ := IsErc721(address, client); isErc721 {
		return ethtools.AddressTypeErc721
	}

	if isErc20, _ := IsErc20(address, client); isErc20 {
		return ethtools.AddressTypeErc20
	}

	return ethtools.AddressTypeOtherContract
}

func getAddressDetailFromBlockchain(address string, client *ethclient.Client) ethtools.AddressDetail {
	ret := ethtools.NewAddressDetail(address)

	// If not a contract then return just a wallet detail
	if !IsContract(address, client) {
		ret.Type = ethtools.AddressTypeWallet
		return ret
	}

	if isErc721, detail := IsErc721(address, client); isErc721 {
		return detail
	}

	if isErc20, detail := IsErc20(address, client); isErc20 {
		return detail
	}

	// Return unknown (other) contract
	ret.Type = ethtools.AddressTypeOtherContract
	return ret
}

func main() {
	filePtr := flag.String("file", "", "get addresses from specified file")
	addressPtr := flag.String("addr", "", "address")
	listPtr := flag.Bool("list", false, "only list entries")
	listEthplorerPtr := flag.Bool("listEthplorer", false, "only list entries")
	addPtr := flag.Bool("add", false, "add to json")
	fromEthplorerPtr := flag.Bool("ethplorer", false, "get address details from Ethplorer")
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

	if *fromEthplorerPtr {
		getTokenFromEthplorerAndSaveToJson(*addressPtr, *addPtr)
	} else {
		fmt.Println("Getting address details from Ethereum node...")
		config := ethtools.GetConfig()
		client, err := ethclient.Dial(config.EthNode)
		ethtools.Perror(err)

		// address := "0xa1a55063d81696a7f3e94c84b323c792d2501bea" // wallet
		// address := "0x629a673a8242c2ac4b7b8c5d8735fbeac21a6205" // nft (sorare)
		// address := "0x3B3ee1931Dc30C1957379FAc9aba94D1C48a5405" // nft (fnd)
		// address := "0xC36442b4a4522E871399CD717aBDD847Ab11FE88" // nft (uniswap v3)
		// address := "0x57f1887a8bf19b14fc0df6fd9b2acc9af147ea85" // nft (ens)
		// address := "0xdc76a2de1861ea49e8b41a1de1e461085e8f369f" // nft (terravirtua)
		// address := "0xa74476443119A942dE498590Fe1f2454d7D4aC0d" // erc20

		// a := GetAddressType(address, client)
		a := getAddressDetailFromBlockchain(*addressPtr, client)
		fmt.Println(a)
		return
	}
}
