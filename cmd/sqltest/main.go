package main

import (
	"github.com/metachris/ethereum-go-experiments/config"
	"github.com/metachris/ethereum-go-experiments/ethstats"
)

func main() {
	db := ethstats.NewDatabaseConnection(config.GetConfig().Database)
	ethstats.AddAddressesFromJsonToDatabase(db)

	// a, err := ethstats.GetAddressFromDatabase(db, "0xdac17f958d2ee523a2206206994597c13d831ec7")
	// fmt.Println(a, err)
}
