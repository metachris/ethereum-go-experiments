package ethstats

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/metachris/ethereum-go-experiments/consts"
	"github.com/metachris/ethereum-go-experiments/core"
	"github.com/metachris/go-ethutils/utils"
)

func ProcessBlockWithReceipts(block *BlockWithTxReceipts, client *ethclient.Client, analysis *core.Analysis) {
	if analysis.Data.StartBlockTimestamp == 0 {
		analysis.Data.StartBlockTimestamp = block.block.Time()
	}

	analysis.Data.EndBlockNumber = block.block.Number().Int64()
	analysis.Data.EndBlockTimestamp = block.block.Time()

	analysis.Data.NumBlocks += 1
	analysis.Data.NumTransactions += len(block.block.Transactions())
	analysis.Data.GasUsed = new(big.Int).Add(analysis.Data.GasUsed, big.NewInt(int64(block.block.GasUsed())))

	// Iterate over all transactions
	for _, tx := range block.block.Transactions() {
		receipt := block.txReceipts[tx.Hash()]
		ProcessTransaction(client, tx, receipt, analysis)
	}

	// If no transactions in this block then record that
	if len(block.block.Transactions()) == 0 {
		analysis.Data.NumBlocksWithoutTx += 1
	}
}

func ProcessTransaction(client *ethclient.Client, tx *types.Transaction, receipt *types.Receipt, analysis *core.Analysis) {
	analysis.AddTxToTopList(tx, receipt)

	txToAddrStats := analysis.GetOrCreateAddressStats(tx.To())
	txFromAddrStats := analysis.GetOrCreateAddressStats(nil)
	if from, err := utils.GetTxSender(tx); err == nil {
		txFromAddrStats = analysis.GetOrCreateAddressStats(&from)
	}

	txToAddrStats.Add1(consts.NumTxReceived)
	txFromAddrStats.Add1(consts.NumTxSent)

	// Gas used is paid and counted no matter if transaction failed or succeeded
	txSuccess := true        // default, used if no receipt
	txGasUsed := common.Big1 // default, used if no receipt
	if receipt != nil {
		txSuccess = receipt.Status == 1
		txGasUsed = big.NewInt(int64(receipt.GasUsed))
	}

	txGasFee := new(big.Int).Mul(txGasUsed, tx.GasPrice())
	analysis.Data.GasFeeTotal = new(big.Int).Add(analysis.Data.GasFeeTotal, txGasFee)
	txFromAddrStats.Add(consts.GasUsed, txGasUsed)
	txFromAddrStats.Add(consts.GasFeeTotal, txGasFee)

	if !txSuccess {
		analysis.Data.NumTransactionsFailed += 1
		analysis.Data.GasFeeFailedTx = new(big.Int).Add(analysis.Data.GasFeeFailedTx, txGasFee)

		txFromAddrStats.Add1(consts.NumTxSentFailed)
		txFromAddrStats.Add(consts.GasFeeFailedTx, txGasFee)

		txToAddrStats.Add1(consts.NumTxReceivedFailed)

		// Count failed flashbots tx
		if len(tx.Data()) > 0 && tx.GasPrice().Uint64() == 0 {
			analysis.Data.NumFlashbotsTransactionsFailed += 1
			// fmt.Printf("0-gas/Flashbots fail tx: https://etherscan.io/tx/%s\n", tx.Hash())
			analysis.TagTransactionStats(core.NewTxStatsFromTransactions(tx, receipt), consts.TxFlashBotsFailed, client)
			txFromAddrStats.Add1(consts.FlashBotsFailedTxSent)
		}

		return
	}

	// TX was successful...
	analysis.Data.ValueTotalWei = analysis.Data.ValueTotalWei.Add(analysis.Data.ValueTotalWei, tx.Value())
	analysis.Data.TxTypes[tx.Type()] += 1

	txFromAddrStats.Add1(consts.NumTxSentSuccess)
	txFromAddrStats.Add(consts.ValueSentWei, tx.Value())

	txToAddrStats.Add1(consts.NumTxReceivedSuccess)
	txToAddrStats.Add(consts.ValueReceivedWei, tx.Value())

	if utils.IsBigIntZero(tx.Value()) {
		analysis.Data.NumTransactionsWithZeroValue += 1
	}

	// Check for smart contract calls: token transfer
	data := tx.Data()
	if len(data) > 0 {
		analysis.Data.NumTransactionsWithData += 1

		txFromAddrStats.Add1(consts.NumTxWithDataSent)
		txToAddrStats.Add1(consts.NumTxWithDataReceived)

		// If no gas price, then count as MEV/Flashbots tx
		if tx.GasPrice().Uint64() == 0 {
			analysis.Data.NumFlashbotsTransactionsSuccess += 1
			txFromAddrStats.Add1(consts.NumTxFlashbotsSent)
			txToAddrStats.Add1(consts.NumTxFlashbotsReceived)

			if core.Cfg.DebugPrintFlashbotsTx {
				fmt.Printf("Flashbots ok tx: https://etherscan.io/tx/%s\n", tx.Hash())
			}
		}

		if len(data) > 4 {
			methodId := hex.EncodeToString(data[:4])
			// fmt.Println(methodId)
			if methodId == "a9059cbb" || methodId == "23b872dd" {
				analysis.EnsureAddressDetailIsLoaded(&txFromAddrStats.AddressDetail)
				analysis.EnsureAddressDetailIsLoaded(&txToAddrStats.AddressDetail)

				// Calculate and store the number of tokens transferred
				valueSenderStats := txFromAddrStats
				valueReceiverStats := txToAddrStats
				var value string
				switch methodId {
				case "a9059cbb": // transfer
					_receiverAddrString := hex.EncodeToString(data[4:36])
					_receiverAddr := common.HexToAddress(_receiverAddrString)
					// fmt.Println("transfer. to", _receiverAddr, "\t .. tx from:", txFromAddrStats.Address)
					valueReceiverStats = analysis.GetOrCreateAddressStats(&_receiverAddr)
					analysis.EnsureAddressDetailIsLoaded(&valueReceiverStats.AddressDetail)

					value = hex.EncodeToString(data[36:68])

				case "23b872dd": // transferFrom
					_senderAddrString := hex.EncodeToString(data[4:36])
					_senderAddr := common.HexToAddress(_senderAddrString)
					valueSenderStats = analysis.GetOrCreateAddressStats(&_senderAddr)
					analysis.EnsureAddressDetailIsLoaded(&valueSenderStats.AddressDetail)

					_receiverAddrString := hex.EncodeToString(data[36:68])
					_receiverAddr := common.HexToAddress(_receiverAddrString)
					valueReceiverStats = analysis.GetOrCreateAddressStats(&_receiverAddr)
					analysis.EnsureAddressDetailIsLoaded(&valueReceiverStats.AddressDetail)

					value = hex.EncodeToString(data[68:100])
					// fmt.Println("transferFrom. from", _senderAddr, ", tx", txFromAddrStats.Address, "... to", _receiverAddr)
				}

				valBigInt := new(big.Int)
				valBigInt.SetString(value, 16)

				// If ERC2 SC call
				if txToAddrStats.AddressDetail.IsErc20() {
					analysis.Data.NumTransactionsErc20Transfer += 1

					// Count sender
					txFromAddrStats.Add1(consts.NumTxErc20Sent)
					txFromAddrStats.Add(consts.Erc20TokensSent, valBigInt)
					if txFromAddrStats.AddressDetail.Address != valueSenderStats.AddressDetail.Address {
						valueSenderStats.Add1(consts.NumTxErc20Sent)
						valueSenderStats.Add(consts.Erc20TokensSent, valBigInt)
					}

					// Count SC transfer calls
					txToAddrStats.Add1(consts.NumTxErc20Transfer)
					txToAddrStats.Add(consts.Erc20TokensTransferred, valBigInt)

					// Count token transfer receiver
					valueReceiverStats.Add1(consts.NumTxErc20Received)
					valueReceiverStats.Add(consts.Erc20TokensReceived, valBigInt)

				} else if txToAddrStats.AddressDetail.IsErc721() {
					// fmt.Println("ERC721 transfer", tx.Hash(), "from", txFromAddrStats.AddressDetail.Address)
					analysis.Data.NumTransactionsErc721Transfer += 1

					// Count sender
					txFromAddrStats.Add1(consts.NumTxErc721Sent)
					// fmt.Println(1, txFromAddrStats.AddressDetail.Address, txFromAddrStats.Stats[consts.NumTxErc721Sent])
					if txFromAddrStats.AddressDetail.Address != valueSenderStats.AddressDetail.Address {
						valueSenderStats.Add1(consts.NumTxErc721Sent)
					}

					// Count SC
					txToAddrStats.Add1(consts.NumTxErc721Transfer)

					// Count token receiver
					valueReceiverStats.Add1(consts.NumTxErc721Received)
				}
			}
		}
	}
}
