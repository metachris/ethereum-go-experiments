package main

import (
	"ethtools"
	"flag"
	"fmt"
	"log"
)

func main() {
	addressPtr := flag.String("addr", "", "address")
	listPtr := flag.Bool("list", false, "only list entries")
	listEthplorerPtr := flag.Bool("listEthplorer", false, "only list entries")
	addPtr := flag.Bool("add", false, "add to json")
	flag.Parse()

	// List addresses
	if *listPtr {
		addressMap := ethtools.GetAddressDetailMap()
		for _, v := range addressMap {
			fmt.Printf("%s \t %-10v \t %-30v %s \t %d\n", v.Address, v.Type, v.Name, v.Symbol, v.Decimals)
		}
		fmt.Printf("%d entries\n", len(addressMap))
		return
	}

	if *listEthplorerPtr {
		addressMap := ethtools.GetEthplorerTokenInfolMap()
		for _, v := range addressMap {
			fmt.Println(v)
			// fmt.Printf("%s \t %-30v %s \t %s\n", v.Address, v.Name, v.Symbol, v.Decimals)
		}
		fmt.Printf("%d entries\n", len(addressMap))
		return
	}

	if len(*addressPtr) != 42 {
		log.Fatal("Not a valid address")
	}

	info, err := ethtools.FetchTokenInfo(*addressPtr)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(info)

	if !*addPtr {
		return
	}

	ethtools.AddEthplorerokenToFullJson(&info)
	ethtools.AddAddressDetailToJson(info.ToAddressDetail())

	// info, err := ethtools.GetTokenInfo("0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599")
}
