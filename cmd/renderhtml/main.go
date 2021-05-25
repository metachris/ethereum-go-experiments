package main

import (
	"ethstats/ethtools"
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
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
		entry, found := ethtools.DbGetAnalysisById(db, *idPtr)
		if !found {
			log.Fatal("Analysis with ID ", *idPtr, " not found")
		}
		entry.CalcNumbers()
		SaveAnalysisToHtml(entry, "/tmp/foo.html")
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

func SaveAnalysisToHtml(analysis ethtools.AnalysisEntry, filename string) {
	fmt.Println(analysis)

	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0600)
	ethtools.Perror(err)
	err = f.Truncate(0)
	ethtools.Perror(err)
	defer f.Close()

	funcs := template.FuncMap{"numberFormat": ethtools.NumberToHumanReadableString}
	tmpl, err := template.New("stats.html").Funcs(funcs).ParseFiles("templates/stats.html")
	// t, err := template.ParseFiles("templates/stats.html")
	ethtools.Perror(err)
	err = tmpl.Execute(f, analysis)
	ethtools.Perror(err)

	fmt.Println("Saved HTML to", filename)
}
