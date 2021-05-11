package main

import (
	"ethstats/ethtools"
	"fmt"
)

func main() {
	config := ethtools.GetConfig()
	fmt.Println(config)
}
