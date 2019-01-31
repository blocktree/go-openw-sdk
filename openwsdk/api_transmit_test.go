package openwsdk

import (
	"github.com/blocktree/OpenWallet/log"
	"testing"
)

func TestAPINode_ServeTransmitNode(t *testing.T) {
	var (
		endRunning = make(chan bool, 1)
	)

	api := testNewAPINode()
	err := api.ServeTransmitNode(9088)
	if err != nil {
		t.Logf("ServeTransmitNode error: %v\n", err)
		return
	}

	tn, err := api.TransmitNode()
	if err != nil {
		t.Logf("TransmitNode error: %v\n", err)
		return
	}
	
	tn.SetConnectHandler(func(transmitNode *TransmitNode, nodeInfo *TrustNodeInfo) {
		log.Infof("nodeInfo: %v", nodeInfo)

		//创建钱包
		wallet := testCreateWalletViaTrustNode(transmitNode, nodeInfo.NodeID)

		//创建账户
		testCreateAccountViaTrustNode(transmitNode, nodeInfo.NodeID, wallet.WalletID)


	})

	<-endRunning
}

func testCreateWalletViaTrustNode(transmitNode *TransmitNode, nodeID string) *Wallet {
	var retWallet *Wallet
	//创建钱包
	transmitNode.CreateWalletViaTrustNode(nodeID, "candy", "12345678", true,
		func(status uint64, msg string, wallet *Wallet) {
			if wallet != nil {
				log.Infof("wallet: %+v\n", wallet)
				retWallet = wallet
			}
		})
	return retWallet
}

func testCreateAccountViaTrustNode(transmitNode *TransmitNode, nodeID string, walletID string) *Account {
	var retAccount *Account
	//创建钱包
	transmitNode.CreateAccountViaTrustNode(nodeID, walletID, "candy", "12345678", "BTC", true,
		func(status uint64, msg string, account *Account, addresses []*Address) {
			if account != nil {
				log.Infof("account: %+v\n", account)
				retAccount = account
			}
		})
	return retAccount
}