package openwsdk

import (
	"github.com/blocktree/OpenWallet/owtp"
	"testing"
)

type Subscriber struct {
}

//OpenwNewTransactionNotify openw新交易单通知
func (s *Subscriber) OpenwNewTransactionNotify(transaction *Transaction) (bool, error) {
	return true, nil
}

func TestAPINode_Subscribe(t *testing.T) {

	var (
		endRunning = make(chan bool, 1)
	)

	api := testNewAPINode()
	api.Subscribe(CallbackModeNewConnection, CallbackNode{
		NodeID:             api.node.NodeID(),
		Address:            "192.168.27.181:9322",
		ConnectType:        owtp.HTTP,
		EnableKeyAgreement: false,
	})

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

	//nodeID := "APINode_Listener"
	//
	//config := owtp.ConnectConfig{
	//	Address:     ":9322",
	//	ConnectType: owtp.HTTP,
	//}
	//wsClient := owtp.RandomOWTPNode()
	//err := wsClient.Connect(nodeID, config)
	//if err != nil {
	//	t.Errorf("Connect unexcepted error: %v", err)
	//	return
	//}

	//params := map[string]interface{}{
	//	"name": "chance",
	//	"age":  18,
	//}
	//err = wsClient.Connect(wsHostNodeID, config)
	//err := wsClient.ConnectAndCall(nodeID, config, "subscribeToAccount", params, true, func(resp owtp.Response) {
	//
	//	result := resp.JsonData()
	//	symbols := result.Get("symbols")
	//	fmt.Printf("symbols: %v\n", symbols)
	//})
	//
	//if err != nil {
	//	t.Errorf("unexcepted error: %v", err)
	//	return
	//}
}
