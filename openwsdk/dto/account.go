package dto

import (
	"github.com/godaddy-x/freego/node/common"
	"github.com/godaddy-x/freego/ormx/sqlc"
)

//easyjson:json
type CreateAccountReq struct {
	common.BaseReq
	WalletID       string   `json:"walletID"`
	Alias          string   `json:"alias"`
	Symbol         string   `json:"symbol"`
	OtherOwnerKeys []string `json:"otherOwnerKeys"`
	ReqSigs        int64    `json:"reqSigs"`
	IsTrust        int64    `json:"isTrust"`
	PublicKey      string   `json:"publicKey"`
	Password       string   `json:"password"`
	AccountIndex   int64    `json:"accountIndex"`
	AccountID      string   `json:"accountID"`
	HdPath         string   `json:"hdPath"`
	Remark         string   `json:"remark"`
	OnlyAccount    int64    `json:"onlyAccount"`
	UserID         int64    `json:"userID"`
}

//easyjson:json
type CreateAccountRes struct {
	Account   AccountResult   `json:"account"`
	Addresses []AddressResult `json:"addresses"`
}

//easyjson:json
type FindAccountByAccountIDReq struct {
	common.BaseReq
	AccountID string `json:"accountID"`
	Symbol    string `json:"symbol"`
	UserID    int64  `json:"userID"`
}

//easyjson:json
type FindAccountByAccountIDRes struct {
	Account AccountResult `json:"account"`
}

//easyjson:json
type GetBalanceByAccountReq struct {
	common.BaseReq
	AccountID  string `json:"accountID"`
	Symbol     string `json:"symbol"`
	ContractID string `json:"contractID"`
	UserID     int64  `json:"userID"`
}

//easyjson:json
type GetBalanceByAccountRes struct {
	Balance BalanceResult `json:"balance"`
}

//easyjson:json
type GetAccountBalanceListReq struct {
	common.BaseReq
	WalletID   string `json:"walletID"`
	AccountID  string `json:"accountID"`
	Symbol     string `json:"symbol"`
	ContractID string `json:"contractID"`
	Type       int64  `json:"type"`
	UserID     int64  `json:"userID"`
	Sort       int    `json:"sort"`
}

//easyjson:json
type GetAccountBalanceListRes struct {
	Result []BalanceResult `json:"Result"`
	Limit  sqlc.Limit      `json:"limit"`
}

//easyjson:json
type FindAccountByWalletIDReq struct {
	common.BaseReq
	WalletID string `json:"walletID"`
	Symbol   string `json:"symbol"`
	UserID   int64  `json:"userID"`
	Sort     int    `json:"sort"`
}

//easyjson:json
type FindAccountByWalletIDRes struct {
	Result []AccountResult `json:"Result"`
	Limit  sqlc.Limit      `json:"limit"`
}

//easyjson:json
type AccountResult struct {
	ID             int64    `json:"id"`
	AppID          string   `json:"appID"`
	WalletID       string   `json:"walletID"`
	AccountID      string   `json:"accountID"`
	Alias          string   `json:"alias"`
	MainSymbol     string   `json:"mainSymbol"`
	OtherOwnerKeys []string `json:"otherOwnerKeys"`
	ReqSigs        int64    `json:"reqSigs"`
	IsTrust        int64    `json:"isTrust"`
	PublicKey      string   `json:"publicKey"`
	HdPath         string   `json:"hdPath"`
	AccountIndex   int64    `json:"accountIndex"`
	AddressIndex   int64    `json:"addressIndex"`
	Ctime          int64    `json:"ctime"`
	Remark         string   `json:"remark"`
}

//easyjson:json
type AddressResult struct {
	ID         int64  `json:"id"`
	AppID      string `json:"appID"`
	WalletID   string `json:"walletID"`
	AccountID  string `json:"accountID"`
	Alias      string `json:"alias"`
	MainSymbol string `json:"mainSymbol"`
	AddrIndex  int64  `json:"addrIndex"`
	Address    string `json:"address"`
	IsMemo     int64  `json:"isMemo"`
	Memo       string `json:"memo"`
	WatchOnly  int64  `json:"watchOnly"`
	PublicKey  string `json:"publicKey"`
	HdPath     string `json:"hdPath"`
	Ctime      int64  `json:"ctime"`
}

//easyjson:json
type BalanceResult struct {
	ID               int64  `json:"id"`
	AppID            string `json:"appID"`
	WalletID         string `json:"walletID"`
	AccountID        string `json:"accountID"`
	Address          string `json:"address"`
	MainSymbol       string `json:"mainSymbol"`
	Symbol           string `json:"symbol"`
	ContractID       string `json:"contractID"`
	ContractAddr     string `json:"contractAddr"`
	Balance          string `json:"balance"`
	ConfirmBalance   string `json:"confirmBalance"`
	UnconfirmBalance string `json:"unconfirmBalance"`
	Utime            int64  `json:"utime"`
	ContractToken    string `json:"contractToken"`
}
