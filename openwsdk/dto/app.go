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
