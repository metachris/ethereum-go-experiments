Iterate over blocks and all contained transactions on the Ethereum blockchain, and build statistics based on the data.

Requires an Ethereum Node to connect to (eg. geth, Infura).

## Geting started

```bash
# Start DB
docker-compose up

# Run analyzer for certain timespan
go run cmd/analyzer/main.go -date 2021-05-01 -len 5m
go run cmd/analyzer/main.go -date 2021-05-01 -len 5m -addDb
go run cmd/analyzer/main.go -date 2021-05-09 -len 1d -addDb -out output/2021-05-09.json | tee output/2021-05-09.txt

# Run analyzer for specific block
go run cmd/analyzer/main.go -block 12381372
go run cmd/analyzer/main.go -block 12381372 -len 10

# Run addresstool - check ethplorer
go run cmd/addresstool/main.go -add -addr 0x69af81e73A73B40adF4f3d4223Cd9b1ECE623074

# Download data from server
scp eth:/server/code/ethereum-go-experiments/data/out/2021-05-03.* .

# Webserver
go run cmd/webserver/main.go
curl localhost:8090/analysis/1
```

Notes:

* Access the adminer DB interface: http://localhost:8080/?pgsql=db&username=user1&db=ethstats&ns=public

---

## To do

* Handle reverted transactions: https://etherscan.io/tx/0x50d55cad863ea3028474076882475f21c29312131a660d9628bad81c098fc30b
* search for TODO

Issues

* tx 0xbcb696bf17f6d47748cd58f667d269c3dd816377db9fe17653be67f8a2d6f377 	 val: 362d66c830df8fb6a0335288ccdf4933a4d2dc2ace19f41285b9081c6cd478f2 	 valBigInt: 24505111316675076152287381684569405521618230748966674567453261522074013563122 / 24505111316675076152287381684569405521618230748966674567453261522074013563122 	 total: 24505111316675076152287381684569405521618230748966674567453261522074013563122 
* tx 0x50d55cad863ea3028474076882475f21c29312131a660d9628bad81c098fc30b 	 val: fffffffffffffffffffffffffffffffffffffffffffffffffffffffff8c2ba4d 	 valBigInt: 115792089237316195423570985008687907853269984665640564039457584007913008183885 / 115792089237316195423570985008687907853269984665640564039457584007913008183885 	 total: 115792089237316195423570985008687907853269984665640564039457584047225353006840 

Maybe?

* Save results for 1h in addition to 1d
* Gas fees
* Add blocks in own goroutine (not in main routine and not one per block)

## Resources

* https://echo.labstack.com/guide/request/
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

