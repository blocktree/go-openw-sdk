package openwsdk

import (
	"encoding/json"
	"github.com/blocktree/OpenWallet/log"
	"github.com/blocktree/OpenWallet/owtp"
)

const (
	SubscribeToAccount = "subscribeToAccount"
	SubscribeToTrade = "subscribeToTrade"
	SubscribeToBlock = "subscribeToBlock"
)

//OpenwNotificationObject openw-server的通知对象
type OpenwNotificationObject interface {

	//OpenwNewTransactionNotify openw新交易单通知
	OpenwNewTransactionNotify(transaction *Transaction) (bool, error)

	//OpenwNewBlockNotify openw新区块头通知
	OpenwNewBlockNotify(blockHeader *BlockHeader) (bool, error)
}

func (api *APINode) subscribeToAccount(ctx *owtp.Context) {
	log.Info("params:", ctx.Params())
	ctx.Response(nil, owtp.StatusSuccess, "subscribeToAccount is not implemented")
}

//subscribeToTrade 处理新交易记录通知
func (api *APINode) subscribeToTrade(ctx *owtp.Context) {
	data := ctx.Params()

	var msg string
	var accepted bool
	var tx Transaction
	err := json.Unmarshal([]byte(data.Raw), &tx)
	if err != nil {
		accepted = false
	} else {
		for o, _ := range api.observers {
			accepted, err = o.OpenwNewTransactionNotify(&tx)
			if err != nil {
				msg = err.Error()
				accepted = false
			}
			if accepted == false {
				break
			}
		}
	}

	ctx.Response(map[string]interface{}{
		"accepted": accepted,
	}, owtp.StatusSuccess, msg)
}


//subscribeToBlock 处理新区块头通知
func (api *APINode) subscribeToBlock(ctx *owtp.Context) {
	data := ctx.Params()

	var msg string
	var accepted bool
	var header BlockHeader
	err := json.Unmarshal([]byte(data.Raw), &header)
	if err != nil {
		accepted = false
	} else {
		for o, _ := range api.observers {
			accepted, err = o.OpenwNewBlockNotify(&header)
			if err != nil {
				msg = err.Error()
				accepted = false
			}
			if accepted == false {
				break
			}
		}
	}

	ctx.Response(map[string]interface{}{
		"accepted": accepted,
	}, owtp.StatusSuccess, msg)
}