package main

import (
	"encoding/json"
	"log"
	"os"
)

const (
	AddressTypeErc20    string = "ERC20"
	AddressTypeErc721   string = "ERC721"
	AddressTypeAddress  string = "Address"
	AddressTypeDex      string = "Dex"
	AddressTypeContract string = "Contract"
)

type AddressDetail struct {
	Address  string `json:"address"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
}

func GetAddressDetailMap() map[string]AddressDetail {
	file, _ := os.Open("addresses-known.json")
	defer file.Close()

	decoder := json.NewDecoder(file)
	var AddressDetails []AddressDetail
	err := decoder.Decode(&AddressDetails)
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Println(len(AddressDetails), AddressDetails[0].Name)

	// Convert to a map
	AddressDetailMap := make(map[string]AddressDetail)
	for _, v := range AddressDetails {
		AddressDetailMap[v.Address] = v
	}

	// fmt.Println(AddressDetailMap["0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D"])
	return AddressDetailMap
}

// func main() {
// 	AddressDetails := GetAddressDetailMap()
// 	fmt.Println(AddressDetails["xxx"])
// }
