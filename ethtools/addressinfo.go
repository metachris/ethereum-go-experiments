// Tools for dealing with Ethereum addresses: AddressDetail struct, read & write token JSON, get from DB
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

var (
	INTERFACEID_ERC165            = [4]byte{1, 255, 201, 167}  // 0x01ffc9a7
	INTERFACEID_ERC721            = [4]byte{128, 172, 88, 205} // 0x80ac58cd
	INTERFACEID_ERC721_METADATA   = [4]byte{91, 94, 19, 159}   // 0x5b5e139f
	INTERFACEID_ERC721_ENUMERABLE = [4]byte{120, 14, 157, 99}  // 0x780e9d63
)

type AddressType string

const (
	AddressTypeWallet        AddressType = "Wallet"
	AddressTypeErc20         AddressType = "Erc20"
	AddressTypeErc721        AddressType = "Erc721"
	AddressTypeErcToken      AddressType = "ErcToken" // Just recognized transfer call. Either ERC20 or ERC721. Will be updated on next sync.
	AddressTypeUnknown       AddressType = "Unknown"  // For new addresses, will be updated on next sync.
	AddressTypeOtherContract AddressType = "OtherContract"
)

type AddressDetail struct {
	Address  string      `json:"address"`
	Type     AddressType `json:"type"`
	Name     string      `json:"name"`
	Symbol   string      `json:"symbol"`
	Decimals uint8       `json:"decimals"`
}

func (a AddressDetail) String() string {
	return fmt.Sprintf("%s [%s] name=%s, symbol=%s, decimals=%d", a.Address, a.Type, a.Name, a.Symbol, a.Decimals)
}

func NewAddressDetail(address string) AddressDetail {
	return AddressDetail{Address: address, Type: AddressTypeUnknown}
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

func AddTokenToJson(address *AddressDetail) {
	addressMap := GetAddressDetailMap(DATASET_TOKENS)
	addressMap[address.Address] = *address

	// Convert map to JSON
	addressList := make([]AddressDetail, 0, len(addressMap))
	for _, address := range addressMap {
		addressList = append(addressList, address)
	}

	b, err := json.MarshalIndent(addressList, "", "  ")
	Perror(err)
	// fmt.Println(b)

	err = ioutil.WriteFile(FN_JSON_TOKENS, b, 0644)
	Perror(err)

	fmt.Println("Updated", FN_JSON_TOKENS)
}
