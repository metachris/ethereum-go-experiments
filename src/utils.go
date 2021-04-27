package main

import (
	"fmt"
	"log"
	"time"
)

func makeTime(dayString string, hour int) time.Time {
	dateString := fmt.Sprintf("%sT%02d:00:00Z", dayString, hour)
	// fmt.Println(ts)
	t, err := time.Parse(time.RFC3339, dateString)
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Println(t)
	return t
}

func getTargetBlocknumber(utcTimestamp int64) int64 {
	// Create a time.Time instance

	// Calculate block-difference from reference block
	referenceBlockNumber := int64(12323940)
	referenceBlockTimestamp := int64(1619546404) // 2021-04-27 19:49:13 +0200 CEST
	avgBlockSec := int64(13)

	secDiff := referenceBlockTimestamp - utcTimestamp
	blocksDiff := secDiff / avgBlockSec
	targetBlock := referenceBlockNumber - blocksDiff
	return targetBlock
}

// func testTimestamp() {
// 	dayStr := "2021-01-27"
// 	hour := 17 // 0:00 - 0:59

// 	targetBlockNumber := getTargetBlocknumber(dayStr, hour)
// 	fmt.Println("target block", targetBlockNumber)
// }

func Abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
