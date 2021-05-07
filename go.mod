module eth

go 1.16

require ethtools v1.0.0

replace ethtools => ./pkg/ethtools

require (
	github.com/ethereum/go-ethereum v1.10.2
	github.com/jmoiron/sqlx v1.3.3 // indirect
	github.com/lib/pq v1.10.1 // indirect
)
