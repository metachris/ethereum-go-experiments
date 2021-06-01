package main

import (
	"flag"
	"fmt"

	"github.com/metachris/ethereum-go-experiments/core"
	"github.com/metachris/ethereum-go-experiments/database"
)

func main() {
	resetPtr := flag.Bool("reset", false, "reset database")
	flag.Parse()

	if *resetPtr {
		cfg := core.GetConfig()
		db := database.NewStatsService(cfg.Database)
		db.Reset()
		return
	}

	fmt.Println("Nothing to do. Check -help")
}
