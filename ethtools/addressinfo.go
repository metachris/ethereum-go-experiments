// Tools for dealing with Ethereum addresses: AddressDetail struct, read & write token JSON, get from DB
package ethtools

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	erc20 "ethstats/ethtools/contracts/erc20"
	erc721 "ethstats/ethtools/contracts/erc721"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
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

func IsContract(address string, client *ethclient.Client) bool {
	addr := common.HexToAddress(address)
	b, err := client.CodeAt(context.Background(), addr, nil)
	Perror(err)
	return len(b) > 0
}

func IsErc721(address string, client *ethclient.Client) (isErc721 bool, detail AddressDetail) {
	detail.Address = address

	addr := common.HexToAddress(address)
	instance, err := erc721.NewErc721(addr, client)
	Perror(err)

	isErc721, err = instance.SupportsInterface(nil, INTERFACEID_ERC721)
	if err != nil || !isErc721 {
		return false, detail
	}

	detail.Type = AddressTypeErc721

	// Check if has metadata
	implementsMetadata, err := instance.SupportsInterface(nil, INTERFACEID_ERC721_METADATA)
	if err == nil && implementsMetadata {
		detail.Name, err = instance.Name(nil)
		Perror(err)

		detail.Symbol, err = instance.Symbol(nil)
		Perror(err)
	}

	return true, detail
}

func IsErc20(address string, client *ethclient.Client) (isErc20 bool, detail AddressDetail) {
	detail.Address = address
	addr := common.HexToAddress(address)
	instance, err := erc20.NewErc20(addr, client)
	Perror(err)

	detail.Name, err = instance.Name(nil)
	if err != nil || len(detail.Name) == 0 {
		return false, detail
	}

	// Needs symbol
	detail.Symbol, err = instance.Symbol(nil)
	if err != nil || len(detail.Symbol) == 0 {
		return false, detail
	}

	// Needs decimals
	detail.Decimals, err = instance.Decimals(nil)
	if err != nil {
		return false, detail
	}

	// Needs totalSupply
	_, err = instance.TotalSupply(nil)
	if err != nil {
		return false, detail
	}

	detail.Type = AddressTypeErc20
	return true, detail
}

func GetAddressDetailFromBlockchain(address string, client *ethclient.Client) AddressDetail {
	ret := NewAddressDetail(address)

	if isErc721, detail := IsErc721(address, client); isErc721 {
		return detail
	}

	if isErc20, detail := IsErc20(address, client); isErc20 {
		return detail
	}

	if IsContract(address, client) {
		ret.Type = AddressTypeOtherContract
		return ret
	}

	ret.Type = AddressTypeWallet
	return ret
}
