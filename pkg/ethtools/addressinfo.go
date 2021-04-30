package ethtools

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
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

func GetAddressDetailMap() map[string]AddressDetail {
	fn, err := filepath.Abs("data/addresses-known.json")
	file, err := os.Open(fn)
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	decoder := json.NewDecoder(file)
	var AddressDetails []AddressDetail
	err = decoder.Decode(&AddressDetails)
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

type EthplorerTokenInfo struct {
	Address  string `json:"address"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals string `json:"decimals"`
}

func GetTokenInfo(address string) (EthplorerTokenInfo, error) {
	target := EthplorerTokenInfo{}

	url := fmt.Sprintf("https://api.ethplorer.io/getTokenInfo/%s?apiKey=freekey", address)
	fmt.Println(url)

	var myClient = &http.Client{Timeout: 10 * time.Second}
	r, err := myClient.Get(url)
	if err != nil {
		return target, err
	}

	defer r.Body.Close()
	if r.StatusCode != 200 {
		return target, errors.New("Error status " + r.Status)
	}

	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&target)
	return target, err
}

// func main() {
// 	// AddressDetails := GetAddressDetailMap()
// 	// fmt.Println(AddressDetails["xxx"])
// 	info, err := GetTokenInfo("0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599")
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	fmt.Println(info)
// }
