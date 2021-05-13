package main

import (
	"bufio"
	"ethstats/ethtools"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	token "ethstats/ethtools/contracts/erc20" // for demo

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
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
		if err != nil {
			panic(err)
		}

		tokenAddress := common.HexToAddress("0xa74476443119A942dE498590Fe1f2454d7D4aC0d")
		instance, err := token.NewErc20(tokenAddress, client)
		if err != nil {
			log.Fatal(err)
		}

		name, err := instance.Name(&bind.CallOpts{})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(name)

	} else {
		// Get from ethplorer
		getToken(*addressPtr, *addPtr)
	}
}
