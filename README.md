
```bash
# Run analyzer
go run cmd/analyzer/main.go -date 2021-05-01 -len 10m | tee log
go run cmd/analyzer/main.go -date 2021-05-03 -len 1d -out data/out/2021-05-03.json | tee data/out/2021-05-03.txt

# Run addresstool

# Download data from server
scp eth:/server/code/ethereum-go-experiments/data/out/2021-05-03.* .
```


---

Resources:

* https://bitinfocharts.com/ethereum/ - inspiration
* https://goethereumbook.org/en/transaction-query/

To do:

* Finding blocks: don't fail if >latest-height
* Gas fees

---

Goals:

* Over given timespan, check all blocks
* Count
  * Transactions
  * Sender
  * Receiver


TX can be ETH transfer or SC invocation

TX types:

* Eth transfer
* SC invocation
  * Generic
  * ERC-20
  * ERC-721

---

## ERC 20 + ERC 721

### transfer

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
*

### transferFrom

```
Function: transferFrom(address from, address to, uint256 tokenId)

MethodID: 0x23b872dd
[0]:  0000000000000000000000006a95158c6cfb05fc1fd1040f46e24e9e66931d76
[1]:  000000000000000000000000ad66810a7358ad499c611d50108d0e0a8733e913
[2]:  0000000000000000000000000000000000000000000000000000000000055516
```

ERC 721:

* https://etherscan.io/tx/0xc4bbf3633cd5fc760ebe3eab4b2d54cad4220125997bb696cf17f0e5e48466a9 (rarible)
* https://etherscan.io/tx/0x18bd3184549fc2456f2968d3565f620413336ab556613440eea803b153f62fc3 (rarible)
* https://etherscan.io/tx/0x2909d76a36a5daf8c9b998841b516461b27961dd4edbc009338f9e87fef2b44d (Hashmasks)


---

Reference block: 1619546404

* https://etherscan.io/block/12323940
* Timestamp (sec): 1619546404
* UTC: Tue Apr 27 2021 18:00:04 GMT+0000
* VIE: 2021-04-27 20:00:04 +0200 CEST
