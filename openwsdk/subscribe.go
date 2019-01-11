package openwsdk

import (
	"encoding/json"
	"github.com/blocktree/OpenWallet/log"
	"github.com/blocktree/OpenWallet/owtp"
)

//OpenwNotificationObject openw-server的通知对象
type OpenwNotificationObject interface {

	//OpenwNewTransactionNotify openw新交易单通知
	OpenwNewTransactionNotify(transaction *Transaction) (bool, error)
}

func (api *APINode) subscribeToAccount(ctx *owtp.Context) {
	log.Info("params:", ctx.Params())
	ctx.Response(nil, owtp.StatusSuccess, "subscribeToAccount is not implemented")
}

//subscribeToTrade 处理新交易记录通知
func (api *APINode) subscribeToTrade(ctx *owtp.Context) {
	data := ctx.Params().Get("tradelog")

	var accepted bool
	var tx Transaction
	err := json.Unmarshal([]byte(data.Raw), &tx)
	if err != nil {
		accepted = false
	} else {
		for o, _ := range api.observers {
			accepted, err = o.OpenwNewTransactionNotify(&tx)
			if err != nil {
				accepted = false
			}
			if accepted == false {
				break
			}
		}
	}

	ctx.Response(map[string]interface{}{
		"accepted": accepted,
	}, owtp.StatusSuccess, err.Error())
}