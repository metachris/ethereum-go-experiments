Iterate over blocks and all contained transactions on the Ethereum blockchain, and build statistics based on the data.

Requires an Ethereum Node to connect to (eg. geth, Infura).

## Geting started

```bash
# Run analyzer for certain timespan
go run cmd/analyzer/main.go -date 2021-05-01 -len 5m
go run cmd/analyzer/main.go -date 2021-05-01 -len 5m -addDb
go run cmd/analyzer/main.go -date 2021-05-09 -len 1d -addDb -out output/2021-05-09.json | tee output/2021-05-09.txt

# Run analyzer for specific block
go run cmd/analyzer/main.go -block 12381372

# Run analyzer starting at specific block, for certain number of blocks (in this case, 10 blocks total)
go run cmd/analyzer/main.go -block 12381372 -len 10

# Run addresstool
go run cmd/addresstool/main.go -add -addr 0x69af81e73A73B40adF4f3d4223Cd9b1ECE623074

# Download data from server
scp eth:/server/code/ethereum-go-experiments/data/out/2021-05-03.* .
```

## Working with DB

```bash
docker-compose up
```

Access the adminer DB interface: http://localhost:8080/?pgsql=db&username=user1&db=ethstats&ns=public

---

## To do

* Save results for 1h in addition to 1d
* search for TODO
* Gas fees
* Add blocks in own goroutine (not in main routine and not one per block)

## Resources

* https://goethereumbook.org/en/transaction-query/
* https://github.com/jmoiron/sqlx

Inspiration & Ideas

* https://bitinfocharts.com/ethereum/
* https://etherscan.io/charts
* https://ycharts.com/indicators/ethereum_transactions_per_day
* https://github.com/ardanlabs/service/blob/master/foundation/database/database.go

---

# Notes

## Token transfer SC calls

### `transfer`

```
Function: transfer(address _to, uint256 _value)

MethodID: 0xa9059cbb

[0]:  000000000000000000000000a907f05e51d1f6f2a9fb0da6cc6da9a9590a8d2b
[1]:  00000000000000000000000000000000000000000000005150ae84a8cdf00000
```

ERC 20:

* https://etherscan.io/tx/0xb36d6fdbda67ee6eb9316b4c269e103f1fcdd0d145599f90f9acdbd2a0665d95
* https://etherscan.io/tx/0x88842ff1bd917bbf1cea042f06f5b71a742792d16ac5fa54aba7953c3e3e0764
* https://etherscan.io/tx/0x4ae525f03921e4368bb8b7973fcddb65db6ee9a1ebd561f9be99a09170dbf96f

ERC 721:

* https://etherscan.io/tx/0x08cc6cf2b6ad598458c1fc71026f4c55d9ffa1c91baf1354ee74204406205b8c (CryptoKitties)


### `transferFrom`

```
Function: transferFrom(address from, address to, uint256 tokenId)

MethodID: 0x23b872dd
[0]:  0000000000000000000000006a95158c6cfb05fc1fd1040f46e24e9e66931d76
[1]:  000000000000000000000000ad66810a7358ad499c611d50108d0e0a8733e913
[2]:  0000000000000000000000000000000000000000000000000000000000055516
```

ERC 20:

* https://etherscan.io/tx/0x0d86cfde5a6e2a1d86f8e9d0d1c30ac8edcdc415235689cdf81260cdd45ad25c (Uniswap)

ERC 721:

* https://etherscan.io/tx/0xc4bbf3633cd5fc760ebe3eab4b2d54cad4220125997bb696cf17f0e5e48466a9 (rarible)
* https://etherscan.io/tx/0x18bd3184549fc2456f2968d3565f620413336ab556613440eea803b153f62fc3 (rarible)
* https://etherscan.io/tx/0x2909d76a36a5daf8c9b998841b516461b27961dd4edbc009338f9e87fef2b44d (Hashmasks)

