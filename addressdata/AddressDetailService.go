package addressdata

import (
	"strings"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/metachris/ethereum-go-experiments/core"
	"github.com/metachris/go-ethutils/addressdetail"
	"github.com/metachris/go-ethutils/smartcontracts"
)

type AddressDetailService struct {
	Client *ethclient.Client

	// Initialize address cache with data from JSON
	Cache map[string]addressdetail.AddressDetail
}

func NewAddressDetailService(client *ethclient.Client) *AddressDetailService {
	return &AddressDetailService{
		Client: client,
		Cache:  GetAddressDetailMap(DATASET_BOTH),
	}
}

func (ads AddressDetailService) EnsureIsLoaded(a *addressdetail.AddressDetail) {
	if !a.IsInitial() {
		return
	}

	b, _ := ads.GetAddressDetail(a.Address, ads.Client)
	a.Address = b.Address
	a.Type = b.Type
	a.Name = b.Name
	a.Symbol = b.Symbol
	a.Decimals = b.Decimals
}

// GetAddressDetail returns the addressdetail.AddressDetail from JSON. If not exists then query the Blockchain and caches it for future use
func (ads *AddressDetailService) GetAddressDetail(address string, client *ethclient.Client) (detail addressdetail.AddressDetail, found bool) {
	// Check in Cache
	addr, found := ads.Cache[strings.ToLower(address)]
	if found {
		return addr, true
	}

	// Without connection, return Detail with just address
	if client == nil {
		return addressdetail.NewAddressDetail(address), false
	} else {
		detail = addressdetail.NewAddressDetail(address) // default
	}

	// Look up in Blockchain
	if core.Cfg.LowApiCallMode {
		return detail, found
	}

	detail, found = GetAddressDetailFromBlockchain(address, client)
	if found {
		ads.AddAddressDetailToCache(detail)
	}
	return detail, found
}

func (ads *AddressDetailService) AddAddressDetailToCache(detail addressdetail.AddressDetail) {
	ads.Cache[strings.ToLower(detail.Address)] = detail
}

func GetAddressDetailFromBlockchain(address string, client *ethclient.Client) (detail addressdetail.AddressDetail, found bool) {
	detail = addressdetail.NewAddressDetail(address)

	// check fr erc721
	isErc721, detail, _ := smartcontracts.IsErc721(address, client)
	// if err != nil {
	// 	return detail, found, err
	// }
	if isErc721 {
		return detail, true
	}

	// check for erc20
	isErc20, detail, _ := smartcontracts.IsErc20(address, client)
	// if err != nil {
	// 	return detail, found, err
	// }
	if isErc20 {
		return detail, true
	}

	// check if any type of smart contract
	isContract, _ := smartcontracts.IsContract(address, client)
	// if err != nil {
	// 	return detail, found, err
	// }
	if isContract {
		detail.Type = addressdetail.AddressTypeOtherContract
		return detail, true
	}

	// return just a wallet
	detail.Type = addressdetail.AddressTypeWallet
	return detail, false
}
