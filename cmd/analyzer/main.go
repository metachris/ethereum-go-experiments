package main

import (
	"flag"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/metachris/ethereum-go-experiments/consts"
	"github.com/metachris/ethereum-go-experiments/core"
	"github.com/metachris/ethereum-go-experiments/ethstats"
	"github.com/metachris/go-ethutils/addressdetail"
	"github.com/metachris/go-ethutils/utils"
)

func main() {
	timestampMainStart := time.Now() // for measuring execution time

	datePtr := flag.String("date", "", "date (yyyy-mm-dd or -1d)")
	hourPtr := flag.Int("hour", 0, "hour (UTC)")
	minPtr := flag.Int("min", 0, "hour (UTC)")
	lenPtr := flag.String("len", "", "num blocks or timespan (4s, 5m, 1h, ...)")
	// outJsonPtr := flag.String("out", "", "filename to store JSON output")
	blockHeightPtr := flag.Int("block", 0, "specific block to check")
	// addToDbPtr := flag.Bool("addDb", false, "add to database")
	flag.Parse()

	if len(*datePtr) == 0 && *blockHeightPtr == 0 {
		log.Fatal("Date or block missing, add with -date <yyyy-mm-dd> or -block <blockNum>")
	}

	fmt.Println("Connecting to Ethereum node at", core.Cfg.EthNode)
	client, err := ethclient.Dial(core.Cfg.EthNode)
	utils.Perror(err)
	startBlock, endBlock, err := utils.FindBlockRange(client, *blockHeightPtr, *datePtr, *hourPtr, *minPtr, *lenPtr)
	utils.Perror(err)
	fmt.Printf("Checking blocks %d to %d...\n", startBlock, endBlock)

	analysis := ethstats.AnalyzeBlocks(client, startBlock, endBlock)
	if !core.Cfg.HideOutput {
		fmt.Printf("\n===================\n  ANALYSIS RESULT  \n===================\n\n")
		printResult(analysis)
	}

	timeNeeded := time.Since(timestampMainStart)
	fmt.Printf("\nAnalysis of %s blocks, %s transactions finished in %.2fs\n", utils.NumberToHumanReadableString(analysis.Data.NumBlocks, 0), utils.NumberToHumanReadableString(analysis.Data.NumTransactions, 0), timeNeeded.Seconds())

	// if *addToDbPtr {
	// 	// Add to database
	// 	fmt.Printf("\nSaving to database...\n")
	// 	timeStartAddToDb := time.Now()
	// 	db.AddAnalysisResultToDatabase(analysis)
	// 	timeNeededAddToDb := time.Since(timeStartAddToDb)
	// 	fmt.Printf("Saved to database (%.2fs)\n", timeNeededAddToDb.Seconds())
	// }

	// if len(*outJsonPtr) > 0 {
	// 	j, err := json.MarshalIndent(exportData, "", " ")
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	outFn := *outJsonPtr
	// 	_ = ioutil.WriteFile(outFn, j, 0644)
	// 	fmt.Println("Saved to " + *outJsonPtr)
	// }
}

func printH1(msg string) {
	m := strings.Trim(msg, "\n")
	fmt.Println(strings.Repeat("=", utf8.RuneCountInString(m)))
	fmt.Println(m)
	fmt.Println(strings.Repeat("=", utf8.RuneCountInString(m)))
}

func printH2(msg string) {
	m := strings.Trim(msg, "\n")
	fmt.Println(msg)
	fmt.Println(strings.Repeat("-", utf8.RuneCountInString(m)))
}

func printTopTx(msg string, txList []core.TxStats) {
	printH2(msg)
	for _, v := range txList {
		fmt.Println(v)
	}
}

func AddressWithName(addressDetail addressdetail.AddressDetail) string {
	return fmt.Sprintf("%s %-28s", addressDetail.Address, addressDetail.Name)
}

func printTopAddr(msg string, list []core.AddressStats, max int) {
	printH2(msg)
	for i, v := range list {
		fmt.Printf("%-66v txInOk: %7d  \t  txOutOk: %7d \t txInFail: %5d \t txOutFail: %5d \t %10v ETH received \t %10v ETH sent \t gasFee %v ETH / for failed: %v ETH \n", AddressWithName(v.AddressDetail), v.Get(consts.NumTxReceivedSuccess), v.Get(consts.NumTxSentSuccess), v.Get(consts.NumTxReceivedFailed), v.Get(consts.NumTxSentFailed), utils.WeiBigIntToEthString(v.Get(consts.ValueReceivedWei), 2), utils.WeiBigIntToEthString(v.Get(consts.ValueSentWei), 2), utils.WeiBigIntToEthString(v.Get(consts.GasFeeTotal), 2), utils.WeiBigIntToEthString(v.Get(consts.GasFeeFailedTx), 2))
		// fmt.Printf("%-66v \n", addressWithName(v.AddressDetail.Address))
		if max > 0 && i == max {
			break
		}
	}
}

// Processes a raw result into the export data structure, and prints the stats to stdout
func printResult(analysis *core.Analysis) {
	fmt.Println("Total blocks:", utils.NumberToHumanReadableString(analysis.Data.NumBlocks, 0))
	fmt.Println("- without tx:", utils.NumberToHumanReadableString(analysis.Data.NumBlocksWithoutTx, 0))
	fmt.Println("")
	fmt.Println("Total transactions:", utils.NumberToHumanReadableString(analysis.Data.NumTransactions, 0), "\t tx-types:", analysis.Data.TxTypes)
	fmt.Printf("- failed:         %7s \t %.2f%%\n", utils.NumberToHumanReadableString(analysis.Data.NumTransactionsFailed, 0), (float64(analysis.Data.NumTransactionsFailed)/float64(analysis.Data.NumTransactions))*100)
	fmt.Printf("- with value:     %7s \t %.2f%%\n", utils.NumberToHumanReadableString(analysis.Data.NumTransactions-analysis.Data.NumTransactionsWithZeroValue, 0), (float64((analysis.Data.NumTransactions-analysis.Data.NumTransactionsWithZeroValue))/float64(analysis.Data.NumTransactions))*100)
	fmt.Printf("- zero value:     %7s \t %.2f%%\n", utils.NumberToHumanReadableString(analysis.Data.NumTransactionsWithZeroValue, 0), (float64(analysis.Data.NumTransactionsWithZeroValue)/float64(analysis.Data.NumTransactions))*100)
	fmt.Printf("- with data:      %7s \t %.2f%%\n", utils.NumberToHumanReadableString(analysis.Data.NumTransactionsWithData, 0), (float64(analysis.Data.NumTransactionsWithData)/float64(analysis.Data.NumTransactions))*100)
	fmt.Printf("- erc20 transfer: %7s \t %.2f%%\n", utils.NumberToHumanReadableString(analysis.Data.NumTransactionsErc20Transfer, 0), (float64(analysis.Data.NumTransactionsErc20Transfer)/float64(analysis.Data.NumTransactions))*100)
	fmt.Printf("- erc721 transfer:%7s \t %.2f%%\n", utils.NumberToHumanReadableString(analysis.Data.NumTransactionsErc721Transfer, 0), (float64(analysis.Data.NumTransactionsErc721Transfer)/float64(analysis.Data.NumTransactions))*100)
	fmt.Printf("- flashbots:       %s ok, %s failed \n", utils.NumberToHumanReadableString(analysis.Data.NumFlashbotsTransactionsSuccess, 0), utils.NumberToHumanReadableString(analysis.Data.NumFlashbotsTransactionsFailed, 0))
	fmt.Println("")

	fmt.Println("Total addresses:", utils.NumberToHumanReadableString(len(analysis.Addresses), 0))
	fmt.Println("Total value transferred:", utils.WeiBigIntToEthString(analysis.Data.ValueTotalWei, 2), "ETH")
	fmt.Println("Total gas fees:", utils.WeiBigIntToEthString(analysis.Data.GasFeeTotal, 2), "ETH")
	fmt.Println("Gas for failed tx:", utils.WeiBigIntToEthString(analysis.Data.GasFeeFailedTx, 2), "ETH")

	if core.Cfg.HideOutput {
		fmt.Println("End because of config.HideOutput")
		return
	}

	fmt.Println("")
	printH1("Transactions")
	printTopTx("\nTop transactions by GAS FEE", analysis.Data.TopTransactions.GasFee)
	printTopTx("\nTop transactions by ETH VALUE", analysis.Data.TopTransactions.Value)
	printTopTx("\nTop transactions by MOST DATA", analysis.Data.TopTransactions.DataSize)
	printTopTx("\nTagged transactions", analysis.Data.TaggedTransactions)

	fmt.Println("")
	printH1("\nSmart Contracts")

	printH2("\nERC20: most token-tranfer tx")
	for _, v := range analysis.Data.TopAddresses[consts.NumTxErc20Transfer] {
		tokensTransferredInUnit, tokenSymbol := utils.GetErc20TokensInUnit(v.Get(consts.Erc20TokensTransferred), v.AddressDetail)
		tokenAmount := fmt.Sprintf("%s %-5v", formatBigFloat(tokensTransferredInUnit), tokenSymbol)
		fmt.Printf("%s \t %8d erc20-tx \t %8d tx \t %32v\n", AddressWithName(v.AddressDetail), v.Get(consts.NumTxErc20Transfer), v.Get(consts.NumTxReceivedSuccess), tokenAmount)
	}

	printH2("\nERC721: most token transfers")
	for _, v := range analysis.Data.TopAddresses["NumTxErc721Transfer"] {
		fmt.Printf("%-100s \t %8d erc721-tx \t %8d tx\n", AddressWithName(v.AddressDetail), v.Get(consts.NumTxErc721Transfer), v.Get(consts.NumTxReceivedSuccess))
	}

	fmt.Println("")
	printH1("\nAddresses")

	interesting := [...]string{consts.NumTxReceivedSuccess, consts.NumTxSentSuccess, consts.ValueReceivedWei, consts.ValueSentWei, consts.NumTxReceivedFailed, consts.NumTxSentFailed, consts.FlashBotsFailedTxSent}
	for _, key := range interesting {
		fmt.Println("")
		printTopAddr(key, analysis.Data.TopAddresses[key], 0)
	}

	// for _, k := range consts.StatsKeys {
	// 	// fmt.Println(k, len(v))
	// 	// if strings.Contains(k, "721") {
	// 	fmt.Println("")
	// 	printTopAddr(k, result.TopAddresses[k], 0)
	// 	// }
	// }
}

func formatBigFloat(number *big.Float) string {
	output := number.Text('f', 0)
	startOffset := 3
	if number.Cmp(big.NewFloat(0)) == -1 { // negative number, starts with -
		startOffset++
	}
	for outputIndex := len(output); outputIndex > startOffset; {
		outputIndex -= 3
		output = output[:outputIndex] + "," + output[outputIndex:]
	}
	return output
}
