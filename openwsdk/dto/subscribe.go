package dto

import "github.com/godaddy-x/freego/node/common"

//easyjson:json
type CreateSubscribeReq struct {
	common.BaseReq
	SubscribeMethod   []string `json:"subscribeMethod"`
	SubscribeContract []string `json:"subscribeContract"`
}

type CreateSubscribeRes struct {
	Result bool `json:"result"`
}
