package main

// import (
// 	"bufio"
// 	"github.com/metachris/ethereum-go-experiments/ethtools"
// 	"flag"
// 	"fmt"
// 	"log"
// 	"os"
// 	"strings"
// 	"time"

// 	"github.com/ethereum/go-ethereum/ethclient"
// )

// func getTokenFromEthplorerAndSaveToJson(address string, saveToJson bool) {
// 	if len(address) != 42 {
// 		log.Fatal("Not a valid address")
// 	}

// 	info, err := ethstats.FetchTokenInfo(address)
// 	if err != nil {
// 		log.Println(err)
// 		return
// 	}

// 	fmt.Println(info)

// 	if saveToJson {
// 		ethstats.AddEthplorerokenToFullJson(&info)
// 		ethstats.AddTokenToJson(info.ToAddressDetail())
// 	}
// }

// // Helper to check multiple addresses from the run output
// func loadAddressesFromRawFile(filename string, saveToJson bool, skipIfExists bool) {
// 	file, err := os.Open(filename)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer file.Close()

// 	addressMap := ethstats.GetAddressDetailMap(ethstats.DATASET_BOTH)

// 	scanner := bufio.NewScanner(file)
// 	for scanner.Scan() {
// 		line := scanner.Text()
// 		parts := strings.Split(line, " ")
// 		if len(parts) > 0 && len(parts[0]) == 42 {
// 			address := strings.ToLower(parts[0])
// 			fmt.Println("Address:", address)

// 			if skipIfExists {
// 				if _, ok := addressMap[address]; ok {
// 					continue // already inside the json
// 				}
// 			}

// 			getTokenFromEthplorerAndSaveToJson(address, saveToJson)
// 			fmt.Println("")
// 			time.Sleep(ethstats.ETHPLORER_TIMEOUT)
// 		}
// 	}
// }

// func main() {
// 	filePtr := flag.String("file", "", "get addresses from specified file")
// 	addressPtr := flag.String("addr", "", "address")
// 	listPtr := flag.Bool("list", false, "only list entries")
// 	listEthplorerPtr := flag.Bool("listEthplorer", false, "only list entries")
// 	addPtr := flag.Bool("add", false, "add to json")
// 	fromEthplorerPtr := flag.Bool("ethplorer", false, "get address details from Ethplorer")
// 	flag.Parse()

// 	// List addresses
// 	if *listPtr {
// 		addressMap := ethstats.GetAddressDetailMap(ethstats.DATASET_BOTH)
// 		for _, v := range addressMap {
// 			fmt.Printf("%s \t %-10v \t %-30v %s \t %d\n", v.Address, v.Type, v.Name, v.Symbol, v.Decimals)
// 		}
// 		fmt.Printf("%d entries\n", len(addressMap))
// 		return
// 	}

// 	// List Ethplorer Tokens
// 	if *listEthplorerPtr {
// 		addressMap := ethstats.GetEthplorerTokenInfolMap()
// 		for _, v := range addressMap {
// 			fmt.Printf("%s \t %-30v %s \t %s\n", v.Address, v.Name, v.Symbol, v.Decimals)
// 		}
// 		fmt.Printf("%d entries\n", len(addressMap))
// 		return
// 	}

// 	if *filePtr != "" {
// 		fmt.Println("from file", *filePtr)
// 		loadAddressesFromRawFile(*filePtr, *addPtr, true)
// 		return
// 	}

// 	if *fromEthplorerPtr {
// 		getTokenFromEthplorerAndSaveToJson(*addressPtr, *addPtr)
// 	} else {
// 		fmt.Println("Getting address details from Ethereum node...")
// 		config := config.GetConfig()
// 		client, err := ethclient.Dial(config.EthNode)
// 		ethstats.Perror(err)

// 		// address := "0xa1a55063d81696a7f3e94c84b323c792d2501bea" // wallet
// 		// address := "0x629a673a8242c2ac4b7b8c5d8735fbeac21a6205" // nft (sorare)
// 		// address := "0x3B3ee1931Dc30C1957379FAc9aba94D1C48a5405" // nft (fnd)
// 		// address := "0xC36442b4a4522E871399CD717aBDD847Ab11FE88" // nft (uniswap v3)
// 		// address := "0x57f1887a8bf19b14fc0df6fd9b2acc9af147ea85" // nft (ens)
// 		// address := "0xdc76a2de1861ea49e8b41a1de1e461085e8f369f" // nft (terravirtua)
// 		// address := "0xa74476443119A942dE498590Fe1f2454d7D4aC0d" // erc20
// 		// address := "0xC46E0E7eCb3EfCC417f6F89b940FFAFf72556382" // other contract

// 		// a := GetAddressType(address, client)
// 		a, _ := ethstats.GetAddressDetailFromBlockchain(*addressPtr, client)
// 		fmt.Println(a)
// 		return
// 	}
// }
