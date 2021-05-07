package main

import (
	"ethtools"
)

func main() {
	db := ethtools.GetDatabase()
	ethtools.AddAddressesToDatabase(db)
}
