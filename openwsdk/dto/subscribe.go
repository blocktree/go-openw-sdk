package dto

import (
	"github.com/godaddy-x/freego/node/common"
)

//easyjson:json
type CreateSubscribeReq struct {
	common.BaseReq
	SubscribeMethod   []string `json:"subscribeMethod"`
	SubscribeContract []string `json:"subscribeContract"`
}

//easyjson:json
type CreateSubscribeRes struct {
	Result bool `json:"result"`
}

//easyjson:json
type FindTradePushListRes struct {
	Result []TradePushResult `json:"result"`
}

//easyjson:json
type FindTradePushListReq struct {
	common.BaseReq
}

//easyjson:json
type TradePushResult struct {
	ID       int64  `json:"id"`
	AppID    string `json:"appID"`
	Data     string `json:"data"`
	DataSign string `json:"dataSign"`
	Ctime    int64  `json:"ctime"`
}

//easyjson:json
type FindTradeBalancePushListRes struct {
	Result []TradeBalancePushResult `json:"result"`
}

//easyjson:json
type FindTradeBalancePushListReq struct {
	common.BaseReq
}

//easyjson:json
type TradeBalancePushResult struct {
	ID       int64  `json:"id"`
	AppID    string `json:"appID"`
	Data     string `json:"data"`
	DataSign string `json:"dataSign"`
	Ctime    int64  `json:"ctime"`
}

//easyjson:json
type ConfirmPushDataRes struct {
	Confirmed int64 `json:"confirmed"`
}

//easyjson:json
type ConfirmPushDataReq struct {
	common.BaseReq
	DataType string  `json:"dataType"` // 数据类型 [Transfer, Balance]
	DataList []int64 `json:"dataList"` // 数据ID列表
}
