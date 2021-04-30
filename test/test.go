package xxx

import (
	"fmt"
	"math/big"
	"strconv"
)

func test1() {
	fmt.Println(("test"))

	arr := [3]string{"a", "b", "c"}
	// arr := []bool{true, true, false}
	fmt.Println("arr", arr)

	b := arr[1:3]
	fmt.Println("b", b)

	arr[1] = "x"
	fmt.Println("arr", arr)
}

func test2() {
	for i := 1; i < 5; i++ {
		fmt.Println(i)
	}

	// fmt.Println("test2")
	// a := make([]int, 5)

	// a[1] = 7
	// printSlice("a", a)

	// for i, v := range a {
	// 	fmt.Println(i, v)
	// }
}

// func makeTime(dayString string, hour int) time.Time {
// 	dateString := fmt.Sprintf("%sT%02d:00:00Z", dayString, hour)
// 	// fmt.Println(ts)
// 	t, err := time.Parse(time.RFC3339, dateString)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	// fmt.Println(t)
// 	return t
// }

// func getTargetBlocknumber(dayString string, hour int) int {
// 	// Create a time.Time instance
// 	targetTime := makeTime(dayString, hour)
// 	fmt.Println(targetTime, targetTime.Unix())

// 	// Calculate block-difference from reference block
// 	referenceBlockNumber := 12323940
// 	referenceBlockTimestamp := 1619546404 // 2021-04-27 19:49:13 +0200 CEST
// 	avgBlockSec := 13

// 	secDiff := referenceBlockTimestamp - int(targetTime.Unix())
// 	blocksDiff := secDiff / avgBlockSec
// 	targetBlock := referenceBlockNumber - blocksDiff
// 	return targetBlock
// }

// func testTimestamp() {
// 	dayStr := "2021-01-27"
// 	hour := 17 // 0:00 - 0:59

// 	targetBlockNumber := getTargetBlocknumber(dayStr, hour)
// 	fmt.Println("target block", targetBlockNumber)
// }

func main() {
	// test2()
	// testTimestamp()

	val := "000000000000000000000000000000000000000000000000010bc88b70c96000"
	n, err := strconv.ParseUint(val, 16, 64)
	if err != nil {
		panic(err)
	}
	fmt.Println(n)

	n2 := new(big.Int)
	n2.SetString(val, 16)
	fmt.Println(n2.Text(10))
}

func printSlice(s string, x []int) {
	fmt.Printf("%s len=%d cap=%d %v\n", s, len(x), cap(x), x)
}
