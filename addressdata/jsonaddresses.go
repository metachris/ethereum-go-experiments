// Tools for dealing with Ethereum addresses: AddressDetail struct, read & write token JSON, get from DB
package addressdata

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/metachris/ethereum-go-experiments/core"
)

const ( // iota is reset to 0
	DATASET_ADDRESSES = iota
	DATASET_TOKENS    = iota
	DATASET_BOTH      = iota
)

const FN_JSON_TOKENS string = "addressdata/tokens.json"
const FN_JSON_ADDRESSES string = "addressdata/addresses.json"

func GetAddressesFromJson(filename string) []core.AddressDetail {
	fn, _ := filepath.Abs(filename)
	file, err := os.Open(fn)
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	// Load JSON
	decoder := json.NewDecoder(file)
	var addressDetails []core.AddressDetail
	err = decoder.Decode(&addressDetails)
	if err != nil {
		log.Fatal(err)
	}

	// type field is not mandatory In JSON. Use wallet as default.
	for i, v := range addressDetails {
		addressDetails[i].Address = strings.ToLower(addressDetails[i].Address)
		if v.Type == "" {
			addressDetails[i].Type = core.AddressTypePubkey
		}
	}

	return addressDetails
}

func GetAddressDetailMap(dataset int) map[string]core.AddressDetail {
	var list []core.AddressDetail
	if dataset == DATASET_ADDRESSES {
		list = GetAddressesFromJson(FN_JSON_ADDRESSES)
	} else if dataset == DATASET_TOKENS {
		list = GetAddressesFromJson(FN_JSON_TOKENS)
	} else if dataset == DATASET_BOTH {
		list = GetAddressesFromJson(FN_JSON_ADDRESSES)
		list2 := GetAddressesFromJson(FN_JSON_TOKENS)
		list = append(list, list2...)
	}

	// Convert to map
	AddressDetailMap := make(map[string]core.AddressDetail)
	for _, v := range list {
		AddressDetailMap[strings.ToLower(v.Address)] = v
	}

	return AddressDetailMap
}
