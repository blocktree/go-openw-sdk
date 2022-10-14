package main

import (
	"flag"
	"github.com/astaxie/beego/config"
	"github.com/blocktree/go-openw-sdk/v2/openwsdk"
	"github.com/blocktree/openwallet/v2/log"
	"github.com/blocktree/openwallet/v2/owtp"
	"github.com/google/uuid"
	"strings"
)

func init() {
	owtp.Debug = false
}

type Subscriber struct {
}

//OpenwNewTransactionNotify openw新交易单通知
func (s *Subscriber) OpenwNewTransactionNotify(transaction *openwsdk.Transaction, subscribeToken string) (bool, error) {
	log.Infof("OpenwNewTransactionNotify")
	log.Infof("---------------------------------")
	log.Infof("subscribeToken: %s", subscribeToken)
	log.Infof("Symbol: %+v", transaction.Symbol)
	log.Infof("contractID: %+v", transaction.ContractID)
	log.Infof("blockHash: %+v", transaction.BlockHash)
	log.Infof("blockHeight: %+v", transaction.BlockHeight)
	log.Infof("txid: %+v", transaction.TxID)
	log.Infof("amount: %+v", transaction.Amount)
	log.Infof("accountID: %+v", transaction.AccountID)
	log.Infof("fees: %+v", transaction.Fees)
	log.Infof("---------------------------------")
	return true, nil
}

//OpenwNewBlockNotify openw新区块头通知
func (s *Subscriber) OpenwNewBlockNotify(blockHeader *openwsdk.BlockHeader, subscribeToken string) (bool, error) {
	log.Infof("OpenwNewBlockNotify")
	log.Infof("---------------------------------")
	log.Infof("subscribeToken: %s", subscribeToken)
	log.Infof("Symbol: %+v", blockHeader.Symbol)
	log.Infof("blockHash: %+v", blockHeader.Hash)
	log.Infof("blockHeight: %+v", blockHeader.Height)
	log.Infof("---------------------------------")
	return true, nil
}

//OpenwBalanceUpdateNotify openw余额更新
func (s *Subscriber) OpenwBalanceUpdateNotify(balance *openwsdk.Balance, tokenBalance *openwsdk.TokenBalance, subscribeToken string) (bool, error) {
	log.Infof("OpenwBalanceUpdateNotify")
	log.Infof("---------------------------------")
	log.Infof("subscribeToken: %s", subscribeToken)
	log.Infof("Symbol: %+v", balance.Symbol)
	log.Infof("Balance: %+v", balance.Balance)
	log.Infof("")
	log.Infof("Token: %+v", tokenBalance.Token)
	log.Infof("Balance: %+v", tokenBalance.Balance)
	log.Infof("---------------------------------")
	return true, nil
}

//OpenwNewSmartContractReceiptNotify 智能合约交易回执通知
func (s *Subscriber) OpenwNewSmartContractReceiptNotify(receipt *openwsdk.SmartContractReceipt, subscribeToken string) (bool, error) {
	log.Infof("OpenwNewSmartContractReceiptNotify")
	log.Infof("---------------------------------")
	log.Infof("subscribeToken: %s", subscribeToken)
	log.Infof("Symbol: %+v", receipt.Symbol)
	log.Infof("contractID: %+v", receipt.ContractID)
	log.Infof("blockHash: %+v", receipt.BlockHash)
	log.Infof("blockHeight: %+v", receipt.BlockHeight)
	log.Infof("txid: %+v", receipt.TxID)
	log.Infof("rawReceipt: %+v", receipt.RawReceipt)
	log.Infof("value: %+v", receipt.Value)
	log.Infof("fees: %+v", receipt.Fees)
	for i, event := range receipt.Events {
		log.Std.Notice("events[%d]: %+v", i, event)
	}
	log.Infof("---------------------------------")
	return true, nil
}

func main() {

	var (
		endRunning = make(chan bool, 1)
	)

	confFile := flag.String("c", "", "callback node config file")
	address := flag.String("ip", "", "callback node IP")

	flag.Parse()

	log.Infof("confFile: %s", *confFile)
	log.Infof("address: %s", *address)

	if *address == "" {
		return
	}

	network := strings.Split(*address, ":")
	port := network[1]

	c, err := config.NewConfig("ini", *confFile)
	if err != nil {
		log.Error("NewConfig error:", err)
		return
	}

	AppID := c.String("AppID")
	AppKey := c.String("AppKey")
	Host := c.String("Host")
	PrivateKey := c.String("PrivateKey")
	EnableKeyAgreement, _ := c.Bool("EnableKeyAgreement")
	EnableSignature, _ := c.Bool("EnableSignature")
	EnableSSL, _ := c.Bool("EnableSSL")
	ConnectType := c.String("ConnectType")
	//owtp.RandomPrivateKey()
	cert, _ := owtp.NewCertificate(PrivateKey)

	config := &openwsdk.APINodeConfig{
		AppID:              AppID,
		AppKey:             AppKey,
		Host:               Host,
		Cert:               cert,
		ConnectType:        owtp.HTTP,
		EnableKeyAgreement: EnableKeyAgreement,
		EnableSSL:          EnableSSL,
		TimeoutSEC:         60,
	}

	api, err := openwsdk.NewAPINodeWithError(config)
	if err != nil {
		log.Errorf("NewAPINode unexpected error: %v", err)
		return
	}

	err = api.BindAppDevice()
	if err != nil {
		log.Errorf("BindAppDevice unexpected error: %v", err)
		return
	}

	log.Debug("NodeID:", api.NodeID())

	subscribeToken := uuid.New().String()

	err = api.Subscribe(
		[]string{
			openwsdk.SubscribeToTrade,
			openwsdk.SubscribeToBlock,
			openwsdk.SubscribeToAccount,
			openwsdk.SubscribeToSmartContractReceipt,
		},
		":"+port,
		openwsdk.CallbackModeNewConnection, openwsdk.CallbackNode{
			NodeID:             api.NodeID(),
			Address:            *address,
			ConnectType:        ConnectType,
			EnableKeyAgreement: EnableKeyAgreement,
			EnableSignature:    EnableSignature,
		},
		subscribeToken)
	if err != nil {
		log.Errorf("Subscribe unexpected error: %v", err)
		return
	}

	subscriber := &Subscriber{}
	api.AddObserver(subscriber)

	<-endRunning

}
