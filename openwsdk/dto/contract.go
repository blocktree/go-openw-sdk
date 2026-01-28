package dto

import (
	"github.com/godaddy-x/freego/node/common"
	"github.com/godaddy-x/freego/ormx/sqlc"
)

//easyjson:json
type ContractResult struct {
	ID         int64  `json:"id"`
	ContractID string `json:"contractID"`
	Symbol     string `json:"symbol"`
	Name       string `json:"name"`
	Decimals   int64  `json:"decimals"`
	Address    string `json:"address"`
	Token      string `json:"token"`
	Protocol   string `json:"protocol"`
	Ctime      int64  `json:"ctime"`
}

//easyjson:json
type GetContractsReq struct {
	common.BaseReq
	Symbol     string `json:"symbol"`
	ContractID string `json:"contractID"`
	MainSymbol string `json:"mainSymbol"`
	Sort       int    `json:"sort"`
}

//easyjson:json
type GetContractsRes struct {
	Result []ContractResult `json:"result"`
	Limit  sqlc.Limit       `json:"limit"`
}

//type SmartContractEvent struct {
//	Symbol       string `json:"symbol"`
//	ContractID   string `json:"contractID"`
//	ContractName string `json:"contractName"`
//	ContractAddr string `json:"contractAddr"`
//	Event        string `json:"event"`
//	Value        string `json:"value"`
//}
//
//type ReceiptResult struct {
//	ID           int64                `json:"id"`
//	AppID        string               `json:"appID"`
//	WalletID     string               `json:"walletID"`
//	AccountID    string               `json:"accountID"`
//	Sid          string               `json:"sid"`
//	TxID         string               `json:"txID"`
//	WxID         string               `json:"wxID"`
//	FromAddress  string               `json:"fromAddress"`
//	ToAddress    string               `json:"toAddress"`
//	Value        string               `json:"value"`
//	Fees         string               `json:"fees"`
//	Symbol       string               `json:"symbol"`
//	ContractID   string               `json:"contractID"`
//	ContractName string               `json:"contractName"`
//	ContractAddr string               `json:"contractAddr"`
//	BlockHash    string               `json:"blockHash"`
//	BlockHeight  int64                `json:"blockHeight"`
//	IsMain       int64                `json:"isMain"`
//	Applytime    int64                `json:"applytime"`
//	Succtime     int64                `json:"succtime"`
//	Dealstate    int64                `json:"dealstate"`
//	Notifystate  int64                `json:"notifystate"`
//	Success      string               `json:"success"`
//	RawReceipt   string               `json:"rawReceipt"`
//	Events       []SmartContractEvent `json:"events"`
//}

//type SubmitSmartContractTradeReq struct {
//	common.BaseReq
//	AppID  string `json:"appID"`
//	RawTx  []byte `json:"rawTx"`
//	UserID int64  `json:"userID"`
//}
//
//type SubmitSmartContractTradeRes struct {
//	Result []byte `json:"result"`
//}
//
//type CallSmartContractABIRes struct {
//	Method    string `json:"method"`
//	Value     string `json:"value"`
//	RawHex    string `json:"rawHex"`
//	Status    int64  `json:"status"`
//	Exception string `json:"exception"`
//}

//type FindSmartContractReceiptReq struct {
//	common.BaseReq
//	TxID         string `json:"txID"`
//	Symbol       string `json:"symbol"`
//	ContractID   string `json:"contractID"`
//	ContractAddr string `json:"contractAddr"`
//	Address      string `json:"address"`
//	Type         int64  `json:"type"`
//	AppID        string `json:"appID"`
//	UserID       int64  `json:"userID"`
//}
//
//type FindSmartContractReceiptRes struct {
//	Result []ReceiptResult `json:"result"`
//}
//
//type FollowSmartContractReceiptReq struct {
//	common.BaseReq
//	AppID           string   `json:"appID"`
//	FollowContracts []string `json:"followContracts"`
//	Type            int64    `json:"type"`
//	UserID          int64    `json:"userID"`
//}
//
//type FollowSmartContractReceiptRes struct {
//	Result bool `json:"result"`
//}
