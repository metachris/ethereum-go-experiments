package main

import ethtools "github.com/metachris/ethereum-go-experiments/ethtools"

func main() {
	db := ethtools.NewDatabaseConnection(ethtools.GetConfig().Database)
	ethtools.AddAddressesFromJsonToDatabase(db)

	// a, err := ethtools.GetAddressFromDatabase(db, "0xdac17f958d2ee523a2206206994597c13d831ec7")
	// fmt.Println(a, err)
}
