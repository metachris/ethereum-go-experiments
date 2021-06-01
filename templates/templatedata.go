package templates

import (
	"sort"

	"github.com/metachris/ethereum-go-experiments/database"
)

type TemplateData struct {
	Analysis     database.AnalysisEntry
	AddressStats *[]database.AnalysisAddressStatsEntryWithAddress
}

func (tv *TemplateData) GetTopStats(start int, maxEntries int, sortMethod func(i int, j int) bool, checkMethod func(a database.AnalysisAddressStatsEntryWithAddress) bool) *[]database.AnalysisAddressStatsEntryWithAddress {
	if start > len(*tv.AddressStats) {
		return nil
	}

	sort.SliceStable(*tv.AddressStats, sortMethod)

	res := make([]database.AnalysisAddressStatsEntryWithAddress, 0, maxEntries)
	for i := start; i < start+maxEntries && i < len(*tv.AddressStats); i++ {
		entry := (*tv.AddressStats)[i]
		if checkMethod(entry) {
			res = append(res, (*tv.AddressStats)[i])
		}
	}
	return &res
}

func (tv *TemplateData) GetTopErc20Transfer(start int, maxEntries int) *[]database.AnalysisAddressStatsEntryWithAddress {
	sortMethod := func(i, j int) bool {
		return (*tv.AddressStats)[i].NumTxErc20Transfer > (*tv.AddressStats)[j].NumTxErc20Transfer
	}
	checkMethod := func(a database.AnalysisAddressStatsEntryWithAddress) bool { return a.NumTxErc20Transfer > 0 }
	return tv.GetTopStats(start, maxEntries, sortMethod, checkMethod)
}

func (tv *TemplateData) GetTopErc721Transfer(start int, maxEntries int) *[]database.AnalysisAddressStatsEntryWithAddress {
	sortMethod := func(i, j int) bool {
		return (*tv.AddressStats)[i].NumTxErc721Transfer > (*tv.AddressStats)[j].NumTxErc721Transfer
	}
	checkMethod := func(a database.AnalysisAddressStatsEntryWithAddress) bool { return a.NumTxErc721Transfer > 0 }
	return tv.GetTopStats(start, maxEntries, sortMethod, checkMethod)
}

func (tv *TemplateData) GetTopFailedTxReceivers(start int, maxEntries int) *[]database.AnalysisAddressStatsEntryWithAddress {
	sortMethod := func(i, j int) bool {
		return (*tv.AddressStats)[i].NumTxReceivedFailed > (*tv.AddressStats)[j].NumTxReceivedFailed
	}
	checkMethod := func(a database.AnalysisAddressStatsEntryWithAddress) bool { return a.NumTxReceivedFailed > 0 }
	return tv.GetTopStats(start, maxEntries, sortMethod, checkMethod)
}

func (tv *TemplateData) GetTopFailedTxSender(start int, maxEntries int) *[]database.AnalysisAddressStatsEntryWithAddress {
	sortMethod := func(i, j int) bool {
		return (*tv.AddressStats)[i].NumTxSentFailed > (*tv.AddressStats)[j].NumTxSentFailed
	}
	checkMethod := func(a database.AnalysisAddressStatsEntryWithAddress) bool { return a.NumTxSentFailed > 0 }
	return tv.GetTopStats(start, maxEntries, sortMethod, checkMethod)
}
