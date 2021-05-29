package ethstats

import (
	"fmt"
	"log"
	"math/big"
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

    NumBlocks           integer NOT NULL,
    NumBlocksWithoutTx  integer NOT NULL,

    GasUsed             NUMERIC(48, 0) NOT NULL,
    GasFeeTotal         NUMERIC(48, 0) NOT NULL,
    GasFeeFailedTx      NUMERIC(48, 0) NOT NULL,

    NumTransactions                  integer NOT NULL,
    NumTransactionsFailed            integer NOT NULL,
    NumTransactionsWithZeroValue     integer NOT NULL,
    NumTransactionsWithData          integer NOT NULL,

    NumTransactionsErc20Transfer     integer NOT NULL,
    NumTransactionsErc721Transfer    integer NOT NULL,

    NumFlashbotsTransactionsSuccess   integer NOT NULL,
    NumFlashbotsTransactionsFailed    integer NOT NULL,

    ValueTotalEth     NUMERIC(24, 8) NOT NULL,
	TotalAddresses    integer NOT NULL
);

CREATE TABLE IF NOT EXISTS analysis_address_stat (
    Id          int GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,

    Analysis_id int REFERENCES analysis (id) NOT NULL,
    Address     text REFERENCES address (address) NOT NULL,

	NumTxSentSuccess       int NOT NULL,
	NumTxSentFailed        int NOT NULL,
	NumTxReceivedSuccess   int NOT NULL,
	NumTxReceivedFailed    int NOT NULL,

	NumTxFlashbotsSent       int NOT NULL,
	NumTxFlashbotsReceived   int NOT NULL,
	NumTxWithDataSent        int NOT NULL,
	NumTxWithDataReceived    int NOT NULL,

	NumTxErc20Sent         int NOT NULL,
	NumTxErc721Sent        int NOT NULL,
	NumTxErc20Received     int NOT NULL,
	NumTxErc721Received    int NOT NULL,
	NumTxErc20Transfer     int NOT NULL,
	NumTxErc721Transfer    int NOT NULL,

	ValueSentEth       NUMERIC(32, 8) NOT NULL,
	ValueReceivedEth   NUMERIC(32, 8) NOT NULL,

	Erc20TokensTransferred  NUMERIC(48, 0) NOT NULL,
    TokensTransferredInUnit  NUMERIC(56, 8) NOT NULL,
    TokensTransferredSymbol  text NOT NULL,

	GasUsed          NUMERIC(48, 0) NOT NULL,
	GasFeeTotal      NUMERIC(48, 0) NOT NULL,
	GasFeeFailedTx   NUMERIC(48, 0) NOT NULL
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

	NumBlocks          int
	NumBlocksWithoutTx int

	GasUsed        string
	GasFeeTotal    string
	GasFeeFailedTx string

	NumTransactions              int
	NumTransactionsFailed        int
	NumTransactionsWithZeroValue int
	NumTransactionsWithData      int

	NumTransactionsErc20Transfer  int
	NumTransactionsErc721Transfer int

	NumFlashbotsTransactionsSuccess int
	NumFlashbotsTransactionsFailed  int

	ValueTotalEth  string
	TotalAddresses int

	// Calculated after fetching from DB:
	GasFeeTotalEth    string
	GasFeeFailedTxEth string
}

func (entry *AnalysisEntry) CalcNumbers() {
	gasFeeTotal := new(big.Int)
	gasFeeTotal.SetString(entry.GasFeeTotal, 10)
	entry.GasFeeTotalEth = WeiBigIntToEthString(gasFeeTotal, 2)

	gasFeeFailed := new(big.Int)
	gasFeeFailed.SetString(entry.GasFeeFailedTx, 10)
	entry.GasFeeFailedTxEth = WeiBigIntToEthString(gasFeeFailed, 2)

	val := new(big.Float)
	val.SetString(entry.ValueTotalEth)
	entry.ValueTotalEth = BigFloatToHumanNumberString(val, 2)
}

type AnalysisAddressStatsEntryWithAddress struct {
	Id int

	Analysis_id int
	Address     string

	NumTxSentSuccess     int
	NumTxSentFailed      int
	NumTxReceivedSuccess int
	NumTxReceivedFailed  int

	NumTxFlashbotsSent     int
	NumTxFlashbotsReceived int
	NumTxWithDataSent      int
	NumTxWithDataReceived  int

	NumTxErc20Sent      int
	NumTxErc721Sent     int
	NumTxErc20Received  int
	NumTxErc721Received int
	NumTxErc20Transfer  int
	NumTxErc721Transfer int

	ValueSentEth     string
	ValueReceivedEth string

	Erc20TokensTransferred  string
	TokensTransferredInUnit string
	TokensTransferredSymbol string

	GasUsed        string
	GasFeeTotal    string
	GasFeeFailedTx string

	// Fields from joined address
	Type     AddressType
	Name     string
	Symbol   string
	Decimals uint8
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

func AddAddressStatsToDatabase(db *sqlx.DB, client *ethclient.Client, analysisId int, addr AddressStats) {
	_addr := strings.ToLower(addr.Address)

	// Ensure any address is added only once per analysis
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM analysis_address_stat WHERE analysis_id = $1 AND address = $2", analysisId, _addr).Scan(&count)
	Perror(err)
	if count > 0 {
		return
	}

	// Make sure address details are loaded
	addr.EnsureAddressDetails(client)
	detail := addr.AddressDetail

	// fmt.Println("+ stats:", addr)
	addrFromDb, foundInDb := GetAddressFromDatabase(db, addr.Address)
	if !foundInDb { // Not in DB, add now
		_, err = db.Exec("INSERT INTO address (address, name, type, symbol, decimals) VALUES ($1, $2, $3, $4, $5)", strings.ToLower(detail.Address), detail.Name, detail.Type, detail.Symbol, detail.Decimals)
		if err != nil {
			fmt.Println("Error inserting address", _addr, strings.ToLower(detail.Address))
			panic(err)
		}

	} else if addrFromDb.Name == "" { // in DB but without information. If we have more infos now then update
		db.MustExec("UPDATE address set name=$2, type=$3, symbol=$4, decimals=$5 WHERE address=$1", strings.ToLower(detail.Address), detail.Name, detail.Type, detail.Symbol, detail.Decimals)
	}

	// Add address-stats entry now
	tokensTransferredInUnit, tokenSymbol := GetErc20TokensInUnit(addr.Erc20TokensTransferred, addr.AddressDetail)
	_, err = db.Exec(`INSERT INTO analysis_address_stat (
		Analysis_id,
		Address,

		NumTxSentSuccess,
		NumTxSentFailed,
		NumTxReceivedSuccess,
		NumTxReceivedFailed,

		NumTxFlashbotsSent,
		NumTxFlashbotsReceived,
		NumTxWithDataSent,
		NumTxWithDataReceived,

		NumTxErc20Sent,
		NumTxErc721Sent,
		NumTxErc20Received,
		NumTxErc721Received,
		NumTxErc20Transfer,
		NumTxErc721Transfer,

		ValueSentEth,
		ValueReceivedEth,

		Erc20TokensTransferred,
		TokensTransferredInUnit,
		TokensTransferredSymbol,

		GasUsed,
		GasFeeTotal,
		GasFeeFailedTx
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)`,
		analysisId,
		_addr,

		addr.NumTxSentSuccess,
		addr.NumTxSentFailed,
		addr.NumTxReceivedSuccess,
		addr.NumTxReceivedFailed,

		addr.NumTxFlashbotsSent,
		addr.NumTxFlashbotsReceived,
		addr.NumTxWithDataSent,
		addr.NumTxWithDataReceived,

		addr.NumTxErc20Sent,
		addr.NumTxErc721Sent,
		addr.NumTxErc20Received,
		addr.NumTxErc721Received,
		addr.NumTxErc20Transfer,
		addr.NumTxErc721Transfer,

		WeiToEth(addr.ValueSentWei).Text('f', 4),
		WeiToEth(addr.ValueReceivedWei).Text('f', 4),

		addr.Erc20TokensTransferred.String(),
		tokensTransferredInUnit.Text('f', 8),
		tokenSymbol,

		addr.GasUsed.String(),
		addr.GasFeeTotal.String(),
		addr.GasFeeFailedTx.String())

	if err != nil {
		log.Println("database: Error inserting stats for", addr)
	}
}

func AddAnalysisResultToDatabase(db *sqlx.DB, client *ethclient.Client, date string, hour int, minute int, sec int, durationSec int, result *AnalysisResult) {
	// Insert Analysis
	analysisId := 0
	// fmt.Println("valTotal", result.ValueTotalEth)
	err := db.QueryRow(`
		INSERT INTO analysis (
			date, hour, minute, sec, durationsec,
			StartBlockNumber, StartBlockTimestamp, EndBlockNumber, EndBlockTimestamp,

			NumBlocks,
			NumBlocksWithoutTx,

			GasUsed,
			GasFeeTotal,
			GasFeeFailedTx,

			NumTransactions,
			NumTransactionsFailed,
			NumTransactionsWithZeroValue,
			NumTransactionsWithData,

			NumTransactionsErc20Transfer,
			NumTransactionsErc721Transfer,

			NumFlashbotsTransactionsSuccess,
			NumFlashbotsTransactionsFailed,

			ValueTotalEth,
			TotalAddresses
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)
		RETURNING id
		`, date, hour, minute, sec, durationSec,
		result.StartBlockNumber, result.StartBlockTimestamp, result.EndBlockNumber, result.EndBlockTimestamp,
		result.NumBlocks, result.NumBlocksWithoutTx,

		result.GasUsed.String(),
		result.GasFeeTotal.String(),
		result.GasFeeFailedTx.String(),

		result.NumTransactions,
		result.NumTransactionsFailed,
		result.NumTransactionsWithZeroValue,
		result.NumTransactionsWithData,

		result.NumTransactionsErc20Transfer,
		result.NumTransactionsErc721Transfer,

		result.NumFlashbotsTransactionsSuccess,
		result.NumFlashbotsTransactionsFailed,

		WeiToEth(result.ValueTotalWei).Text('f', 2),
		len(result.Addresses)).Scan(&analysisId)
	Perror(err)
	fmt.Println("Analysis ID:", analysisId)

	// Setup DB worker pool
	numWorkers := 10

	var wg sync.WaitGroup // for waiting until all blocks are written into DB
	addressInfoQueue := make(chan AddressStats, 50)
	saveAddrStatToDbWorker := func() {
		defer wg.Done()
		for addr := range addressInfoQueue {
			AddAddressStatsToDatabase(db, client, analysisId, addr)
		}
	}

	// Start workers for adding address-stats to database (using 1 sqlx instance)
	for w := 1; w <= numWorkers; w++ {
		wg.Add(1)
		go saveAddrStatToDbWorker()
	}

	entries := result.GetAllTopAddressStats()
	fmt.Printf("Saving %d AddresseStats...\n", len(entries))
	for _, addr := range entries {
		addressInfoQueue <- addr
	}

	fmt.Println("Waiting for db workers to finish...")
	close(addressInfoQueue)
	wg.Wait()
}

func DbGetAnalysisById(db *sqlx.DB, analysisId int) (entry AnalysisEntry, found bool) {
	err := db.Get(&entry, "SELECT * FROM analysis WHERE id=$1", analysisId)
	if err != nil {
		// fmt.Println(err)
		return entry, false
	}
	return entry, true
}

// func DbGetAnalysesByDate(db *sqlx.DB, date string) (analyses []AnalysisEntry, found bool) {
// 	err := db.Select(&analyses, "SELECT * FROM analysis WHERE date=$1", date)
// 	if err != nil {
// 		fmt.Println(err)
// 		return analyses, false
// 	}
// 	return analyses, true
// }

func DbGetAddressStatEntriesForAnalysisId(db *sqlx.DB, analysisId int) (entries *[]AnalysisAddressStatsEntryWithAddress, err error) {
	// entries := make([]AnalysisAddressStatsEntry, 0)
	rows, err := db.Queryx("SELECT * FROM analysis_address_stat INNER JOIN address ON (address.address = analysis_address_stat.address) WHERE Analysis_id=$1", analysisId)
	if err != nil {
		fmt.Println(err)
		return entries, err
	}

	_entries := make([]AnalysisAddressStatsEntryWithAddress, 0)
	for rows.Next() {
		var row AnalysisAddressStatsEntryWithAddress
		err = rows.StructScan(&row)
		Perror(err)
		_entries = append(_entries, row)
	}

	return &_entries, nil
}
