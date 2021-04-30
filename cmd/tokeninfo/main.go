package main

import (
	"ethtools"
	"flag"
	"fmt"
	"log"
)

func main() {
	addrPtr := flag.String("addr", "x", "a string")
	flag.Parse()
	fmt.Println("Hello", addrPtr, *addrPtr)
	// fmt.Println("Hello", ethtools.One)

	// // AddressDetails := GetAddressDetailMap()
	// // fmt.Println(AddressDetails["xxx"])
	// info, err := ethtools.GetTokenInfo("0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599")
	info, err := ethtools.GetTokenInfo(*addrPtr)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(info)
}
