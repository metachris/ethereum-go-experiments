package addressdata

import (
	"strings"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/metachris/ethereum-go-experiments/core"
)

// Initialize address cache with data from JSON
var AddressDetailCache = GetAddressDetailMap(DATASET_BOTH)

// GetAddressDetail returns the core.AddressDetail from JSON. If not exists then query the Blockchain and caches it for future use
func GetAddressDetail(address string, client *ethclient.Client) (detail core.AddressDetail, found bool) {
	// Check in Cache
	addr, found := AddressDetailCache[strings.ToLower(address)]
	if found {
		return addr, true
	}

	// Without connection, return Detail with just address
	if client == nil {
		return core.NewAddressDetail(address), false
	} else {
		detail = core.NewAddressDetail(address) // default
	}

	// Look up in Blockchain
	if core.GetConfig().LowApiCallMode {
		return detail, found
	}

	detail, found = core.GetAddressDetailFromBlockchain(address, client)
	if found {
		AddAddressDetailToCache(detail)
	}
	return detail, found
}

func AddAddressDetailToCache(detail core.AddressDetail) {
	AddressDetailCache[strings.ToLower(detail.Address)] = detail
}
