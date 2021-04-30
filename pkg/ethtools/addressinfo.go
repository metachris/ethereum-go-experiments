package ethtools

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

const (
	AddressTypeErc20    string = "ERC20"
	AddressTypeErc721   string = "ERC721"
	AddressTypeAddress  string = "Address"
	AddressTypeContract string = "Contract"
)

type AddressDetail struct {
	Address  string `json:"address"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
}

const FN_ADDRESS_JSON string = "data/addresses.json"

func GetAddressDetailMap() map[string]AddressDetail {
	fn, err := filepath.Abs(FN_ADDRESS_JSON)
	file, err := os.Open(fn)
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	// Load JSON
	decoder := json.NewDecoder(file)
	var AddressDetails []AddressDetail
	err = decoder.Decode(&AddressDetails)
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Println(len(AddressDetails), AddressDetails[0].Name)

	// Convert to map
	AddressDetailMap := make(map[string]AddressDetail)
	for _, v := range AddressDetails {
		AddressDetailMap[v.Address] = v
	}

	// fmt.Println(AddressDetailMap["0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D"])
	return AddressDetailMap
}

func AddAddressDetailToJson(address *AddressDetail) {
	addressMap := GetAddressDetailMap()
	addressMap[address.Address] = *address

	// Convert map to JSON
	addressList := make([]AddressDetail, 0, len(addressMap))
	for _, address := range addressMap {
		addressList = append(addressList, address)
	}

	b, err := json.MarshalIndent(addressList, "", "  ")
	Check(err)
	// fmt.Println(b)

	err = ioutil.WriteFile(FN_ADDRESS_JSON, b, 0644)
	Check(err)

	fmt.Println("Updated", FN_ADDRESS_JSON)
}
