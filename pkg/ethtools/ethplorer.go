package ethtools

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	FN_ADDRESS_JSON_ETHPLORER string        = "data/ethplorer_tokens.json"
	ETHPLORER_API_KEY         string        = "EK-eTp4w-dZq2YNw-fs9hU"
	ETHPLORER_TIMEOUT         time.Duration = 1 * time.Second
	// ETHPLORER_API_KEY string = "freekey"
)

type EthplorerTokenInfo struct {
	Address           string   `json:"address"`
	Name              string   `json:"name"`
	Decimals          string   `json:"decimals"`
	Symbol            string   `json:"symbol"`
	TotalSupply       string   `json:"totalSupply"`
	Owner             string   `json:"owner"`
	TransfersCount    int      `json:"transfersCount"`
	LastUpdated       int      `json:"lastUpdated"`
	IssuancesCount    int      `json:"issuancesCount"`
	HoldersCount      int      `json:"holdersCount"`
	Description       string   `json:"description"`
	Website           string   `json:"website"`
	Image             string   `json:"image"`
	Coingecko         string   `json:"coingecko"`
	EthTransfersCount int      `json:"ethTransfersCount"`
	PublicTags        []string `json:"publicTags"`
	CountOps          int      `json:"countOps"`
}

type InternalEthplorerTokenInfo struct {
	EthplorerTokenInfo
	DownloadedAt int64 // UTC timestamp in seconds
}

func (tokenInfo InternalEthplorerTokenInfo) ToAddressDetail() *AddressDetail {
	decimals, _ := strconv.Atoi(tokenInfo.Decimals)

	return &AddressDetail{
		Address:  tokenInfo.Address,
		Name:     tokenInfo.Name,
		Symbol:   tokenInfo.Symbol,
		Decimals: decimals,
		Type:     "TOKEN",
	}
}

func FetchTokenInfo(address string) (InternalEthplorerTokenInfo, error) {
	target := InternalEthplorerTokenInfo{}

	url := fmt.Sprintf("https://api.ethplorer.io/getTokenInfo/%s?apiKey="+ETHPLORER_API_KEY, address)
	fmt.Println(url)

	var myClient = &http.Client{Timeout: 10 * time.Second}
	r, err := myClient.Get(url)
	Check(err)

	defer r.Body.Close()
	if r.StatusCode != 200 {
		return target, errors.New("Error status " + r.Status)
	}

	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&target)
	target.DownloadedAt = time.Now().UTC().Unix()

	return target, err
}

func GetEthplorerTokenInfolMap() map[string]InternalEthplorerTokenInfo {
	fn, err := filepath.Abs(FN_ADDRESS_JSON_ETHPLORER)
	file, err := os.Open(fn)
	Check(err)
	defer file.Close()

	// Load JSON
	decoder := json.NewDecoder(file)
	var InternalEthplorerTokenInfos []InternalEthplorerTokenInfo
	err = decoder.Decode(&InternalEthplorerTokenInfos)
	Check(err)
	// fmt.Println(len(InternalEthplorerTokenInfos), InternalEthplorerTokenInfos[0].Name)

	// Convert to map
	tokenInfoMap := make(map[string]InternalEthplorerTokenInfo)
	for _, v := range InternalEthplorerTokenInfos {
		tokenInfoMap[strings.ToLower(v.Address)] = v
	}

	// fmt.Println(EthplorerTokenInfoMap["0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D"])
	return tokenInfoMap
}

func AddEthplorerokenToFullJson(tokenInfo *InternalEthplorerTokenInfo) {
	tokenMap := GetEthplorerTokenInfolMap()
	tokenMap[tokenInfo.Address] = *tokenInfo

	// Convert map to JSON
	tokenList := make([]InternalEthplorerTokenInfo, 0, len(tokenMap))
	for _, token := range tokenMap {
		tokenList = append(tokenList, token)
	}

	b, err := json.MarshalIndent(tokenList, "", "  ")
	Check(err)
	// fmt.Println(b)

	err = ioutil.WriteFile(FN_ADDRESS_JSON_ETHPLORER, b, 0644)
	Check(err)

	fmt.Println("Updated", FN_ADDRESS_JSON_ETHPLORER)
}
