package performance_test

import (
	"github.com/blocktree/go-openw-sdk/v2/openwsdk"
	"github.com/blocktree/openwallet/v2/log"
	"github.com/blocktree/openwallet/v2/owtp"
	"testing"
)

type Subscriber struct{}

//OpenwNewTransactionNotify openw新交易单通知
func (s *Subscriber) OpenwNewTransactionNotify(transaction *openwsdk.Transaction, subscribeToken string) (bool, error) {
	log.Info("Symbol: %+v", transaction.Symbol)
	log.Infof("contractID: %+v", transaction.ContractID)
	log.Infof("blockHash: %+v", transaction.BlockHash)
	log.Infof("blockHeight: %+v", transaction.BlockHeight)
	log.Infof("txid: %+v", transaction.TxID)
	log.Infof("amount: %+v", transaction.Amount)
	log.Infof("accountID: %+v", transaction.AccountID)
	log.Infof("fees: %+v", transaction.Fees)
	log.Infof("subscribeToken: %s", subscribeToken)
	log.Infof("---------------------------------")
	return true, nil
}

//OpenwNewBlockNotify openw新区块头通知
func (s *Subscriber) OpenwNewBlockNotify(blockHeader *openwsdk.BlockHeader, subscribeToken string) (bool, error) {
	log.Infof("Symbol: %+v", blockHeader.Symbol)
	log.Infof("blockHash: %+v", blockHeader.Hash)
	log.Infof("blockHeight: %+v", blockHeader.Height)
	log.Infof("subscribeToken: %s", subscribeToken)
	log.Infof("---------------------------------")
	return true, nil
}

//OpenwBalanceUpdateNotify openw余额更新
func (s *Subscriber) OpenwBalanceUpdateNotify(balance *openwsdk.Balance, tokenBalance *openwsdk.TokenBalance, subscribeToken string) (bool, error) {
	log.Infof("Symbol: %+v", balance.Symbol)
	log.Infof("Balance: %+v", balance.Balance)
	log.Infof("Token: %+v", tokenBalance.Token)
	log.Infof("Balance: %+v", tokenBalance.Balance)
	log.Infof("subscribeToken: %s", subscribeToken)
	log.Infof("---------------------------------")
	return true, nil
}

//OpenwNewSmartContractReceiptNotify 智能合约交易回执通知
func (s *Subscriber) OpenwNewSmartContractReceiptNotify(receipt *openwsdk.SmartContractReceipt, subscribeToken string) (bool, error) {
	return true, nil
}

////OpenwNFTTransferNotify NFT合约交易数据通知
func (s *Subscriber) OpenwNFTTransferNotify(transfer *openwsdk.NFTTransfer, subscribeToken string) (bool, error) {
	return true, nil
}

// 订阅方法列表
func TestAPINode_Subscribe(t *testing.T) {
	err := api.Subscribe(
		[]string{
			openwsdk.SubscribeToTrade,
			openwsdk.SubscribeToBlock,
			openwsdk.SubscribeToNFTTransfer,
		},
		":9322",
		openwsdk.CallbackModeNewConnection, openwsdk.CallbackNode{
			NodeID:             api.NodeID(),
			Address:            "127.0.0.1:9322",
			ConnectType:        owtp.HTTP,
			EnableKeyAgreement: false,
			EnableSSL:          false,
			EnableSignature:    false,
		},
		"hello world")
	if err != nil {
		t.Logf("Subscribe unexpected error: %v\n", err)
		return
	}
	api.AddObserver(&Subscriber{})
	select {}
}
