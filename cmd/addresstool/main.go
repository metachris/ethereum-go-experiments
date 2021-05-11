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
	} else {
		getToken(*addressPtr, *addPtr)
	}
}
