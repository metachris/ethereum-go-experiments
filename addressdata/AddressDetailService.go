package addressdata

import (
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/metachris/ethereum-go-experiments/core"
)

type AddressDetailService struct{}

func (ads AddressDetailService) EnsureIsLoaded(a *core.AddressDetail, client *ethclient.Client) {
	if a.IsLoaded() {
		return
	}

	b, _ := GetAddressDetail(a.Address, client)
	a.Address = b.Address
	a.Type = b.Type
	a.Name = b.Name
	a.Symbol = b.Symbol
	a.Decimals = b.Decimals
}
