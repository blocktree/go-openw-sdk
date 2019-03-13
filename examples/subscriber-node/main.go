package main

import (
	"github.com/blocktree/openwallet/log"
	"github.com/blocktree/openwallet/owtp"
	"github.com/blocktree/go-openw-sdk/openwsdk"
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

	//--------------- PRIVATE KEY ---------------
	//CaeQzossEasxDmDx4sS12eQC2L7zzNGVwEW2T1CKK3ZS
	//--------------- PUBLIC KEY ---------------
	//3Gve895o6aarxYzgLu8tKy3EXVFmFw6oFh1dbpVXmy8VtRaxa6tzpKRPc568549Q5jLpNJGbkXY5HqoQH5gvbg6o
	//--------------- NODE ID ---------------
	//4YBHa3d3vAceSRngPWrsm1cSPJudFQSzNAhPGschFw47

	//cert, _ := owtp.NewCertificate("CaeQzossEasxDmDx4sS12eQC2L7zzNGVwEW2T1CKK3ZS")
	cert, _ := owtp.NewCertificate(owtp.RandomPrivateKey())

	config := &openwsdk.APINodeConfig{
		AppID:  "8df7420d3917afa0172ea9c85e07ab55",
		AppKey: "faa14b5e2cf119cd6d38bda45b49eb02b333a1b1ff6f10703acb554011ebfb1e",
		Host:   "120.78.83.180",
		//AppID:  "b4b1962d415d4d30ec71b28769fda585",
		//AppKey: "8c511cb683041f3589419440fab0a7b7710907022b0d035baea9001d529ca72f",
		//Host: "192.168.27.181:8422",
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
