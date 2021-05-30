package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"math/big"
	"os"

	"github.com/metachris/ethereum-go-experiments/ethstats"
	"github.com/metachris/ethereum-go-experiments/templates"
)

func main() {
	// datePtr := flag.String("date", "", "date (yyyy-mm-dd)")
	idPtr := flag.Int("id", 0, "Analysis id")
	outPtr := flag.String("out", "/tmp/ethstats.html", "Output filename (html)")
	flag.Parse()

	if *idPtr == 0 {
		log.Fatal("Missing -id argument")
	}

	config := ethstats.GetConfig()
	fmt.Println(config)

	db := ethstats.NewDatabaseConnection(ethstats.GetConfig().Database)

	analysis, found := ethstats.DbGetAnalysisById(db, *idPtr)
	if !found {
		log.Fatal("Analysis with ID ", *idPtr, " not found")
	}

	// Prepare some numbers
	analysis.CalcNumbers()

	fmt.Println("Getting stats entries...")
	entries, err := ethstats.DbGetAddressStatEntriesForAnalysisId(db, *idPtr)
	ethstats.Perror(err)
	fmt.Println(len(*entries), "entries")

	filename := *outPtr // "/tmp/foo.html"
	SaveAnalysisToHtml(analysis, entries, filename)
	fmt.Println("Saved HTML to", filename)
}

func SaveAnalysisToHtml(analysis ethstats.AnalysisEntry, stats *[]ethstats.AnalysisAddressStatsEntryWithAddress, filename string) {
	// fmt.Println(analysis)

	// Prepare HTML output file
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0600)
	ethstats.Perror(err)
	err = f.Truncate(0)
	ethstats.Perror(err)
	defer f.Close()

	// Prepare template data
	tmplData := templates.TemplateData{
		Analysis:     analysis,
		AddressStats: stats,
	}

	weiStrToHumanEth := func(weiStr string) string {
		wei := new(big.Int)
		wei.SetString(weiStr, 10)
		return ethstats.WeiBigIntToEthString(wei, 2)
	}

	// Prepare template functions
	funcs := template.FuncMap{
		"add":                     func(x, y int) int { return x + y },
		"numberFormat":            ethstats.NumberToHumanReadableString,
		"topErc20":                tmplData.GetTopErc20Transfer,
		"topErc721":               tmplData.GetTopErc721Transfer,
		"getTopFailedTxReceivers": tmplData.GetTopFailedTxReceivers,
		"getTopFailedTxSender":    tmplData.GetTopFailedTxSender,
		"weiStrToHumanEth":        weiStrToHumanEth,
	}

	// Execute template
	tmpl, err := template.New("stats.html").Funcs(funcs).ParseFiles("templates/stats.html")
	ethstats.Perror(err)
	err = tmpl.Execute(f, tmplData)
	ethstats.Perror(err)
}
