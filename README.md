Iterate over blocks and all contained transactions on the Ethereum blockchain, and build statistics based on the data.

Requires an Ethereum Node to connect to (eg. geth, Infura).

## Geting started

```bash
# Start DB
docker-compose up

# Load the environment variables
source .env.example

# Run analyzer for certain timespan
go run cmd/analyzer/main.go -date 2021-05-01 -len 5m
go run cmd/analyzer/main.go -date 2021-05-01 -len 5m -addDb
go run cmd/analyzer/main.go -date 2021-05-17 -len 1d -addDb | tee tmp.txt
go run cmd/analyzer/main.go -date 2021-05-17 -len 1d -addDb -out output/2021-05-17.json | tee output/2021-05-17.txt

# Run analyzer for specific block
go run cmd/analyzer/main.go -block 12381372
go run cmd/analyzer/main.go -block 12381372 -len 10

# Run addresstool to get info about an address
go run cmd/addresstool/main.go -addr 0x69af81e73A73B40adF4f3d4223Cd9b1ECE623074
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

* Gas paid per address. In particular interesting for sent failed transactions.
* Save results for 1h granularity, add up for daily stats?
* Speeding up things?
  * Maybe paralellize processing single blocks, and merging the individual results.

Maybe?

* Search for "todo"
* Gas fees per transaction?
* Add blocks in own goroutine (not in main routine and not one per block)
* Tests
  * contract type detection

---

# Documentation

## Token Analytics

Smart Contract Types:

* ERC20 (token, supported) https://eips.ethereum.org/EIPS/eip-20
* ERC721 (NFT, supported) https://eips.ethereum.org/EIPS/eip-721
* ERC777 (improved token, counts as ERC20) https://eips.ethereum.org/EIPS/eip-777
* ERC1155 (multi-token, not yet supported) https://eips.ethereum.org/EIPS/eip-1155

---

### Problematic Transactions

* tx 0xbcb696bf17f6d47748cd58f667d269c3dd816377db9fe17653be67f8a2d6f377 	 val: 362d66c830df8fb6a0335288ccdf4933a4d2dc2ace19f41285b9081c6cd478f2 	 valBigInt: 24505111316675076152287381684569405521618230748966674567453261522074013563122 / 24505111316675076152287381684569405521618230748966674567453261522074013563122 	 total: 24505111316675076152287381684569405521618230748966674567453261522074013563122
* tx 0x50d55cad863ea3028474076882475f21c29312131a660d9628bad81c098fc30b 	 val: fffffffffffffffffffffffffffffffffffffffffffffffffffffffff8c2ba4d 	 valBigInt: 115792089237316195423570985008687907853269984665640564039457584007913008183885 / 115792089237316195423570985008687907853269984665640564039457584007913008183885 	 total: 115792089237316195423570985008687907853269984665640564039457584047225353006840

## Resources

* https://echo.labstack.com/guide/request/
* https://goethereumbook.org/en/transaction-query/
* https://github.com/jmoiron/sqlx

Inspiration & Ideas

* https://bitinfocharts.com/ethereum/
* https://etherscan.io/charts
* https://ycharts.com/indicators/ethereum_transactions_per_day
* https://github.com/ardanlabs/service/blob/master/foundation/database/database.go

Contracts:

* ERC20: https://github.com/fxfactorial/defi-abigen
* ERC721: https://github.com/0xcert/ethereum-erc721/blob/master/src/contracts/tokens/erc721.sol

---

# Notes

## Token transfer smart-contract calls

There are 2 relevant methods:

* `transfer(address, uint256)` - MethodID: `0xa9059cbb`
  * ERC-20: `transfer(address _to, uint256 _value)`
  * ERC-721: `transfer(address _to, uint256 _tokenId)`

* `transferFrom(address, address, uint256)` - MethodID: `0x23b872dd`
  * ERC-20: `transferFrom(address src, address dst, uint256 rawAmount)`
  * ERC-721: `transferFrom(address from, address to, uint256 tokenId)`

Analyzing a transaction, it's unknown if the data is for an ERC-20 or ERC-721 contract.

What we'd like to know:

* ERC-20: Sum of tokens transferred
* ERC-721: Number of tokens transferred

Problems:

* erc-721 `_tokenId` method can be quite a large number, and it's unnecessary to count the total value.

---

### `transfer` and `transferFrom` details & examples

Function: `transfer(address, uint256)` - MethodID: `0xa9059cbb`

```
[0]:  000000000000000000000000a907f05e51d1f6f2a9fb0da6cc6da9a9590a8d2b
[1]:  00000000000000000000000000000000000000000000005150ae84a8cdf00000
```

ERC 20: `transfer(address _to, uint256 _value)`

* https://etherscan.io/tx/0xb36d6fdbda67ee6eb9316b4c269e103f1fcdd0d145599f90f9acdbd2a0665d95
* https://etherscan.io/tx/0x88842ff1bd917bbf1cea042f06f5b71a742792d16ac5fa54aba7953c3e3e0764
* https://etherscan.io/tx/0x4ae525f03921e4368bb8b7973fcddb65db6ee9a1ebd561f9be99a09170dbf96f

ERC 721: `transfer(address _to, uint256 _tokenId)`

* https://etherscan.io/tx/0x08cc6cf2b6ad598458c1fc71026f4c55d9ffa1c91baf1354ee74204406205b8c (CryptoKitties)


### `transferFrom`

Function: `transferFrom(address, address, uint256)` - MethodID: `0x23b872dd`

```
[0]:  0000000000000000000000006a95158c6cfb05fc1fd1040f46e24e9e66931d76
[1]:  000000000000000000000000ad66810a7358ad499c611d50108d0e0a8733e913
[2]:  0000000000000000000000000000000000000000000000000000000000055516
```

ERC 20:

Function: `transferFrom(address src, address dst, uint256 rawAmount)`

* https://etherscan.io/tx/0x0d86cfde5a6e2a1d86f8e9d0d1c30ac8edcdc415235689cdf81260cdd45ad25c (Uniswap)

ERC 721:

Function: `transferFrom(address from, address to, uint256 tokenId)`

* https://etherscan.io/tx/0xc4bbf3633cd5fc760ebe3eab4b2d54cad4220125997bb696cf17f0e5e48466a9 (rarible)
* https://etherscan.io/tx/0x18bd3184549fc2456f2968d3565f620413336ab556613440eea803b153f62fc3 (rarible)
* https://etherscan.io/tx/0x2909d76a36a5daf8c9b998841b516461b27961dd4edbc009338f9e87fef2b44d (Hashmasks)


---

## Log

#### 2021-05-18

* Do I really need to check status of every transaction? Analyze performance slows to 10%. Only for SC calls: 20%
