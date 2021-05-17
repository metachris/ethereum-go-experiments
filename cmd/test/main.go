package main

import (
	"encoding/hex"
	"fmt"
)

func main() {
	// config := ethtools.GetConfig()
	// fmt.Println(config)

	// val := "000000000000000000000000eb95ff72eab9e8d8fdb545fe15587accf410b42e0000000000000000000000009b40c70964ef961fabb8bf14302347f9de121909362d66c830df8fb6a0335288ccdf4933a4d2dc2ace19f41285b9081c6cd478f2"
	// n2 := new(big.Int)
	// n2.SetString(val, 16)
	x, err := hex.DecodeString("780e9d63")
	if err != nil {
		panic(err)
	}
	fmt.Println(x)

}
