package dto

import "github.com/godaddy-x/freego/node/common"

//easyjson:json
type AppLoginReq struct {
	common.BaseReq
	AppID string `json:"appID"`
	Sign  string `json:"sign"`
	Nonce string `json:"nonce"`
	Time  int64  `json:"time"`
}

//easyjson:json
type AppLoginRes struct {
	Subject string `json:"subject"`
}

//easyjson:json
type WalletResult struct {
	Alias    string `json:"alias"`
	WalletID string `json:"walletID"`
	RootPath string `json:"rootPath"`
	Version  int    `json:"version"`
}

//easyjson:json
type FindWalletListReq struct {
	common.BaseReq
}

//easyjson:json
type FindWalletListRes struct {
	Result []WalletResult `json:"result"`
}

//easyjson:json
type UnlockWalletReq struct {
	common.BaseReq
	Filename string `json:"filename"`
}

//easyjson:json
type UnlockWalletRes struct {
	WalletID string `json:"walletID"`
}

//easyjson:json
type CreateWalletReq struct {
	common.BaseReq
	Alias string `json:"alias"`
}

//easyjson:json
type CreateWalletRes struct {
	WalletID string `json:"walletID"`
}

//easyjson:json
type CreateAccountReq struct {
	common.BaseReq
	WalletID string `json:"walletID"`
}

//easyjson:json
type CreateAccountRes struct {
	AccountID string `json:"accountID"`
}
