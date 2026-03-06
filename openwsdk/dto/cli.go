package dto

import "github.com/godaddy-x/freego/node/common"

//easyjson:json
type CliWalletResult struct {
	Alias    string `json:"alias"`
	WalletID string `json:"walletID"`
	RootPath string `json:"rootPath"`
	Version  int    `json:"version"`
}

//easyjson:json
type CliFindWalletListReq struct {
	common.BaseReq
}

//easyjson:json
type CliFindWalletListRes struct {
	Result []WalletResult `json:"result"`
}

//easyjson:json
type CliUnlockWalletReq struct {
	common.BaseReq
	Filename string `json:"filename"`
}

//easyjson:json
type CliUnlockWalletRes struct {
	WalletID string `json:"walletID"`
}

//easyjson:json
type CliCreateWalletReq struct {
	common.BaseReq
	Alias string `json:"alias"`
}

//easyjson:json
type CliCreateWalletRes struct {
	WalletID string `json:"walletID"`
}

//easyjson:json
type CliCreateAccountReq struct {
	common.BaseReq
	WalletID  string `json:"walletID"`
	LastIndex int64  `json:"lastIndex"` // 錢包所屬帳戶ID最後索引值
	Curve     int64  `json:"curve"`
}

//easyjson:json
type CliCreateAccountRes struct {
	WalletID       string   `json:"walletID"`
	AccountID      string   `json:"accountID"`
	OtherOwnerKeys []string `json:"otherOwnerKeys"`
	ReqSigs        int64    `json:"reqSigs"`
	PublicKey      string   `json:"publicKey"`
	HdPath         string   `json:"hdPath"`
	AccountIndex   int64    `json:"accountIndex"`
	AddressIndex   int64    `json:"addressIndex"`
}

//easyjson:json
type CliSignTransactionReq struct {
	common.BaseReq
	Type      int64  `json:"type"` // 0.普通交易 1.汇总交易
	Data      string `json:"data"`
	TradeSign string `json:"tradeSign"` // CLI系统进行校验签名
}

//easyjson:json
type CliSignTransactionRes struct {
	SignerList map[string]string `json:"signerList"`
}

type CliSignTradeKeyReq struct {
	common.BaseReq
	Type int64  `json:"type"` // 0.普通交易 1.汇总交易
	Data string `json:"data"`
}

//easyjson:json
type CliSignTradeKeyRes struct {
	Sign string `json:"sign"`
}
