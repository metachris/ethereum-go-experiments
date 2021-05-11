package main

import (
	"ethstats/ethtools"
	"flag"
	"fmt"
	"log"
)

func ProcessAnalysis(analysis ethtools.AnalysisEntry) {
	fmt.Println(analysis)
}

func main() {
	datePtr := flag.String("date", "", "date (yyyy-mm-dd)")
	idPtr := flag.Int("id", 0, "Analysis id")
	flag.Parse()

	if len(*datePtr) == 0 && *idPtr == 0 {
		log.Fatal("Missing -date or -id argument")
	}

	config := ethtools.GetConfig()
	fmt.Println(config)

	db := ethtools.NewDatabaseConnection(ethtools.GetConfig())

	if *idPtr > 0 {
		entry, found := ethtools.DbGetAnalysisById(db, *idPtr)
		if !found {
			log.Fatal("Analysis with ID ", *idPtr, " not found")
		}
		ProcessAnalysis(entry)
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
