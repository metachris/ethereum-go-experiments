package main

import (
	"flag"
	"fmt"

	"github.com/metachris/ethereum-go-experiments/ethstats"
)

func main() {
	resetPtr := flag.Bool("reset", false, "reset database")
	flag.Parse()

	if *resetPtr {
		cfg := ethstats.GetConfig()
		db := ethstats.GetDatabase(cfg.Database)
		ethstats.ResetDatabase(db)
		return
	}

	fmt.Println("Nothing to do. Check -help")
}