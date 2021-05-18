package main

import (
	"ethstats/ethtools"
	"flag"
	"fmt"
)

func main() {
	resetPtr := flag.Bool("reset", false, "reset database")
	flag.Parse()

	if *resetPtr {
		cfg := ethtools.GetConfig()
		db := ethtools.GetDatabase(cfg.Database)
		ethtools.ResetDatabase(db)
		return
	}

	fmt.Println("Nothing to do. Check -help")
}
