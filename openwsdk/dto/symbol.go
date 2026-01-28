package dto

import "github.com/godaddy-x/freego/node/common"

//easyjson:json
type SymbolResult struct {
	MaxHeight         int64  `json:"maxHeight"`
	Curve             int64  `json:"curve"`
	Symbol            string `json:"symbol"`
	Decimals          int64  `json:"decimals"`
	Confirm           int64  `json:"confirm"`
	BalanceMode       int64  `json:"balanceMode"`
	Icon              string `json:"icon"`
	SupportMemo       int64  `json:"supportMemo"`
	OnlyContract      int64  `json:"onlyContract"`
	WithdrawStop      int64  `json:"withdrawStop"`
	BlockStop         int64  `json:"blockStop"`
	FeeRate           string `json:"feeRate"`
	Unit              string `json:"unit"`
	MainSymbol        string `json:"mainSymbol"`
	Name              string `json:"name"`
	Hash              string `json:"hash"`
	Merkleroot        string `json:"merkleroot"`
	Previousblockhash string `json:"previousblockhash"`
	Height            int64  `json:"height"`
	Version           int64  `json:"version"`
	Time              int64  `json:"time"`
	Fork              bool   `json:"fork"`
	Confirmations     string `json:"confirmations"`
}

//easyjson:json
type SymbolBlockListReq struct {
	common.BaseReq
	Symbol string `json:"symbol"`
}

//easyjson:json
type SymbolBlockListRes struct {
	Result []SymbolResult `json:"result"`
}

//easyjson:json
type GetBlockStatusReq struct {
	common.BaseReq
	Symbol string `json:"symbol"`
}

//easyjson:json
type GetBlockStatusRes struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
}
