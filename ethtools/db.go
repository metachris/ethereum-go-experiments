package ethtools

import (
	"fmt"
	"math/big"
	"net/url"
	"sort"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/core/types"
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

    ValueTotalEth                    NUMERIC(24, 8) NOT NULL,
    NumBlocks                        integer NOT NULL,
    NumTransactions                  integer NOT NULL,
    NumTransactionsWithZeroValue     integer NOT NULL,
    NumTransactionsWithData          integer NOT NULL,
    NumTransactionsWithTokenTransfer integer NOT NULL,
    TotalAddresses                   integer NOT NULL
);

CREATE TABLE IF NOT EXISTS analysis_address_stat (
    Id          int GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,

    Analysis_id int REFERENCES analysis (id) NOT NULL,
    Address     text REFERENCES address (address) NOT NULL,

	NumTxSent                    int NOT NULL,
	NumTxReceived                int NOT NULL,
	NumTxWithData                int NOT NULL,
	NumTxTokenTransfer           int NOT NULL,
	NumTxTokenMethodTransfer     int NOT NULL,
	NumTxTokenMethodTransferFrom int NOT NULL,

	ValueSentEth       NUMERIC(24, 8) NOT NULL,
	ValueReceivedEth   NUMERIC(24, 8) NOT NULL,
	TokensTransferred  NUMERIC(32, 0) NOT NULL,

    TokensTransferredInUnit  NUMERIC(48, 8) NOT NULL,
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
	Id                               int
	Date                             string
	Hour                             int
	Minute                           int
	Sec                              int
	DurationSec                      int
	StartBlockNumber                 int
	StartBlockTimestamp              int
	EndBlockNumber                   int
	EndBlockTimestamp                int
	ValueTotalEth                    string
	NumBlocks                        int
	NumTransactions                  int
	NumTransactionsWithZeroValue     int
	NumTransactionsWithData          int
	NumTransactionsWithTokenTransfer int
	TotalAddresses                   int
}

var db *sqlx.DB

// GetDatabase returns an already existing DB connection. If not exists then creates it
func GetDatabase(cfg Config) *sqlx.DB {
	if db != nil {
		return db
	}
	db = NewDatabaseConnection(cfg)
	return db
}

// NewDatabaseConnection creates a new sqlx.DB database connection
func NewDatabaseConnection(cfg Config) *sqlx.DB {
	sslMode := "require"
	dbConfig := cfg.Database
	if dbConfig.DisableTLS {
		sslMode = "disable"
	}

	q := make(url.Values)
	q.Set("sslmode", sslMode)
	q.Set("timezone", "utc")

	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(dbConfig.User, dbConfig.Password),
		Host:     dbConfig.Host,
		Path:     dbConfig.Name,
		RawQuery: q.Encode(),
	}

	db := sqlx.MustConnect("postgres", u.String())
	db.MustExec(schema)
	return db
}

func GetAddressFromDatabase(db *sqlx.DB, address string) (addr AddressDetail, found bool) {
	err := db.Get(&addr, "SELECT * FROM address WHERE address=$1", strings.ToLower(address))
	// fmt.Println("_", err, addr)
	if err != nil {
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
	db.QueryRow("SELECT COUNT(*) FROM block WHERE number = $1", block.Header().Number.Int64()).Scan(&count)
	if count > 0 {
		return
	}

	// Insert
	db.MustExec("INSERT INTO block (Number, Time, NumTx, GasUsed, GasLimit) VALUES ($1, $2, $3, $4, $5)", block.Number().String(), block.Header().Time, len(block.Transactions()), block.GasUsed(), block.GasLimit())
}

func AddAnalysisResultToDatabase(db *sqlx.DB, date string, hour int, minute int, sec int, durationSec int, result *AnalysisResult) {
	// Insert Analysis
	analysisId := 0
	db.QueryRow("INSERT INTO analysis (date, hour, minute, sec, durationsec, StartBlockNumber, StartBlockTimestamp, EndBlockNumber, EndBlockTimestamp, ValueTotalEth, NumBlocks, NumTransactions, NumTransactionsWithZeroValue, NumTransactionsWithData, NumTransactionsWithTokenTransfer, TotalAddresses) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16) RETURNING id",
		date, hour, minute, sec, durationSec, result.StartBlockNumber, result.StartBlockTimestamp, result.EndBlockNumber, result.EndBlockTimestamp, result.ValueTotalEth, result.NumBlocks, result.NumTransactions, result.NumTransactionsWithZeroValue, result.NumTransactionsWithData, result.NumTransactionsWithTokenTransfer, len(result.Addresses)).Scan(&analysisId)
	fmt.Println("Added analysis to database with ID", analysisId)

	// Create addresses array for sorting
	_addresses := make([]AddressInfo, 0, len(result.Addresses))
	for _, k := range result.Addresses {
		_addresses = append(_addresses, *k)
	}

	addAddressAndStats := func(addr AddressInfo) {
		// fmt.Println("+ stats:", addr)
		addrFromDb, foundInDb := GetAddressFromDatabase(db, addr.Address)
		if !foundInDb { // Not in DB, add now
			detail, foundInJson := AllAddressesFromJson[strings.ToLower(strings.ToLower(addr.Address))]
			if !foundInJson {
				detail = AddressDetail{Address: addr.Address}
				if addr.NumTxTokenTransfer > 0 {
					detail.Type = AddressTypeToken
				}
			}
			db.MustExec("INSERT INTO address (address, name, type, symbol, decimals) VALUES ($1, $2, $3, $4, $5)", strings.ToLower(detail.Address), detail.Name, detail.Type, detail.Symbol, detail.Decimals)

		} else if addrFromDb.Name == "" { // in DB but without information. If we have more infos now then update
			detail, foundInJson := AllAddressesFromJson[strings.ToLower(strings.ToLower(addr.Address))]
			if foundInJson {
				db.MustExec("UPDATE address set name=$2, type=$3, symbol=$4, decimals=$5 WHERE address=$1", strings.ToLower(detail.Address), detail.Name, detail.Type, detail.Symbol, detail.Decimals)
			}
		}

		// Avoid adding one address multiple times per analysis
		var count int
		db.QueryRow("SELECT COUNT(*) FROM analysis_address_stat WHERE analysis_id = $1 AND address = $2", analysisId, strings.ToLower(addr.Address)).Scan(&count)
		if count > 0 {
			return
		}

		// Add address-stats entry now
		valRecEth := WeiToEth(addr.ValueReceivedWei).Text('f', 2)
		valSentEth := WeiToEth(addr.ValueSentWei).Text('f', 2)
		tokensTransferredInUnit, tokenSymbol, _ := TokenAmountToUnit(addr.TokensTransferred, addr.Address)
		db.MustExec("INSERT INTO analysis_address_stat (analysis_id, address, numtxsent, numtxreceived, NumTxWithData, NumTxTokenTransfer, NumTxTokenMethodTransfer, NumTxTokenMethodTransferFrom, valuesenteth, valuereceivedeth, tokenstransferred, tokenstransferredinunit, tokenstransferredsymbol) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)",
			analysisId, strings.ToLower(addr.Address), addr.NumTxSent, addr.NumTxReceived, addr.NumTxWithData, addr.NumTxTokenTransfer, addr.NumTxTokenMethodTransfer, addr.NumTxTokenMethodTransferFrom, valSentEth, valRecEth, addr.TokensTransferred.String(), tokensTransferredInUnit.Text('f', 8), tokenSymbol)
	}

	var wg sync.WaitGroup // for waiting until all blocks are written into DB
	addressInfoQueue := make(chan AddressInfo, 50)
	saveAddrStatToDbWorker := func() {
		defer wg.Done()
		for addr := range addressInfoQueue {
			addAddressAndStats(addr)
		}
	}

	for w := 1; w <= 5; w++ {
		wg.Add(1)
		go saveAddrStatToDbWorker()
	}

	fmt.Println("Adding address stats to DB...")
	fmt.Println("- Tokens transferred")
	numAddressesTokenTransfers := 100
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].NumTxTokenTransfer > _addresses[j].NumTxTokenTransfer })
	for i := 0; i < len(_addresses) && i < numAddressesTokenTransfers; i++ {
		if _addresses[i].NumTxTokenTransfer > 0 {
			addressInfoQueue <- _addresses[i]
			// addAddressAndStats(_addresses[i])
		}
	}

	fmt.Println("- Num tx received")
	numAddressesTxReceived := 25
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].NumTxReceived > _addresses[j].NumTxReceived })
	for i := 0; i < len(_addresses) && i < numAddressesTxReceived; i++ {
		if _addresses[i].NumTxReceived > 0 {
			addressInfoQueue <- _addresses[i]
			// addAddressAndStats(_addresses[i])
		}
	}

	fmt.Println("- Num tx sent")
	numAddressesTxSent := 25
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].NumTxSent > _addresses[j].NumTxSent })
	for i := 0; i < len(_addresses) && i < numAddressesTxSent; i++ {
		if _addresses[i].NumTxSent > 0 {
			addressInfoQueue <- _addresses[i]
			// addAddressAndStats(_addresses[i])
		}
	}

	fmt.Println("- Value received")
	numAddressesValueReceived := 25
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].ValueReceivedWei.Cmp(_addresses[j].ValueReceivedWei) == 1 })
	for i := 0; i < len(_addresses) && i < numAddressesValueReceived; i++ {
		if _addresses[i].ValueReceivedWei.Cmp(big.NewInt(0)) == 1 { // if valueReceived > 0
			addressInfoQueue <- _addresses[i]
			// addAddressAndStats(_addresses[i])
		}
	}

	fmt.Println("- Value sent")
	numAddressesValueSent := 25
	sort.SliceStable(_addresses, func(i, j int) bool { return _addresses[i].ValueSentWei.Cmp(_addresses[j].ValueSentWei) == 1 })
	for i := 0; i < len(_addresses) && i < numAddressesValueSent; i++ {
		if _addresses[i].ValueSentWei.Cmp(big.NewInt(0)) == 1 { // if valueSent > 0
			addressInfoQueue <- _addresses[i]
			// addAddressAndStats(_addresses[i])
		}
	}

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
