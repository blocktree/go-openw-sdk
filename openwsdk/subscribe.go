package openwsdk

import (
	"encoding/json"
	"fmt"
	"github.com/blocktree/openwallet/v2/log"
	"github.com/blocktree/openwallet/v2/owtp"
)

const (
	SubscribeToAccount              = "subscribeToAccount"              //订阅余额更新通信
	SubscribeToTrade                = "subscribeToTrade"                //订阅新交易单通知
	SubscribeToBlock                = "subscribeToBlock"                //订阅新区块链头通知
	SubscribeToSmartContractReceipt = "subscribeToSmartContractReceipt" //订阅智能合约交易回执通知
)

//OpenwNotificationObject openw-server的通知对象
type OpenwNotificationObject interface {

	//OpenwNewTransactionNotify openw新交易单通知
	OpenwNewTransactionNotify(transaction *Transaction, subscribeToken string) (bool, error)

	//OpenwNewBlockNotify openw新区块头通知
	OpenwNewBlockNotify(blockHeader *BlockHeader, subscribeToken string) (bool, error)

	//OpenwBalanceUpdateNotify openw余额更新
	OpenwBalanceUpdateNotify(balance *Balance, tokenBalance *TokenBalance, subscribeToken string) (bool, error)

	//OpenwNewSmartContractReceiptNotify 智能合约交易回执通知
	OpenwNewSmartContractReceiptNotify(receipt *SmartContractReceipt, subscribeToken string) (bool, error)
}

//ServeNotification 开启监听服务，接收通知
func (api *APINode) ServeNotification(listenAddr string, connectType string) error {

	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}

	//开启监听
	log.Infof("%s start to listen [%s] connection...", listenAddr, connectType)
	return api.node.Listen(owtp.ConnectConfig{
		Address:     listenAddr,
		ConnectType: connectType,
	})
}

//StopServeNotification 关闭监听通知
func (api *APINode) StopServeNotification(connectType string) {
	log.Infof("API Node close listener [%s] connection...", connectType)
	api.node.CloseListener(connectType)
}

//AddObserver 添加观测者
func (api *APINode) AddObserver(obj OpenwNotificationObject) error {
	api.mu.Lock()

	defer api.mu.Unlock()

	if obj == nil {
		return nil
	}
	if _, exist := api.observers[obj]; exist {
		//已存在，不重复订阅
		return nil
	}

	api.observers[obj] = true

	return nil
}

//RemoveObserver 移除观测者
func (api *APINode) RemoveObserver(obj OpenwNotificationObject) error {
	api.mu.Lock()
	defer api.mu.Unlock()

	delete(api.observers, obj)

	return nil
}

func (api *APINode) subscribeToAccount(ctx *owtp.Context) {
	data := ctx.Params()

	var (
		msg      string
		accepted bool
		err      error
	)

	if ctx.Peer.PID() != api.subscribeInfo.notifierNodeID {
		ctx.Response(map[string]interface{}{
			"accepted": false,
		}, owtp.StatusSuccess, msg)
		log.Warningf("get balance update notify by unknown notifier NodeID: %s", ctx.Peer.PID())
		return
	}

	balance := NewBalance(data)
	tokenBalance := NewTokenBalance(data.Get("tokenBalance"))
	subscribeToken := data.Get("subscribeToken").String()
	for o, _ := range api.observers {
		accepted, err = o.OpenwBalanceUpdateNotify(balance, tokenBalance, subscribeToken)
		if err != nil {
			msg = err.Error()
			accepted = false
		}
		if accepted == false {
			break
		}
	}

	ctx.Response(map[string]interface{}{
		"accepted": accepted,
	}, owtp.StatusSuccess, msg)

}

//subscribeToTrade 处理新交易记录通知
func (api *APINode) subscribeToTrade(ctx *owtp.Context) {
	data := ctx.Params()

	var msg string
	var accepted bool
	var tx Transaction

	if ctx.Peer.PID() != api.subscribeInfo.notifierNodeID {
		ctx.Response(map[string]interface{}{
			"accepted": false,
		}, owtp.StatusSuccess, msg)
		log.Warningf("get transaction notify by unknown notifier NodeID: %s", ctx.Peer.PID())
		return
	}

	subscribeToken := data.Get("subscribeToken").String()
	err := json.Unmarshal([]byte(data.Raw), &tx)
	if err != nil {
		accepted = false
	} else {
		for o, _ := range api.observers {
			accepted, err = o.OpenwNewTransactionNotify(&tx, subscribeToken)
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

	if ctx.Peer.PID() != api.subscribeInfo.notifierNodeID {
		ctx.Response(map[string]interface{}{
			"accepted": false,
		}, owtp.StatusSuccess, msg)
		log.Warningf("get new block notify by unknown notifier NodeID: %s", ctx.Peer.PID())
		return
	}

	subscribeToken := data.Get("subscribeToken").String()
	err := json.Unmarshal([]byte(data.Raw), &header)
	if err != nil {
		accepted = false
	} else {
		for o, _ := range api.observers {
			accepted, err = o.OpenwNewBlockNotify(&header, subscribeToken)
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

//subscribeToSmartContractReceipt 处理新智能合约交易回执通知
func (api *APINode) subscribeToSmartContractReceipt(ctx *owtp.Context) {
	data := ctx.Params()

	var msg string
	var accepted bool
	var receipt SmartContractReceipt

	if ctx.Peer.PID() != api.subscribeInfo.notifierNodeID {
		ctx.Response(map[string]interface{}{
			"accepted": false,
		}, owtp.StatusSuccess, msg)
		log.Warningf("get smart contract receipt notify by unknown notifier NodeID: %s", ctx.Peer.PID())
		return
	}

	subscribeToken := data.Get("subscribeToken").String()
	err := json.Unmarshal([]byte(data.Raw), &receipt)
	if err != nil {
		accepted = false
	} else {
		for o, _ := range api.observers {
			accepted, err = o.OpenwNewSmartContractReceiptNotify(&receipt, subscribeToken)
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


func (api *APINode) checkNodeIsOnline(ctx *owtp.Context) {
	ctx.Response(nil, owtp.StatusSuccess, "success")
}