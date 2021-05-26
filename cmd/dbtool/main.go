package main

import (
	"flag"
	"fmt"

	ethtools "github.com/metachris/ethereum-go-experiments/ethtools"
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
