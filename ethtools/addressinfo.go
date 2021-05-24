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
	INTERFACEID_ERC1155           = [4]byte{217, 182, 122, 38} // 0xd9b67a26
)

type AddressType string

const (
	AddressTypeInit AddressType = "" // Init value

	// After detection
	AddressTypeErc20         AddressType = "Erc20"
	AddressTypeErc721        AddressType = "Erc721"
	AddressTypeErcToken      AddressType = "ErcToken" // Just recognized transfer call. Either ERC20 or ERC721. Will be updated on next sync.
	AddressTypeOtherContract AddressType = "OtherContract"
	AddressTypePubkey        AddressType = "Wallet" // nothing else detected, probably just a wallet
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

func (a *AddressDetail) IsLoaded() bool {
	return a.Type != AddressTypeInit
}

func (a *AddressDetail) IsErc20() bool {
	return a.Type == AddressTypeErc20
}

func (a *AddressDetail) IsErc721() bool {
	return a.Type == AddressTypeErc721
}

// Returns a new unknown address detail
func NewAddressDetail(address string) AddressDetail {
	return AddressDetail{Address: address, Type: AddressTypeInit}
}

const ( // iota is reset to 0
	DATASET_ADDRESSES = iota
	DATASET_TOKENS    = iota
	DATASET_BOTH      = iota
)

const FN_JSON_TOKENS string = "data/tokens.json"
const FN_JSON_ADDRESSES string = "data/addresses.json"

var AddressDetailCache = GetAddressDetailMap(DATASET_BOTH)

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

	// type field is not mandatory In JSON. Use wallet as default.
	for i, v := range addressDetails {
		addressDetails[i].Address = strings.ToLower(addressDetails[i].Address)
		if v.Type == "" {
			addressDetails[i].Type = AddressTypePubkey
		}
	}

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

	// fmt.Println(AddressDetailMap[strings.ToLower("0x9f8F72aA9304c8B593d555F12eF6589cC3A579A2")])
	// fmt.Println(AddressDetailMap["0x9f8F72aA9304c8B593d555F12eF6589cC3A579A2"])
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

func SmartContractSupportsInterface(address string, interfaceId [4]byte, client *ethclient.Client) bool {
	addr := common.HexToAddress(address)
	instance, err := erc721.NewErc721(addr, client) // the SupportsInterface signature is the same for all contract types, so we can just use the ERC721 interface
	Perror(err)
	isSupported, err := instance.SupportsInterface(nil, INTERFACEID_ERC721)
	return err == nil && isSupported
}

// TODO: Currently returns true for every SC that supports INTERFACEID_ERC165. It should really be INTERFACEID_ERC721,
// but that doesn't detect some SCs, eg. cryptokitties https://etherscan.io/address/0x06012c8cf97BEaD5deAe237070F9587f8E7A266d#readContract
// As a quick fix, just checks ERC165 and count it as ERC721 address. Improve with further/better SC method checks.
func IsErc721(address string, client *ethclient.Client) (isErc721 bool, detail AddressDetail) {
	detail.Address = address

	addr := common.HexToAddress(address)
	instance, err := erc721.NewErc721(addr, client)
	Perror(err)

	isErc721, err = instance.SupportsInterface(nil, INTERFACEID_ERC165)
	if err != nil || !isErc721 {
		return false, detail
	}

	// It appears to be ERC721
	detail.Type = AddressTypeErc721

	// Try to get a name and symbol
	detail.Name, _ = instance.Name(nil)
	// if err != nil {
	// 	// eg. "abi: cannot marshal in to go slice: offset 33 would go over slice boundary (len=32)"
	// 	// ignore, since we don't check erc721 metadata extension
	// }

	detail.Symbol, _ = instance.Symbol(nil)
	// if err != nil {
	// 	// ignore, since we don't check erc721 metadata extension
	// }

	return true, detail
}

func IsErc20(address string, client *ethclient.Client) (isErc20 bool, detail AddressDetail) {
	detail.Address = address
	addr := common.HexToAddress(address)
	instance, err := erc20.NewErc20(addr, client)
	Perror(err)

	detail.Name, err = instance.Name(nil)
	if err != nil || len(detail.Name) == 0 {
		// fmt.Println(1, err)
		return false, detail
	}

	// Needs symbol
	detail.Symbol, err = instance.Symbol(nil)
	if err != nil || len(detail.Symbol) == 0 {
		// fmt.Println(2)
		return false, detail
	}

	// Needs decimals
	detail.Decimals, err = instance.Decimals(nil)
	if err != nil {
		// fmt.Println(3, err)
		return false, detail
	}

	// Needs totalSupply
	_, err = instance.TotalSupply(nil)
	if err != nil {
		// fmt.Println(4)
		return false, detail
	}

	detail.Type = AddressTypeErc20
	return true, detail
}

func GetAddressDetailFromBlockchain(address string, client *ethclient.Client) (detail AddressDetail, found bool) {
	ret := NewAddressDetail(address)

	if isErc721, detail := IsErc721(address, client); isErc721 {
		return detail, true
	}

	if isErc20, detail := IsErc20(address, client); isErc20 {
		return detail, true
	}

	if IsContract(address, client) {
		ret.Type = AddressTypeOtherContract
		return ret, true
	}

	ret.Type = AddressTypePubkey
	return ret, false
}

// GetAddressDetail returns the AddressDetail from JSON. If not exists then query the Blockchain
func GetAddressDetail(address string, client *ethclient.Client) (detail AddressDetail, found bool) {
	// Check in Cache
	addr, found := AddressDetailCache[strings.ToLower(address)]
	if found {
		return addr, true
	}

	// Without connection, return Detail with just address
	if client == nil {
		return NewAddressDetail(address), false
	}

	// Look up in Blockchain
	detail, found = GetAddressDetailFromBlockchain(address, client)
	if found {
		AddAddressDetailToCache(detail)
	}
	return detail, found
}

func AddAddressDetailToCache(detail AddressDetail) {
	AddressDetailCache[strings.ToLower(detail.Address)] = detail
}
