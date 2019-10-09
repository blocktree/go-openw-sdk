package openwsdk

import (
	"fmt"
	"github.com/blocktree/openwallet/log"
	"github.com/blocktree/openwallet/owtp"
	"testing"
)

type Subscriber struct {
}

//OpenwNewTransactionNotify openw新交易单通知
func (s *Subscriber) OpenwNewTransactionNotify(transaction *Transaction) (bool, error) {
	log.Infof("Symbol: %+v", transaction.Symbol)
	log.Infof("contractID: %+v", transaction.ContractID)
	log.Infof("blockHash: %+v", transaction.BlockHash)
	log.Infof("blockHeight: %+v", transaction.BlockHeight)
	log.Infof("txid: %+v", transaction.Txid)
	log.Infof("amount: %+v", transaction.Amount)
	log.Infof("accountID: %+v", transaction.AccountID)
	log.Infof("fees: %+v", transaction.Fees)
	log.Infof("---------------------------------")
	return true, nil
}

//OpenwNewBlockNotify openw新区块头通知
func (s *Subscriber) OpenwNewBlockNotify(blockHeader *BlockHeader) (bool, error) {
	log.Infof("Symbol: %+v", blockHeader.Symbol)
	log.Infof("blockHash: %+v", blockHeader.Hash)
	log.Infof("blockHeight: %+v", blockHeader.Height)
	log.Infof("---------------------------------")
	return true, nil
}

//OpenwBalanceUpdateNotify openw余额更新
func (s *Subscriber) OpenwBalanceUpdateNotify(balance *Balance, tokenBalance *TokenBalance) (bool, error) {
	log.Infof("Symbol: %+v", balance.Symbol)
	log.Infof("Balance: %+v", balance.Balance)
	log.Infof("Token: %+v", tokenBalance.Token)
	log.Infof("Balance: %+v", tokenBalance.Balance)
	log.Infof("---------------------------------")
	return true, nil
}

func TestAPINode_Subscribe(t *testing.T) {

	var (
		endRunning = make(chan bool, 1)
	)

	api := testNewAPINode()
	log.Debug("NodeID:", api.NodeID())
	err := api.Subscribe(
		[]string{
			SubscribeToTrade,
			SubscribeToBlock,
		},
		":9322",
		CallbackModeNewConnection, CallbackNode{
			NodeID:             api.NodeID(),
			Address:            "192.168.27.179:9322",
			ConnectType:        owtp.Websocket,
			EnableKeyAgreement: false,
			EnableSSL:          true,
		})
	if err != nil {
		t.Logf("Subscribe unexpected error: %v\n", err)
		return
	}

	subscriber := &Subscriber{}
	api.AddObserver(subscriber)

	<-endRunning
}

func TestAPINode_Listener(t *testing.T) {

	//var (
	//	endRunning = make(chan bool, 1)
	//)
	//
	//api := testNewAPINode()
	//api.node.Listen(owtp.ConnectConfig{
	//	Address:     ":9322",
	//	ConnectType: owtp.HTTP,
	//})
	//
	//<-endRunning

	////等待推送
	//time.Sleep(5 * time.Second)
	//
	//api.RemoveObserver(subscriber)
	//
	////等待推送
	//time.Sleep(5 * time.Second)
}

func TestAPINode_Call(t *testing.T) {

	nodeID := "APINode_Listener"

	config := owtp.ConnectConfig{
		Address:            "127.0.0.1:8422",
		ConnectType:        owtp.HTTP,
		EnableSSL:          false,
		EnableKeyAgreement: true,
	}
	wsClient := owtp.RandomOWTPNode()
	_, err := wsClient.Connect(nodeID, config)
	if err != nil {
		t.Errorf("Connect unexcepted error: %v", err)
		return
	}

	params := map[string]interface{}{
		"name": "chance",
		"age":  18,
	}
	//err = wsClient.Connect(wsHostNodeID, config)
	err = wsClient.ConnectAndCall(nodeID, config, "subscribeToTrade", params, true, func(resp owtp.Response) {

		result := resp.JsonData()
		symbols := result.Get("symbols")
		fmt.Printf("symbols: %v\n", symbols)
	})

	if err != nil {
		t.Errorf("unexcepted error: %v", err)
		return
	}
}
