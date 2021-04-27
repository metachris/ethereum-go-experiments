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

func getTargetBlocknumber(dayString string, hour int) int64 {
	// Create a time.Time instance
	targetTime := makeTime(dayString, hour)
	fmt.Println(targetTime, targetTime.Unix())

	// Calculate block-difference from reference block
	referenceBlockNumber := 12323940
	referenceBlockTimestamp := 1619546404 // 2021-04-27 19:49:13 +0200 CEST
	avgBlockSec := 13

	secDiff := referenceBlockTimestamp - int(targetTime.Unix())
	blocksDiff := secDiff / avgBlockSec
	targetBlock := referenceBlockNumber - blocksDiff
	return int64(targetBlock)
}

func testTimestamp() {
	dayStr := "2021-01-27"
	hour := 17 // 0:00 - 0:59

	targetBlockNumber := getTargetBlocknumber(dayStr, hour)
	fmt.Println("target block", targetBlockNumber)
}
