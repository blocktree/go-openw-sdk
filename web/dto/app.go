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
	KeyID    string `json:"keyID"`
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
	WalletList map[string]string `json:"walletList"`
}

//easyjson:json
type UnlockWalletRes struct {
	Failed []string `json:"failed"`
}
