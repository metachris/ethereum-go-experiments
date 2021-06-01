// package ethstats

// import (
// 	"github.com/ethereum/go-ethereum/ethclient"
// 	"github.com/jmoiron/sqlx"
// )

// // func AddAddressesFromJsonToDatabase(db *sqlx.DB) {
// // 	addressMap := GetAddressDetailMap(DATASET_BOTH)

// // 	for _, v := range addressMap {
// // 		fmt.Printf("%s \t %-10v \t %-30v %s \t %d\n", v.Address, v.Type, v.Name, v.Symbol, v.Decimals)

// // 		// Check if exists
// // 		_, found := GetAddressFromDatabase(db, v.Address)
// // 		if found {
// // 			fmt.Println("- already in DB")
// // 			continue
// // 		}

// // 		db.MustExec("INSERT INTO address (address, name, type, symbol, decimals) VALUES ($1, $2, $3, $4, $5)", strings.ToLower(v.Address), v.Name, v.Type, v.Symbol, v.Decimals)
// // 	}
// // }

// func AddAddressStatsToDatabase(db *sqlx.DB, client *ethclient.Client, analysisId int, addr AddressStats) {
// 	// _addr := strings.ToLower(addr.Address)

// 	// // Ensure any address is added only once per analysis
// 	// var count int
// 	// err := db.QueryRow("SELECT COUNT(*) FROM analysis_address_stat WHERE analysis_id = $1 AND address = $2", analysisId, _addr).Scan(&count)
// 	// Perror(err)
// 	// if count > 0 {
// 	// 	return
// 	// }

// 	// // Make sure address details are loaded
// 	// addr.EnsureAddressDetails(client)
// 	// detail := addr.AddressDetail

// 	// // fmt.Println("+ stats:", addr)
// 	// addrFromDb, foundInDb := GetAddressFromDatabase(db, addr.Address)
// 	// if !foundInDb { // Not in DB, add now
// 	// 	_, err = db.Exec("INSERT INTO address (address, name, type, symbol, decimals) VALUES ($1, $2, $3, $4, $5)", strings.ToLower(detail.Address), detail.Name, detail.Type, detail.Symbol, detail.Decimals)
// 	// 	if err != nil {
// 	// 		fmt.Println("Error inserting address", _addr, strings.ToLower(detail.Address))
// 	// 		panic(err)
// 	// 	}

// 	// } else if addrFromDb.Name == "" { // in DB but without information. If we have more infos now then update
// 	// 	db.MustExec("UPDATE address set name=$2, type=$3, symbol=$4, decimals=$5 WHERE address=$1", strings.ToLower(detail.Address), detail.Name, detail.Type, detail.Symbol, detail.Decimals)
// 	// }

// 	// // Add address-stats entry now
// 	// tokensTransferredInUnit, tokenSymbol := GetErc20TokensInUnit(addr.Erc20TokensTransferred, addr.AddressDetail)
// 	// _, err = db.Exec(`INSERT INTO analysis_address_stat (
// 	// 	Analysis_id,
// 	// 	Address,

// 	// 	NumTxSentSuccess,
// 	// 	NumTxSentFailed,
// 	// 	NumTxReceivedSuccess,
// 	// 	NumTxReceivedFailed,

// 	// 	NumTxFlashbotsSent,
// 	// 	NumTxFlashbotsReceived,
// 	// 	NumTxWithDataSent,
// 	// 	NumTxWithDataReceived,

// 	// 	NumTxErc20Sent,
// 	// 	NumTxErc721Sent,
// 	// 	NumTxErc20Received,
// 	// 	NumTxErc721Received,
// 	// 	NumTxErc20Transfer,
// 	// 	NumTxErc721Transfer,

// 	// 	ValueSentEth,
// 	// 	ValueReceivedEth,

// 	// 	Erc20TokensTransferred,
// 	// 	TokensTransferredInUnit,
// 	// 	TokensTransferredSymbol,

// 	// 	GasUsed,
// 	// 	GasFeeTotal,
// 	// 	GasFeeFailedTx
// 	// ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)`,
// 	// 	analysisId,
// 	// 	_addr,

// 	// 	addr.NumTxSentSuccess,
// 	// 	addr.NumTxSentFailed,
// 	// 	addr.NumTxReceivedSuccess,
// 	// 	addr.NumTxReceivedFailed,

// 	// 	addr.NumTxFlashbotsSent,
// 	// 	addr.NumTxFlashbotsReceived,
// 	// 	addr.NumTxWithDataSent,
// 	// 	addr.NumTxWithDataReceived,

// 	// 	addr.NumTxErc20Sent,
// 	// 	addr.NumTxErc721Sent,
// 	// 	addr.NumTxErc20Received,
// 	// 	addr.NumTxErc721Received,
// 	// 	addr.NumTxErc20Transfer,
// 	// 	addr.NumTxErc721Transfer,

// 	// 	WeiToEth(addr.ValueSentWei).Text('f', 4),
// 	// 	WeiToEth(addr.ValueReceivedWei).Text('f', 4),

// 	// 	addr.Erc20TokensTransferred.String(),
// 	// 	tokensTransferredInUnit.Text('f', 8),
// 	// 	tokenSymbol,

// 	// 	addr.GasUsed.String(),
// 	// 	addr.GasFeeTotal.String(),
// 	// 	addr.GasFeeFailedTx.String())

// 	// if err != nil {
// 	// 	log.Println("database: Error inserting stats for", addr)
// 	// }
// }

// // func AddAnalysisResultToDatabase(db *sqlx.DB, client *ethclient.Client, date string, hour int, minute int, sec int, durationSec int, result *AnalysisResult) {
// // 	// Insert Analysis
// // 	analysisId := 0
// // 	// fmt.Println("valTotal", result.ValueTotalEth)
// // 	err := db.QueryRow(`
// // 		INSERT INTO analysis (
// // 			date, hour, minute, sec, durationsec,
// // 			StartBlockNumber, StartBlockTimestamp, EndBlockNumber, EndBlockTimestamp,

// // 			NumBlocks,
// // 			NumBlocksWithoutTx,

// // 			GasUsed,
// // 			GasFeeTotal,
// // 			GasFeeFailedTx,

// // 			NumTransactions,
// // 			NumTransactionsFailed,
// // 			NumTransactionsWithZeroValue,
// // 			NumTransactionsWithData,

// // 			NumTransactionsErc20Transfer,
// // 			NumTransactionsErc721Transfer,

// // 			NumFlashbotsTransactionsSuccess,
// // 			NumFlashbotsTransactionsFailed,

// // 			ValueTotalEth,
// // 			TotalAddresses
// // 		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)
// // 		RETURNING id
// // 		`, date, hour, minute, sec, durationSec,
// // 		result.StartBlockNumber, result.StartBlockTimestamp, result.EndBlockNumber, result.EndBlockTimestamp,
// // 		result.NumBlocks, result.NumBlocksWithoutTx,

// // 		result.GasUsed.String(),
// // 		result.GasFeeTotal.String(),
// // 		result.GasFeeFailedTx.String(),

// // 		result.NumTransactions,
// // 		result.NumTransactionsFailed,
// // 		result.NumTransactionsWithZeroValue,
// // 		result.NumTransactionsWithData,

// // 		result.NumTransactionsErc20Transfer,
// // 		result.NumTransactionsErc721Transfer,

// // 		result.NumFlashbotsTransactionsSuccess,
// // 		result.NumFlashbotsTransactionsFailed,

// // 		WeiToEth(result.ValueTotalWei).Text('f', 2),
// // 		len(result.Addresses)).Scan(&analysisId)
// // 	Perror(err)
// // 	fmt.Println("Analysis ID:", analysisId)

// // 	// Setup DB worker pool
// // 	var wg sync.WaitGroup // for waiting until all blocks are written into DB
// // 	addressInfoQueue := make(chan AddressStats, 50)

// // 	// Start workers for adding address-stats to database (using 1 sqlx instance)
// // 	numWorker := 1
// // 	for w := 1; w <= numWorker; w++ {
// // 		wg.Add(1)
// // 		go func() {
// // 			defer wg.Done()
// // 			for addr := range addressInfoQueue {
// // 				AddAddressStatsToDatabase(db, client, analysisId, addr)
// // 			}
// // 		}()
// // 	}

// // 	entries := result.GetAllTopAddressStats()
// // 	fmt.Printf("Saving %d AddresseStats...\n", len(entries))
// // 	// for _, addr := range entries {
// // 	// 	addressInfoQueue <- addr
// // 	// }

// // 	fmt.Println("Waiting for db workers to finish...")
// // 	close(addressInfoQueue)
// // 	wg.Wait()
// // }
