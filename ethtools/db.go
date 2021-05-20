package ethtools

import (
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var schema = `
CREATE TABLE IF NOT EXISTS address (
    Address  text NOT NULL,
    Name     text NOT NULL,
    Type     text NOT NULL,
    Symbol   text NOT NULL,
    Decimals integer NOT NULL,

    PRIMARY KEY(address)
);

CREATE TABLE IF NOT EXISTS analysis (
    Id          int GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,

    Date        text NOT NULL,
    Hour        integer NOT NULL,
    Minute      integer NOT NULL,
    Sec         integer NOT NULL,
    DurationSec integer NOT NULL,

    StartBlockNumber    integer NOT NULL,
    StartBlockTimestamp integer NOT NULL,
    EndBlockNumber      integer NOT NULL,
    EndBlockTimestamp   integer NOT NULL,

    NumBlocks                        integer NOT NULL,
    NumBlocksWithoutTx               integer NOT NULL,
    NumTransactions                  integer NOT NULL,
    NumTransactionsFailed            integer NOT NULL,
    NumTransactionsWithZeroValue     integer NOT NULL,
    NumTransactionsWithData          integer NOT NULL,
    NumTransactionsWithTokenTransfer integer NOT NULL,
    TotalAddresses                   integer NOT NULL,

    ValueTotalEth   NUMERIC(24, 8) NOT NULL,
    GasUsed         bigint NOT NULL
);

CREATE TABLE IF NOT EXISTS analysis_address_stat (
    Id          int GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,

    Analysis_id int REFERENCES analysis (id) NOT NULL,
    Address     text REFERENCES address (address) NOT NULL,

	NumTxSent                    int NOT NULL,
	NumTxReceived                int NOT NULL,
	NumFailedTxSent              int NOT NULL,
	NumFailedTxReceived          int NOT NULL,
	GasUsed                      bigint NOT NULL,

	NumTxWithData                int NOT NULL,
	NumTxTokenTransfer           int NOT NULL,
	NumTxTokenMethodTransfer     int NOT NULL,
	NumTxTokenMethodTransferFrom int NOT NULL,

	ValueSentEth       NUMERIC(32, 8) NOT NULL,
	ValueReceivedEth   NUMERIC(32, 8) NOT NULL,
	TokensTransferred  NUMERIC(48, 0) NOT NULL,

    TokensTransferredInUnit  NUMERIC(56, 8) NOT NULL,
    TokensTransferredSymbol  text NOT NULL
);

CREATE TABLE IF NOT EXISTS block (
	Number    int,
	Time      int,
	NumTx     int,
	GasUsed   int,
	GasLimit  int,

	PRIMARY KEY (Number)
)
`

type AnalysisEntry struct {
	Id          int
	Date        string
	Hour        int
	Minute      int
	Sec         int
	DurationSec int

	StartBlockNumber    int
	StartBlockTimestamp int
	EndBlockNumber      int
	EndBlockTimestamp   int

	NumBlocks                        int
	NumBlocksWithoutTx               int
	NumTransactions                  int
	NumTransactionsFailed            int
	NumTransactionsWithZeroValue     int
	NumTransactionsWithData          int
	NumTransactionsWithTokenTransfer int
	TotalAddresses                   int

	ValueTotalEth string
	GasUsed       uint64
}

var db *sqlx.DB

// GetDatabase returns an already existing DB connection. If not exists then creates it
func GetDatabase(cfg PostgresConfig) *sqlx.DB {
	if db != nil {
		return db
	}
	db = NewDatabaseConnection(cfg)
	return db
}

func DropDatabase(db *sqlx.DB) {
	db.MustExec(`DROP TABLE "analysis_address_stat";`)
	db.MustExec(`DROP TABLE "analysis";`)
	db.MustExec(`DROP TABLE "address";`)
	db.MustExec(`DROP TABLE "block";`)
}

func ResetDatabase(db *sqlx.DB) {
	DropDatabase(db)
	db.MustExec(schema)
}

// NewDatabaseConnection creates a new sqlx.DB database connection
func NewDatabaseConnection(cfg PostgresConfig) *sqlx.DB {
	sslMode := "require"
	if cfg.DisableTLS {
		sslMode = "disable"
	}

	q := make(url.Values)
	q.Set("sslmode", sslMode)
	q.Set("timezone", "utc")

	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.User, cfg.Password),
		Host:     cfg.Host,
		Path:     cfg.Name,
		RawQuery: q.Encode(),
	}

	db := sqlx.MustConnect("postgres", u.String())
	db.MustExec(schema)
	return db
}

func GetAddressFromDatabase(db *sqlx.DB, address string) (addr AddressDetail, found bool) {
	err := db.Get(&addr, "SELECT * FROM address WHERE address=$1", strings.ToLower(address))
	if err != nil {
		// fmt.Println("err:", err, addr)
		return addr, false
	}
	return addr, true
}

func AddAddressesFromJsonToDatabase(db *sqlx.DB) {
	addressMap := GetAddressDetailMap(DATASET_BOTH)

	for _, v := range addressMap {
		fmt.Printf("%s \t %-10v \t %-30v %s \t %d\n", v.Address, v.Type, v.Name, v.Symbol, v.Decimals)

		// Check if exists
		_, found := GetAddressFromDatabase(db, v.Address)
		if found {
			fmt.Println("- already in DB")
			continue
		}

		db.MustExec("INSERT INTO address (address, name, type, symbol, decimals) VALUES ($1, $2, $3, $4, $5)", strings.ToLower(v.Address), v.Name, v.Type, v.Symbol, v.Decimals)
	}
}

func AddBlockToDatabase(db *sqlx.DB, block *types.Block) {
	// Check count
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM block WHERE number = $1", block.Header().Number.Int64()).Scan(&count)
	Perror(err)
	if count > 0 {
		return
	}

	// Insert
	db.MustExec("INSERT INTO block (Number, Time, NumTx, GasUsed, GasLimit) VALUES ($1, $2, $3, $4, $5)", block.Number().String(), block.Header().Time, len(block.Transactions()), block.GasUsed(), block.GasLimit())
}

func AddAnalysisResultToDatabase(db *sqlx.DB, client *ethclient.Client, date string, hour int, minute int, sec int, durationSec int, result *AnalysisResult) {
	// Insert Analysis
	analysisId := 0
	err := db.QueryRow(
		"INSERT INTO analysis (date, hour, minute, sec, durationsec, StartBlockNumber, StartBlockTimestamp, EndBlockNumber, EndBlockTimestamp, ValueTotalEth, NumBlocks, NumBlocksWithoutTx, NumTransactions, NumTransactionsFailed, NumTransactionsWithZeroValue, NumTransactionsWithData, NumTransactionsWithTokenTransfer, TotalAddresses, GasUsed) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19) RETURNING id",
		date, hour, minute, sec, durationSec, result.StartBlockNumber, result.StartBlockTimestamp, result.EndBlockNumber, result.EndBlockTimestamp, result.ValueTotalEth, result.NumBlocks, result.NumBlocksWithoutTx, result.NumTransactions, result.NumTransactionsFailed, result.NumTransactionsWithZeroValue, result.NumTransactionsWithData, result.NumTransactionsErc20Transfer, len(result.Addresses), result.GasUsed).Scan(&analysisId)
	Perror(err)
	fmt.Println("Analysis ID:", analysisId)

	addAddressAndStats := func(addr AddressStats) {
		// fmt.Println("+ stats:", addr)
		addrFromDb, foundInDb := GetAddressFromDatabase(db, addr.Address)
		if !foundInDb { // Not in DB, add now
			detail := GetAddressDetail(addr.Address, client)
			db.MustExec("INSERT INTO address (address, name, type, symbol, decimals) VALUES ($1, $2, $3, $4, $5)", strings.ToLower(detail.Address), detail.Name, detail.Type, detail.Symbol, detail.Decimals)

		} else if addrFromDb.Name == "" { // in DB but without information. If we have more infos now then update
			detail := GetAddressDetail(addr.Address, client)
			db.MustExec("UPDATE address set name=$2, type=$3, symbol=$4, decimals=$5 WHERE address=$1", strings.ToLower(detail.Address), detail.Name, detail.Type, detail.Symbol, detail.Decimals)
		}

		// Avoid adding one address multiple times per analysis
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM analysis_address_stat WHERE analysis_id = $1 AND address = $2", analysisId, strings.ToLower(addr.Address)).Scan(&count)
		Perror(err)
		if count > 0 {
			return
		}

		// Add address-stats entry now
		// valRecEth := WeiBigIntToEthString(addr.ValueReceivedWei, 2)
		// valSentEth := WeiBigIntToEthString(addr.ValueSentWei, 2)
		// tokensTransferredInUnit, tokenSymbol := addr.TokensTransferredInUnit(client)
		// db.MustExec("INSERT INTO analysis_address_stat (analysis_id, address, numtxsent, numtxreceived, NumFailedTxSent, NumFailedTxReceived, NumTxWithData, NumTxTokenTransfer, NumTxTokenMethodTransfer, NumTxTokenMethodTransferFrom, valuesenteth, valuereceivedeth, tokenstransferred, tokenstransferredinunit, tokenstransferredsymbol, GasUsed) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)",
		// 	analysisId, strings.ToLower(addr.Address), addr.NumTxSentSuccess, addr.NumTxReceivedSuccess, addr.NumTxSentFailed, addr.NumFailedTxReceived, addr.NumTxWithData, addr.NumTxTokenTransfer, addr.NumTxTokenMethodTransfer, addr.NumTxTokenMethodTransferFrom, valSentEth, valRecEth, addr.TokensTransferred.String(), tokensTransferredInUnit.Text('f', 8), tokenSymbol, addr.GasUsed)
	}

	// Setup worker pool
	var wg sync.WaitGroup // for waiting until all blocks are written into DB
	addressInfoQueue := make(chan AddressStats, 50)
	saveAddrStatToDbWorker := func() {
		defer wg.Done()
		for addr := range addressInfoQueue {
			addAddressAndStats(addr)
		}
	}

	// Start workers for adding address-stats to database (using 1 sqlx instance)
	numWorkers := 10
	for w := 1; w <= numWorkers; w++ {
		wg.Add(1)
		go saveAddrStatToDbWorker()
	}

	fmt.Println("Saving addresses and stats...")
	// fmt.Printf("- Tokens transferred: %d\n", len(result.TopAddresses.NumTokenTransfers))
	// for _, addr := range result.TopAddresses.NumTokenTransfers {
	// 	addressInfoQueue <- addr
	// }

	// fmt.Printf("- Num tx received: %d\n", len(result.TopAddresses.NumTxReceived))
	// for _, addr := range result.TopAddresses.NumTxReceived {
	// 	addressInfoQueue <- addr
	// }

	// fmt.Printf("- Num tx sent: %d\n", len(result.TopAddresses.NumTxSent))
	// for _, addr := range result.TopAddresses.NumTxSent {
	// 	addressInfoQueue <- addr
	// }

	// fmt.Printf("- Value received: %d\n", len(result.TopAddresses.ValueReceived))
	// for _, addr := range result.TopAddresses.ValueReceived {
	// 	addressInfoQueue <- addr
	// }

	// fmt.Printf("- Value sent: %d\n", len(result.TopAddresses.ValueSent))
	// for _, addr := range result.TopAddresses.ValueSent {
	// 	addressInfoQueue <- addr
	// }

	// fmt.Printf("- Failed tx received: %d\n", len(result.TopAddresses.NumFailedTxReceived))
	// for _, addr := range result.TopAddresses.NumFailedTxReceived {
	// 	addressInfoQueue <- addr
	// }

	// fmt.Printf("- Failed tx sent: %d\n", len(result.TopAddresses.NumFailedTxSent))
	// for _, addr := range result.TopAddresses.NumFailedTxSent {
	// 	addressInfoQueue <- addr
	// }

	fmt.Println("Waiting for workers to finish...")
	close(addressInfoQueue)
	wg.Wait()
}

func DbGetAnalysisById(db *sqlx.DB, analysisId int) (entry AnalysisEntry, found bool) {
	err := db.Get(&entry, "SELECT * FROM analysis WHERE id=$1", analysisId)
	if err != nil {
		return entry, false
	}
	return entry, true
}

func DbGetAnalysesByDate(db *sqlx.DB, date string) (analyses []AnalysisEntry, found bool) {
	err := db.Select(&analyses, "SELECT * FROM analysis WHERE date=$1", date)
	if err != nil {
		return analyses, false
	}
	return analyses, true
}
