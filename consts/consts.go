package consts

// Types of stats
var (
	NumTxReceived        string = "NumTxReceived"
	NumTxReceivedSuccess        = "NumTxReceivedSuccess"
	NumTxReceivedFailed         = "NumTxReceivedFailed"

	NumTxSent        = "NumTxSent"
	NumTxSentSuccess = "NumTxSentSuccess"
	NumTxSentFailed  = "NumTxSentFailed"

	NumTxFlashbotsSent     = "NumTxFlashbotsSent"
	NumTxFlashbotsReceived = "NumTxFlashbotsReceived"

	NumTxWithDataSent     = "NumTxWithDataSent"
	NumTxWithDataReceived = "NumTxWithDataReceived"

	NumTxErc20Sent      = "NumTxErc20Sent"
	NumTxErc721Sent     = "NumTxErc721Sent"
	NumTxErc20Received  = "NumTxErc20Received"
	NumTxErc721Received = "NumTxErc721Received"
	NumTxErc20Transfer  = "NumTxErc20Transfer"
	NumTxErc721Transfer = "NumTxErc721Transfer"

	ValueSentWei     = "ValueSentWei"
	ValueReceivedWei = "ValueReceivedWei"

	Erc20TokensSent        = "Erc20TokensSent"
	Erc20TokensReceived    = "Erc20TokensReceived"
	Erc20TokensTransferred = "Erc20TokensTransferred"

	GasUsed        = "GasUsed"
	GasFeeTotal    = "GasFeeTotal"
	GasFeeFailedTx = "GasFeeFailedTx"
)

var StatsKeys = [...]string{
	NumTxReceived, NumTxReceivedSuccess, NumTxReceivedFailed,
	NumTxSent, NumTxSentSuccess, NumTxSentFailed,
	NumTxFlashbotsSent, NumTxFlashbotsReceived,
	NumTxWithDataSent, NumTxWithDataReceived,
	NumTxErc721Sent, NumTxErc20Received,
	NumTxErc721Received,
	NumTxErc20Transfer,
	NumTxErc721Transfer,
	ValueSentWei,
	ValueReceivedWei,
	Erc20TokensSent,
	Erc20TokensReceived,
	Erc20TokensTransferred,
	GasUsed,
	GasFeeTotal,
	GasFeeFailedTx,
}
