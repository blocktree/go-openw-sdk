package dto

import (
	"github.com/godaddy-x/freego/node/common"
	"github.com/godaddy-x/freego/ormx/sqlc"
)

//easyjson:json
type CreateWalletReq struct {
	common.BaseReq
	Alias    string `json:"alias"`
	WalletID string `json:"walletID"`
	RootPath string `json:"rootPath"`
	UserID   int64  `json:"userID"`
}

//easyjson:json
type CreateWalletRes struct {
	Result       bool   `json:"result"`
	WalletID     string `json:"walletID"`
	RootPath     string `json:"rootPath"`
	Alias        string `json:"alias"`
	AccountIndex int64  `json:"accountIndex"`
}

//easyjson:json
type FindWalletByWalletIDReq struct {
	common.BaseReq
	WalletID string `json:"walletID"`
	UserID   int64  `json:"userID"`
}

//easyjson:json
type FindWalletByWalletIDRes struct {
	Result WalletResult `json:"result"`
}

//easyjson:json
type FindWalletByParamsReq struct {
	common.BaseReq
	WalletID string `json:"walletID"`
	UserID   int64  `json:"userID"`
	Sort     int    `json:"sort"`
}

//easyjson:json
type FindWalletByParamsRes struct {
	Result []WalletResult `json:"result"`
	Limit  sqlc.Limit     `json:"limit"`
}

//easyjson:json
type WalletResult struct {
	ID           int64  `json:"id"`
	AppID        string `json:"appID"`
	WalletID     string `json:"walletID"`
	RootPath     string `json:"rootPath"`
	Alias        string `json:"alias"`
	AccountIndex int64  `json:"accountIndex"`
	Ctime        int64  `json:"ctime"`
}
