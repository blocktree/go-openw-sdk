package dto

import (
	"github.com/godaddy-x/freego/node/common"
	"github.com/godaddy-x/freego/ormx/sqlc"
)

//easyjson:json
type SmartContract struct {
	ContractID string `json:"contractID"`
	Symbol     string `json:"symbol"`
	Address    string `json:"address"`
	Token      string `json:"token"`
	Protocol   string `json:"protocol"`
	Name       string `json:"name"`
	Decimals   uint64 `json:"decimals"`
}

//easyjson:json
type Address struct {
	Alias     string `json:"alias"`
	Symbol    string `json:"symbol"`
	AddrIndex int64  `json:"addrIndex"`
	Address   string `json:"address"`
	IsMemo    int64  `json:"isMemo"`
	Memo      string `json:"memo"`
	WatchOnly int64  `json:"watchOnly"`
	PublicKey string `json:"publicKey"`
	HdPath    string `json:"hdPath"`
}

//easyjson:json
type TokenBalanceResult struct {
	Balance  BalanceResult `json:"balance"`
	Contract SmartContract `json:"contract"`
}

//easyjson:json
type UnspentResult struct {
	ID            string `json:"id"`
	Txid          string `json:"txid"`
	Vout          uint64 `json:"vout"`
	Address       string `json:"address"`
	Account       string `json:"account"`
	ScriptPubKey  string `json:"scriptPubKey"`
	Amount        string `json:"amount"`
	Confirmations uint64 `json:"confirmations"`
	Spendable     bool   `json:"spendable"`
	Solvable      bool   `json:"solvable"`
}

//easyjson:json
type CreateAddressReq struct {
	common.BaseReq
	WalletID  string `json:"walletID"`
	AccountID string `json:"accountID"`
	Symbol    string `json:"symbol"`
	Count     int64  `json:"count"`
	UserID    int64  `json:"userID"`
}

//easyjson:json
type CreateAddressRes struct {
	Addresses []AddressResult `json:"addresses"`
}

//easyjson:json
type ImportAddressReq struct {
	common.BaseReq
	WalletID  string    `json:"walletID"`
	AccountID string    `json:"accountID"`
	Symbol    string    `json:"symbol"`
	Address   []string  `json:"address"`
	UserID    int64     `json:"userID"`
	Addresses []Address `json:"addresses"`
}

//easyjson:json
type ImportAddressRes struct {
	Addresses   []AddressResult `json:"addresses"`
	CreateCount int64           `json:"createCount"`
}

//easyjson:json
type FindAddressByAddressReq struct {
	common.BaseReq
	Address string `json:"address"`
	Symbol  string `json:"symbol"`
	UserID  int64  `json:"userID"`
}

//easyjson:json
type FindAddressByAddressRes struct {
	Address AddressResult `json:"address"`
}

//easyjson:json
type FindAddressByAccountIDReq struct {
	common.BaseReq
	AccountID string `json:"accountID"`
	UserID    int64  `json:"userID"`
	Sort      int    `json:"sort"`
}

//easyjson:json
type FindAddressByAccountIDRes struct {
	Result []AddressResult `json:"result"`
	Limit  sqlc.Limit      `json:"limit"`
}

//easyjson:json
type VerifyAddressReq struct {
	common.BaseReq
	Symbol  string `json:"symbol"`
	Address string `json:"address"`
	UserID  int64  `json:"userID"`
}

//easyjson:json
type VerifyAddressRes struct {
	Result bool `json:"result"`
}

//easyjson:json
type GetBalanceByAddressReq struct {
	common.BaseReq
	Address    string `json:"address"`
	Symbol     string `json:"symbol"`
	ContractID string `json:"contractID"`
	UserID     int64  `json:"userID"`
}

//easyjson:json
type GetBalanceByAddressRes struct {
	Balance BalanceResult `json:"balance"`
}

//easyjson:json
type GetAddressBalanceListReq struct {
	common.BaseReq
	WalletID   string `json:"walletID"`
	AccountID  string `json:"accountID"`
	Address    string `json:"address"`
	Symbol     string `json:"symbol"`
	ContractID string `json:"contractID"`
	Type       int64  `json:"type"`
	UserID     int64  `json:"userID"`
	Sort       int    `json:"sort"`
}

//easyjson:json
type GetAddressBalanceListRes struct {
	Result []BalanceResult `json:"Result"`
	Limit  sqlc.Limit      `json:"limit"`
}

//easyjson:json
type GetBalanceByAddressOnChainReq struct {
	common.BaseReq
	Symbol    string   `json:"symbol"`
	AccountID string   `json:"accountID"`
	Addresses []string `json:"addresses"`
}

//easyjson:json
type GetBalanceByAddressOnChainRes struct {
	Balances    []BalanceResult `json:"balances"`
	BalanceType int64           `json:"balanceType"`
}

//easyjson:json
type GetTokenBalanceByAddressOnChainReq struct {
	common.BaseReq
	Symbol          string   `json:"symbol"`
	AccountID       string   `json:"accountID"`
	Addresses       []string `json:"addresses"`
	ContractID      string   `json:"contractID"`
	ContractAddress string   `json:"contractAddress"`
	ContractDecimal int64    `json:"contractDecimal"`
	ContractToken   string   `json:"contractToken"`
}

//easyjson:json
type GetTokenBalanceByAddressOnChainRes struct {
	Balances    []TokenBalanceResult `json:"balances"`
	BalanceType int64                `json:"balanceType"`
}

//easyjson:json
type GetUnspentOnChainReq struct {
	common.BaseReq
	Symbol  string   `json:"symbol"`
	Address []string `json:"address"`
	Min     uint64   `json:"min"`
}

//easyjson:json
type GetUnspentOnChainRes struct {
	Result []UnspentResult `json:"result"`
}
