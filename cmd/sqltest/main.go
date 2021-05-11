package main

import "ethstats/ethtools"

func main() {
	db := ethtools.OpenDatabase(ethtools.GetDbConfig())
	// db := ethtools.OpenDatabase(ethtools.GetProdDbConfig())
	ethtools.AddAddressesFromJsonToDatabase(db)
	// a, err := ethtools.GetAddressFromDatabase(db, "0xdac17f958d2ee523a2206206994597c13d831ec7")
	// fmt.Println(a, err)
}
