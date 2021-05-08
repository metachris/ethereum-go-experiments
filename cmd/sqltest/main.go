package main

import (
	"ethtools"
)

func main() {
	db := ethtools.GetDatabase()
	ethtools.AddAddressesFromJsonToDatabase(db)
	// a, err := ethtools.GetAddressFromDatabase(db, "0xdac17f958d2ee523a2206206994597c13d831ec7")
	// fmt.Println(a, err)
}
