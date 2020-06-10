package openwsdk

import (
	"encoding/json"
	"fmt"
	"github.com/blocktree/openwallet/v2/log"
	"github.com/blocktree/openwallet/v2/owtp"
	"github.com/google/uuid"
	"testing"
	"time"
)

func testServeTransmitNode(f func(transmitNode *TransmitNode, nodeInfo *TrustNodeInfo)) *TransmitNode {

	api := testNewAPINode()
	err := api.ServeTransmitNode("127.0.0.1:9088")
	if err != nil {
		log.Errorf("ServeTransmitNode error: %v\n", err)
		return nil
	}

	tn, err := api.TransmitNode()
	if err != nil {
		log.Errorf("TransmitNode error: %v\n", err)
		return nil
	}

	tn.SetConnectHandler(f)

	time.Sleep(8 * time.Second)

	return tn
}

func TestAPINode_ServeTransmitNode(t *testing.T) {
	testServeTransmitNode(func(transmitNode *TransmitNode, nodeInfo *TrustNodeInfo) {
		log.Infof("nodeInfo: %v", nodeInfo)
	})
}

func TestTransmitNode_GetTrustNodeInfo(t *testing.T) {
	testServeTransmitNode(func(transmitNode *TransmitNode, nodeInfo *TrustNodeInfo) {

		transmitNode.GetTrustNodeInfo(nodeInfo.NodeID, true,
			func(status uint64, msg string, nodeInfo *TrustNodeInfo) {
				log.Infof("nodeInfo: %v", nodeInfo)
			})
	})
}

func TestTransmitNode_GetTrustNodeInfoDirectCall(t *testing.T) {
	tn := testServeTransmitNode(func(transmitNode *TransmitNode, nodeInfo *TrustNodeInfo) {

	})

	nodeID := "4YBHa3d3vAceSRngPWrsm1cSPJudFQSzNAhPGschFw47"

	err := tn.GetTrustNodeInfo(nodeID, true,
		func(status uint64, msg string, nodeInfo *TrustNodeInfo) {
			log.Infof("nodeInfo: %v", nodeInfo)
		})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTransmitNode_CreateWalletViaTrustNode(t *testing.T) {

	alias := "openwallet"
	password := "12345678"

	testServeTransmitNode(func(transmitNode *TransmitNode, nodeInfo *TrustNodeInfo) {

		var newwallet *Wallet

		//创建钱包
		err := transmitNode.CreateWalletViaTrustNode(nodeInfo.NodeID, alias, password, true,
			func(status uint64, msg string, wallet *Wallet) {
				if wallet != nil {
					newwallet = wallet
					log.Infof("wallet: %+v\n", wallet)
				}
			})
		if err != nil {
			t.Errorf("CreateWalletViaTrustNode unexpected error: %v", err)
		}

		walletID := newwallet.WalletID
		alias := "openwallet_VSYS"
		password := "12345678"
		symbol := "VSYS"
		//创建账户
		err = transmitNode.CreateAccountViaTrustNode(nodeInfo.NodeID, walletID, alias, password, symbol, true,
			func(status uint64, msg string, account *Account, addresses []*Address) {
				if account != nil {
					log.Infof("account: %+v\n", account)
					for i, a := range addresses {
						log.Infof("address[%d]:%+v", i, a)
					}
				}
			})
		if err != nil {
			t.Errorf("CreateAccountViaTrustNode unexpected error: %v", err)
		}
	})
}

func TestTransmitNode_CreateAccountViaTrustNode(t *testing.T) {

	testServeTransmitNode(func(transmitNode *TransmitNode, nodeInfo *TrustNodeInfo) {

		walletID := "WFXtudgu9Q5ktpcfDPC8gVEbHF1t1QWiVV"
		alias := "openwallet_NAS"
		password := ""
		symbol := "NAS"
		//创建账户
		transmitNode.CreateAccountViaTrustNode(nodeInfo.NodeID, walletID, alias, password, symbol, true,
			func(status uint64, msg string, account *Account, addresses []*Address) {
				if account != nil {
					log.Infof("account: %+v\n", account)
					for i, a := range addresses {
						log.Infof("address[%d]:%+v", i, a)
					}
				}
			})
	})
}

func TestTransmitNode_SendTransactionViaTrustNode(t *testing.T) {

	testServeTransmitNode(func(transmitNode *TransmitNode, nodeInfo *TrustNodeInfo) {

		//accountID := "3i26MQmtuWVVnw8GnRCVopG3pi8MaYU6RqWVV2E1hwJx"
		//address := "mgCzMJDyJoqa6XE3RSdNGvD5Bi5VTWudRq"

		accountID := "AEeBZy321NzbLWPFkjyGfFyUn8hHX2g93f4wMBVumin4"
		address := "TRJJ9Mq4aMjdmKWpTDJAgbYNoY2P9Facg5"

		password := "12345678"
		sid := uuid.New().String()
		log.Infof("sid: %s", sid)
		transmitNode.SendTransactionViaTrustNode(nodeInfo.NodeID, accountID, password, sid,
			"", "0.998", address, "", "",
			true, func(status uint64, msg string, successTx []*Transaction, failedRawTxs []*FailedRawTransaction) {
				log.Infof("status: %d, msg: %s", status, msg)
				log.Info("============== success ==============")

				for _, tx := range successTx {
					log.Infof("tx: %+v", tx)
				}

				log.Info("")

				log.Info("============== fail ==============")

				for _, tx := range failedRawTxs {
					log.Infof("tx: %+v", tx.Reason)
				}

			})
	})
}

func TestTransmitNode_SetSummaryInfoViaTrustNode(t *testing.T) {
	testServeTransmitNode(func(transmitNode *TransmitNode, nodeInfo *TrustNodeInfo) {

		setting := &SummarySetting{
			WalletID:        "W3LxqTNAcXFqW7HGcTuERRLXKdNWu17Ccx",
			AccountID:       "GfxqbJdTFgPZKHjtoZQcygdpcdSUCd8dj7gX7B5yvFoj",
			SumAddress:      "0x50efe0a38381dfee9ab8947e81362199d3cf63d7",
			Threshold:       "30",
			MinTransfer:     "0.001",
			RetainedBalance: "0",
			Confirms:        0,
		}

		transmitNode.SetSummaryInfoViaTrustNode(nodeInfo.NodeID, setting, true, func(status uint64, msg string) {
			log.Infof("msg:%+v", msg)
		})
	})
}

func TestTransmitNode_FindSummaryInfoByWalletIDViaTrustNode(t *testing.T) {

	testServeTransmitNode(func(transmitNode *TransmitNode, nodeInfo *TrustNodeInfo) {

		walletID := "WN84dVZXpgVixsvXnU8jkFWD1qWHp15LpA"

		transmitNode.FindSummaryInfoByWalletIDViaTrustNode(nodeInfo.NodeID, walletID,
			true, func(status uint64, msg string, summarySettings []*SummarySetting) {
				for i, value := range summarySettings {
					log.Infof("SummarySetting[%d]: %+v", i, value)
				}
			})
	})
}

func TestTransmitNode_StartSummaryTaskViaTrustNode(t *testing.T) {

	testServeTransmitNode(func(transmitNode *TransmitNode, nodeInfo *TrustNodeInfo) {

		plain := `
{
    "wallets": [
        {
            "walletID": "WFXtudgu9Q5ktpcfDPC8gVEbHF1t1QWiVV",
            "accounts": [ 
                {
                    "accountID": "9XXLQfJAC55S2PugGqoyhi7FLZeGrfgJSMN7JePj75Sf",               
                    "threshold": "0.1",              
                    "minTransfer": "0",           
                    "retainedBalance": "0",           
                    "confirms": 0,                    
                    "onlyContracts": false,          
                    "contracts": {                            
                        "all": {                              
                            "threshold": "10",              
                            "minTransfer": "0",            
                            "retainedBalance": "0"           
                        }     
                    },
                    "feesSupportAccount": {          
                        "accountID": "6QvockspNbzxumH9soSL7PCerrsRsrfbqzC17Xs4Txhn",        
                        "lowBalanceWarning": "0.05",  
                        "lowBalanceStop": "0.001",     
                        "feesScale": "1"            
                    }
                },
                {
                    "accountID": "BBxgBEn7AoRhNqsS7vjD625B5SafFFdY1QMX7Zq8M9jn",               
                    "threshold": "0.1",              
                    "minTransfer": "0",           
                    "retainedBalance": "0",           
                    "confirms": 0,                    
                    "onlyContracts": false,          
                    "contracts": {                            
                        "all": {                              
                            "threshold": "10",              
                            "minTransfer": "0",            
                            "retainedBalance": "0"           
                        }     
                    },
                    "feesSupportAccount": {          
                        "accountID": "6QvockspNbzxumH9soSL7PCerrsRsrfbqzC17Xs4Txhn",        
                        "lowBalanceWarning": "0.05",  
                        "lowBalanceStop": "0.001",     
                        "feesScale": "1"            
                    }
                }
            ]
        }
    ]
}

`
		var summaryTask SummaryTask
		err := json.Unmarshal([]byte(plain), &summaryTask)
		if err != nil {
			log.Error("json.Unmarshal error:", err)
			return
		}

		transmitNode.StartSummaryTaskViaTrustNode(nodeInfo.NodeID, 120, &summaryTask, SummaryTaskOperateTypeReset,
			true, func(status uint64, msg string) {
				log.Infof("status: %d, msg: %+v", status, msg)
			})
	})
}

func TestTransmitNode_StopSummaryTaskViaTrustNode(t *testing.T) {
	testServeTransmitNode(func(transmitNode *TransmitNode, nodeInfo *TrustNodeInfo) {

		transmitNode.StopSummaryTaskViaTrustNode(nodeInfo.NodeID, true, func(status uint64, msg string) {
			log.Infof("msg:%+v", msg)
		})
	})
}

func TestTransmitNode_UpdateInfoViaTrustNode(t *testing.T) {
	testServeTransmitNode(func(transmitNode *TransmitNode, nodeInfo *TrustNodeInfo) {

		transmitNode.UpdateInfoViaTrustNode(nodeInfo.NodeID, true, func(status uint64, msg string) {
			log.Infof("msg:%+v", msg)
		})
	})
}

func TestTransmitNode_AppendSummaryTaskViaTrustNode(t *testing.T) {

	testServeTransmitNode(func(transmitNode *TransmitNode, nodeInfo *TrustNodeInfo) {

		setting := &SummarySetting{
			WalletID:        "WN84dVZXpgVixsvXnU8jkFWD1qWHp15LpA",
			AccountID:       "7ww2Gpfy8pN6HTngbMFBTEMAaVRGEpkmsiNkgAgqGQGf",
			SumAddress:      "0x4f544cbd23c42950a5fe7f967c3e6938955a1718",
			Threshold:       "1",
			MinTransfer:     "0.01",
			RetainedBalance: "0",
			Confirms:        1,
		}

		transmitNode.SetSummaryInfoViaTrustNode(nodeInfo.NodeID, setting, true, func(status uint64, msg string) {
			log.Infof("msg:%+v", msg)
		})

		plain := `

{
	"wallets": [{
		"walletID": "WN84dVZXpgVixsvXnU8jkFWD1qWHp15LpA",
		"password": "12345678",
		"accounts": [{
			"accountID": "7ww2Gpfy8pN6HTngbMFBTEMAaVRGEpkmsiNkgAgqGQGf"
		}]
	}]
}

`
		var summaryTask SummaryTask
		err := json.Unmarshal([]byte(plain), &summaryTask)
		if err != nil {
			log.Error("json.Unmarshal error:", err)
			return
		}

		transmitNode.AppendSummaryTaskViaTrustNode(nodeInfo.NodeID, &summaryTask,
			true, func(status uint64, msg string) {
				log.Infof("msg:%+v", msg)
			})
	})
}

func TestTransmitNode_RemoveSummaryTaskViaTrustNode(t *testing.T) {
	testServeTransmitNode(func(transmitNode *TransmitNode, nodeInfo *TrustNodeInfo) {

		transmitNode.RemoveSummaryTaskViaTrustNode(nodeInfo.NodeID,
			"WN84dVZXpgVixsvXnU8jkFWD1qWHp15LpA",
			"A3Mxhqm65kTgS2ybHLenNrZzZNtLGVobDFYdpc1ge4eK",
			true, func(status uint64, msg string) {
				log.Infof("msg:%+v", msg)
			})
	})
}

func TestTransmitNode_GetCurrentSummaryTaskViaTrustNode(t *testing.T) {
	testServeTransmitNode(func(transmitNode *TransmitNode, nodeInfo *TrustNodeInfo) {

		transmitNode.GetCurrentSummaryTaskViaTrustNode(nodeInfo.NodeID,
			true, func(status uint64, msg string, task *SummaryTask) {
				log.Infof("msg:%+v", msg)
				for _, w := range task.Wallets {
					log.Infof("task wallet:%+v", w.WalletID)
					for _, a := range w.Accounts {
						log.Infof("task account:%+v", a.AccountID)
					}
				}

			})
	})
}

func TestTransmitNode_GetSummaryTaskLogViaTrustNode(t *testing.T) {
	testServeTransmitNode(func(transmitNode *TransmitNode, nodeInfo *TrustNodeInfo) {

		transmitNode.GetSummaryTaskLogViaTrustNode(nodeInfo.NodeID, 0, 200,
			true, func(status uint64, msg string, taskLog []*SummaryTaskLog) {
				log.Infof("msg:%+v", msg)
				for _, r := range taskLog {
					log.Infof("taskLog: %+v", r)
				}

			})
	})
}

func TestTransmitNode_GetLocalWalletListViaTrustNode(t *testing.T) {
	testServeTransmitNode(func(transmitNode *TransmitNode, nodeInfo *TrustNodeInfo) {

		transmitNode.GetLocalWalletListViaTrustNode(nodeInfo.NodeID,
			true, func(status uint64, msg string, wallets []*Wallet) {
				log.Infof("msg:%+v", msg)
				for _, r := range wallets {
					log.Infof("wallet: %+v", r)
				}

			})
	})
}

func TestTransmitNode_GetTrustAddressListViaTrustNode(t *testing.T) {
	testServeTransmitNode(func(transmitNode *TransmitNode, nodeInfo *TrustNodeInfo) {

		transmitNode.GetTrustAddressListViaTrustNode(nodeInfo.NodeID,
			"",
			true, func(status uint64, msg string, trustAddressList []*TrustAddress, enableTrustAddress bool) {
				log.Infof("msg:%+v", msg)
				for _, r := range trustAddressList {
					log.Infof("turstaddress: %+v", r)
				}
				log.Infof("enableTrustAddress: %v", enableTrustAddress)
			})
	})
}

func TestTransmitNode_SignTransactionViaTrustNode(t *testing.T) {

	testServeTransmitNode(func(transmitNode *TransmitNode, nodeInfo *TrustNodeInfo) {

		walletID := "W3LxqTNAcXFqW7HGcTuERRLXKdNWu17Ccx"
		accountID := "65Y9FgipAS2M7ankrt4o3MR2Z1EEPNZKBqyQNsKt9wnj"
		address := "AR8LWKndC2ztfLoCobZhHEwkQCUZk1yKEsF"
		sid := uuid.New().String()
		password := ""
		coin := Coin{
			Symbol:     "VSYS",
			IsContract: false,
		}

		var (
			retRawTx       *RawTransaction
			retSignedRawTx *RawTransaction
			retTx          []*Transaction
			retFailed      []*FailedRawTransaction
			err            error
		)

		api, _ := transmitNode.APINode()
		api.CreateTrade(accountID, sid, coin, "0.01", address, "", "", true,
			func(status uint64, msg string, rawTx *RawTransaction) {
				if status != owtp.StatusSuccess {
					err = fmt.Errorf(msg)
					return
				}

				retRawTx = rawTx
			})

		if err != nil {
			t.Logf("CreateTrade unexpected error: %v\n", err)
			return
		}

		log.Infof("sid: %s", sid)
		transmitNode.SignTransactionViaTrustNode(nodeInfo.NodeID, walletID, retRawTx, password,
			true, func(status uint64, msg string, signedRawTx *RawTransaction) {
				log.Infof("status: %d, msg: %s", status, msg)
				log.Infof("signedRawTx: %+v", signedRawTx)
				if status != owtp.StatusSuccess {
					err = fmt.Errorf(msg)
					return
				}
				retSignedRawTx = signedRawTx
			})

		if err != nil {
			t.Logf("SignTransactionViaTrustNode unexpected error: %v\n", err)
			return
		}

		api.SubmitTrade([]*RawTransaction{retSignedRawTx}, true,
			func(status uint64, msg string, successTx []*Transaction, failedRawTxs []*FailedRawTransaction) {
				if status != owtp.StatusSuccess {
					err = fmt.Errorf(msg)
					return
				}

				retTx = successTx
				retFailed = failedRawTxs
			})

		if err != nil {
			t.Logf("SubmitTrade unexpected error: %v\n", err)
			return
		}

		log.Info("============== success ==============")

		for _, tx := range retTx {
			log.Infof("tx: %+v", tx)
		}

		log.Info("")

		log.Info("============== fail ==============")

		for _, tx := range retFailed {
			log.Infof("tx: %+v", tx.Reason)
		}
	})
}

func TestTransmitNode_TriggerABIViaTrustNode(t *testing.T) {

	testServeTransmitNode(func(transmitNode *TransmitNode, nodeInfo *TrustNodeInfo) {

		accountID := "7KgNQFx35ijMA43NgY89uaiwi9Tm4MH1PH68Kpnaqstu"
		contractAddress := "0x550cdb1020046b3115a4f8ccebddfb28b66beb27"
		abiParam := []string{"transfer", "0x19a4b5d6ea319a5d5ad1d4cc00a5e2e28cac5ec3", "3456"}
		password := "12345678"
		sid := uuid.New().String()
		log.Infof("sid: %s", sid)
		transmitNode.TriggerABIViaTrustNode(nodeInfo.NodeID, accountID, password, sid,
			contractAddress, "0", "", abiParam,
			true, func(status uint64, msg string, receipt *SmartContractReceipt) {
				log.Infof("status: %d, msg: %s", status, msg)
				log.Infof("receipt: %+v", receipt)
			})
	})
}
