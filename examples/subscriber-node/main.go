package main

import (
	"github.com/astaxie/beego/config"
	"github.com/blocktree/go-openw-sdk/openwsdk"
	"github.com/blocktree/openwallet/log"
	"github.com/blocktree/openwallet/owtp"
	"path/filepath"
)

type Subscriber struct {
}

//OpenwNewTransactionNotify openw新交易单通知
func (s *Subscriber) OpenwNewTransactionNotify(transaction *openwsdk.Transaction) (bool, error) {
	log.Infof("OpenwNewTransactionNotify")
	log.Infof("---------------------------------")
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
func (s *Subscriber) OpenwNewBlockNotify(blockHeader *openwsdk.BlockHeader) (bool, error) {
	log.Infof("OpenwNewBlockNotify")
	log.Infof("---------------------------------")
	log.Infof("Symbol: %+v", blockHeader.Symbol)
	log.Infof("blockHash: %+v", blockHeader.Hash)
	log.Infof("blockHeight: %+v", blockHeader.Height)
	log.Infof("---------------------------------")
	return true, nil
}

func testNewAPINode() (*openwsdk.APINode, error) {

	confFile := filepath.Join("conf", "node.ini")

	c, err := config.NewConfig("ini", confFile)
	if err != nil {
		log.Error("NewConfig error:", err)
		return nil, nil
	}

	AppID := c.String("AppID")
	AppKey := c.String("AppKey")
	Host := c.String("Host")

	cert, _ := owtp.NewCertificate(owtp.RandomPrivateKey())

	config := &openwsdk.APINodeConfig{
		AppID:              AppID,
		AppKey:             AppKey,
		Host:               Host,
		Cert:               cert,
		ConnectType:        owtp.HTTP,
		EnableSignature:    false,
		EnableKeyAgreement: false,
		TimeoutSEC:         60,
	}

	api, err := openwsdk.NewAPINodeWithError(config)
	if err != nil {
		return nil, err
	}

	err = api.BindAppDevice()
	if err != nil {
		return nil, err
	}

	return api, nil
}

func main() {

	var (
		endRunning = make(chan bool, 1)
	)

	api, err := testNewAPINode()
	log.Debug("NodeID:", api.NodeID())
	if err != nil {
		log.Errorf("NewAPINode unexpected error: %v", err)
		return
	}

	err = api.Subscribe(
		[]string{
			openwsdk.SubscribeToTrade,
			openwsdk.SubscribeToBlock,
		},
		":30020",
		openwsdk.CallbackModeNewConnection, openwsdk.CallbackNode{
			NodeID:             api.NodeID(),
			Address:            "120.78.83.180:30020",
			ConnectType:        owtp.Websocket,
			EnableKeyAgreement: false,
		})
	if err != nil {
		log.Errorf("Subscribe unexpected error: %v", err)
		return
	}

	subscriber := &Subscriber{}
	api.AddObserver(subscriber)

	<-endRunning

}
