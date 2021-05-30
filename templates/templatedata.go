package templates

import (
	"sort"

	"github.com/metachris/ethereum-go-experiments/ethstats"
)

type TemplateData struct {
	Analysis     ethstats.AnalysisEntry
	AddressStats *[]ethstats.AnalysisAddressStatsEntryWithAddress
}

func (tv *TemplateData) GetTopErc20Transfer(start int, maxEntries int) *[]ethstats.AnalysisAddressStatsEntryWithAddress {
	if start > len(*tv.AddressStats) {
		return nil
	}

	sort.SliceStable(*tv.AddressStats, func(i, j int) bool {
		return (*tv.AddressStats)[i].NumTxErc20Transfer > (*tv.AddressStats)[j].NumTxErc20Transfer
	})

	res := make([]ethstats.AnalysisAddressStatsEntryWithAddress, 0, maxEntries)
	for i := start; i < start+maxEntries && i < len(*tv.AddressStats); i++ {
		if (*tv.AddressStats)[i].NumTxErc20Transfer > 0 {
			res = append(res, (*tv.AddressStats)[i])
		}
	}
	return &res
}

func (tv *TemplateData) GetTopErc721Transfer(maxEntries int) *[]ethstats.AnalysisAddressStatsEntryWithAddress {
	sort.SliceStable(*tv.AddressStats, func(i, j int) bool {
		return (*tv.AddressStats)[i].NumTxErc721Transfer > (*tv.AddressStats)[j].NumTxErc721Transfer
	})

	// Add to result
	res := make([]ethstats.AnalysisAddressStatsEntryWithAddress, 0)
	for i := 0; i < maxEntries && i < len(*tv.AddressStats); i++ {
		if (*tv.AddressStats)[i].NumTxErc721Transfer > 0 {
			res = append(res, (*tv.AddressStats)[i])
		}
	}
	return &res
}

func (tv *TemplateData) GetTopFailedTxReceivers(maxEntries int) *[]ethstats.AnalysisAddressStatsEntryWithAddress {
	sort.SliceStable(*tv.AddressStats, func(i, j int) bool {
		return (*tv.AddressStats)[i].NumTxReceivedFailed > (*tv.AddressStats)[j].NumTxReceivedFailed
	})

	// Add to result
	res := make([]ethstats.AnalysisAddressStatsEntryWithAddress, 0)
	for i := 0; i < maxEntries && i < len(*tv.AddressStats); i++ {
		if (*tv.AddressStats)[i].NumTxReceivedFailed > 0 {
			res = append(res, (*tv.AddressStats)[i])
		}
	}
	return &res
}

func (tv *TemplateData) GetTopFailedTxSender(maxEntries int) *[]ethstats.AnalysisAddressStatsEntryWithAddress {
	sort.SliceStable(*tv.AddressStats, func(i, j int) bool {
		return (*tv.AddressStats)[i].NumTxSentFailed > (*tv.AddressStats)[j].NumTxSentFailed
	})

	// Add to result
	res := make([]ethstats.AnalysisAddressStatsEntryWithAddress, 0)
	for i := 0; i < maxEntries && i < len(*tv.AddressStats); i++ {
		if (*tv.AddressStats)[i].NumTxSentFailed > 0 {
			res = append(res, (*tv.AddressStats)[i])
		}
	}
	return &res
}
