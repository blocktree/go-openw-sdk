package dto

import (
	"github.com/godaddy-x/freego/node/common"
	"github.com/godaddy-x/freego/ormx/sqlc"
)

//easyjson:json
type TradeLogResult struct {
	ID           int64    `json:"id"`
	AppID        string   `json:"appID"`
	WalletID     string   `json:"walletID"`
	AccountID    string   `json:"accountID"`
	Sid          string   `json:"sid"`
	TxID         string   `json:"txID"`
	TxType       int64    `json:"txType"`
	TxAction     string   `json:"txAction"`
	WxID         string   `json:"wxID"`
	FromAddress  []string `json:"fromAddress"`
	FromAddressV []string `json:"fromAddressV"`
	ToAddress    []string `json:"toAddress"`
	ToAddressV   []string `json:"toAddressV"`
	Amount       string   `json:"amount"`
	Fees         string   `json:"fees"`
	Type         int64    `json:"type"`
	Symbol       string   `json:"symbol"`
	ContractID   string   `json:"contractID"`
	IsContract   int64    `json:"isContract"`
	Confirm      int64    `json:"confirm"`
	BlockHash    string   `json:"blockHash"`
	BlockHeight  int64    `json:"blockHeight"`
	IsMemo       int64    `json:"isMemo"`
	IsMain       int64    `json:"isMain"`
	Memo         string   `json:"memo"`
	Applytime    int64    `json:"applytime"`
	Decimals     int64    `json:"decimals"`
	ContractName string   `json:"contractName"`
	ContractAddr string   `json:"contractAddr"`
	Succtime     int64    `json:"succtime"`
	Dealstate    int64    `json:"dealstate"`
	Notifystate  int64    `json:"notifystate"`
	Success      string   `json:"success"`
	BalanceMode  int64    `json:"balanceMode"`
}

//easyjson:json
type FindTradeLogReq struct {
	common.BaseReq
	WalletID    string `json:"walletID"`
	AccountID   string `json:"accountID"`
	TxID        string `json:"txID"`
	WxID        string `json:"wxID"`
	Sid         string `json:"sid"`
	Symbol      string `json:"symbol"`
	BlockHeight int64  `json:"blockHeight"`
	Type        int64  `json:"type"`
	IsMain      int64  `json:"isMain"`
	IsContract  int64  `json:"isContract"`
	ContractID  string `json:"contractID"`
	Address     string `json:"address"`
	UserID      int64  `json:"userID"`
	Sort        int    `json:"sort"`
}

//easyjson:json
type FindTradeLogRes struct {
	Result []TradeLogResult `json:"result"`
	Limit  sqlc.Limit       `json:"limit"`
}
