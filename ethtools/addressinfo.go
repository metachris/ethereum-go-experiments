package ethtools

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	// AddressTypeErc20    string = "ERC20"
	// AddressTypeErc721   string = "ERC721"
	AddressTypeToken    string = "ERC_Token"
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

const ( // iota is reset to 0
	DATASET_ADDRESSES = iota
	DATASET_TOKENS    = iota
	DATASET_BOTH      = iota
)

const FN_JSON_TOKENS string = "data/tokens.json"
const FN_JSON_ADDRESSES string = "data/addresses.json"

var AllAddressesFromJson = GetAddressDetailMap(DATASET_BOTH)

func getDatasetFromJson(filename string) []AddressDetail {
	fn, _ := filepath.Abs(filename)
	file, err := os.Open(fn)
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	// Load JSON
	decoder := json.NewDecoder(file)
	var addressDetails []AddressDetail
	err = decoder.Decode(&addressDetails)
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Println(len(AddressDetails), AddressDetails[0].Name)
	return addressDetails
}

func GetAddressDetailMap(dataset int) map[string]AddressDetail {
	var list []AddressDetail
	if dataset == DATASET_ADDRESSES {
		list = getDatasetFromJson(FN_JSON_ADDRESSES)
	} else if dataset == DATASET_TOKENS {
		list = getDatasetFromJson(FN_JSON_TOKENS)
	} else if dataset == DATASET_BOTH {
		list = getDatasetFromJson(FN_JSON_ADDRESSES)
		list2 := getDatasetFromJson(FN_JSON_TOKENS)
		list = append(list, list2...)
	}

	// Convert to map
	AddressDetailMap := make(map[string]AddressDetail)
	for _, v := range list {
		AddressDetailMap[strings.ToLower(v.Address)] = v
	}

	// fmt.Println(AddressDetailMap["0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D"])
	return AddressDetailMap
}

func AddToken(address *AddressDetail) {
	addressMap := GetAddressDetailMap(DATASET_TOKENS)
	addressMap[address.Address] = *address

	// Convert map to JSON
	addressList := make([]AddressDetail, 0, len(addressMap))
	for _, address := range addressMap {
		addressList = append(addressList, address)
	}

	b, err := json.MarshalIndent(addressList, "", "  ")
	Check(err)
	// fmt.Println(b)

	err = ioutil.WriteFile(FN_JSON_TOKENS, b, 0644)
	Check(err)

	fmt.Println("Updated", FN_JSON_TOKENS)
}
