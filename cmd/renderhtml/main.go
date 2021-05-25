package main

import (
	"ethstats/ethtools"
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"sort"
)

func main() {
	datePtr := flag.String("date", "", "date (yyyy-mm-dd)")
	idPtr := flag.Int("id", 0, "Analysis id")
	flag.Parse()

	if len(*datePtr) == 0 && *idPtr == 0 {
		log.Fatal("Missing -date or -id argument")
	}

	config := ethtools.GetConfig()
	fmt.Println(config)

	db := ethtools.NewDatabaseConnection(ethtools.GetConfig().Database)

	if *idPtr > 0 {
		analysis, found := ethtools.DbGetAnalysisById(db, *idPtr)
		if !found {
			log.Fatal("Analysis with ID ", *idPtr, " not found")
		}
		analysis.CalcNumbers()

		fmt.Println("Getting stats entries...")
		entries, err := ethtools.DbGetAddressStatEntriesForAnalysisId(db, *idPtr)
		ethtools.Perror(err)
		fmt.Println(len(*entries))

		SaveAnalysisToHtml(analysis, entries, "/tmp/foo.html")
		return
	}

	if len(*datePtr) > 0 {
		entries, found := ethtools.DbGetAnalysesByDate(db, *datePtr)
		if !found {
			log.Fatal("Analysis with date ", *datePtr, " not found")
		}
		fmt.Println(entries)
	}
}

type TemplateVar struct {
	Analysis     ethtools.AnalysisEntry
	AddressStats *[]ethtools.AnalysisAddressStatsEntryWithAddress
}

func (tv *TemplateVar) GetTopErc20Transfer(maxEntries int) *[]ethtools.AnalysisAddressStatsEntryWithAddress {
	res := make([]ethtools.AnalysisAddressStatsEntryWithAddress, 0)

	// Sort list
	sort.SliceStable(*tv.AddressStats, func(i, j int) bool {
		return (*tv.AddressStats)[i].NumTxErc20Transfer > (*tv.AddressStats)[j].NumTxErc20Transfer
	})

	// Add to result
	for i := 0; i < maxEntries && i < len(*tv.AddressStats); i++ {
		if (*tv.AddressStats)[i].NumTxErc20Transfer > 0 {
			res = append(res, (*tv.AddressStats)[i])
		}
	}

	return &res
}

func (tv *TemplateVar) GetTopErc721Transfer(maxEntries int) *[]ethtools.AnalysisAddressStatsEntryWithAddress {
	res := make([]ethtools.AnalysisAddressStatsEntryWithAddress, 0)

	// Sort list
	sort.SliceStable(*tv.AddressStats, func(i, j int) bool {
		return (*tv.AddressStats)[i].NumTxErc721Transfer > (*tv.AddressStats)[j].NumTxErc721Transfer
	})

	// Add to result
	for i := 0; i < maxEntries && i < len(*tv.AddressStats); i++ {
		if (*tv.AddressStats)[i].NumTxErc721Transfer > 0 {
			res = append(res, (*tv.AddressStats)[i])
		}
	}

	return &res
}

func SaveAnalysisToHtml(analysis ethtools.AnalysisEntry, stats *[]ethtools.AnalysisAddressStatsEntryWithAddress, filename string) {
	fmt.Println(analysis)

	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0600)
	ethtools.Perror(err)
	err = f.Truncate(0)
	ethtools.Perror(err)
	defer f.Close()

	tmplData := TemplateVar{
		Analysis:     analysis,
		AddressStats: stats,
	}

	funcs := template.FuncMap{
		"numberFormat": ethtools.NumberToHumanReadableString,
		"topErc20":     tmplData.GetTopErc20Transfer,
		"topErc721":    tmplData.GetTopErc721Transfer,
	}
	tmpl, err := template.New("stats.html").Funcs(funcs).ParseFiles("templates/stats.html")
	// t, err := template.ParseFiles("templates/stats.html")
	ethtools.Perror(err)
	err = tmpl.Execute(f, tmplData)
	ethtools.Perror(err)

	fmt.Println("Saved HTML to", filename)
}
