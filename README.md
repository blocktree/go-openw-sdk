# go-openw-api-sdk
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

基于OWTP协议库，封装所有与openw-server钱包服务API交互方法。用于集成到go语言开发下的应用方系统。

## 概述

[TOC]

## Build development environment

The requirements to build OpenWallet are:

- Golang version 1.10 or later
- xgo (Go CGO cross compiler)
- Properly configured Go language environment
- Golang supported operating system

### 依赖blocktree本地库

github.com/blocktree/go-owcrypt
github.com/blocktree/go-owcdrivers
github.com/blocktree/openwallet
github.com/blocktree/go-openw-server

---

## go-openw-sdk
   
基于OWTP协议库，封装所有与openw-server钱包服务API交互方法。用于集成到go语言开发下的应用方系统。

### APINode

开发者通过APINode，与openw-server进行数据交互，实现OpenWallet钱包体系的管理功能。
APINode详细使用教程可查看[api_node测试用例](./openwsdk/api_node_test.go)。

```go

    //随机生成一个通信证书
    cert, _ := owtp.NewCertificate(owtp.RandomPrivateKey(), "")

    //配置APISDK参数
	config := &APINodeConfig{
    		AppID:  "1234abcd",
    		AppKey: "abcd1234",
    		Host:   "api.openwallet.cn",
    		Cert:               cert,
    		ConnectType:        owtp.HTTP,
    		EnableKeyAgreement: true,
	}

    //创建API实例
	api := NewAPINode(config)
	
	//App授权当前通信设备
	api.BindAppDevice()
	
	//查询币种列表，sync = true 同步线程，false 异步线程
	api.GetSymbolList(0, 1000, true, func(status uint64, msg string, symbols []*Symbol) {

		for _, s := range symbols {
			fmt.Printf("symbol: %+v\n", s)
		}

	})

```

### 订阅通知

```go

//订阅者需要实现OpenwNotificationObject接口
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

//运行订阅
func RunSubscribe() {

	var (
		endRunning = make(chan bool, 1)
	)
	
	//调用Subscribe进行订阅
	
	//订阅方法有几种
    //SubscribeToAccount    //订阅余额更新通信
    //SubscribeToTrade      //订阅新交易单通知
    //SubscribeToBlock      //订阅新区块链头通知
    
	err := api.Subscribe(
		[]string{
			SubscribeToTrade,
			//SubscribeToBlock,
		},
		":9322",    //本地服务开启端口，用于接收通知请求
		CallbackModeNewConnection, 
		CallbackNode{   //因为通知是异步的，需要订阅时需要提交一个回调服务节点
			NodeID:             api.node.NodeID(),
			Address:            "192.168.27.179:9322",      //这个回调服务对应于你开启的端口
			ConnectType:        owtp.Websocket,
			EnableKeyAgreement: false,
		})
	if err != nil {
		return
	}

    //订阅后，加入监听者队列，获得通知回调
	subscriber := &Subscriber{}
	api.AddObserver(subscriber)

	<-endRunning
}

```

### TransmitNode

TransmitNode是用于与授信的钱包托管节点进行双向交互。
钱包种子和密钥签名相关的操作会托管在授信节点上处理，满足于业务系统隔离于冷热钱包的安全方案。
需要配置go-openw-cli使用，通过`go-openw-cli -c=节点配置 trustserver`，在授信节点上启动后台服务。
详细可查看[api_transmit测试用例](./openwsdk/api_transmit_test.go)。

```go

    //启动监听器
    err := api.ServeTransmitNode("127.0.0.1:9088")
	if err != nil {
		log.Errorf("ServeTransmitNode error: %v\n", err)
		return
	}

    //转发服务处理器
	transmitNode, err := api.TransmitNode()
	if err != nil {
		log.Errorf("TransmitNode error: %v\n", err)
		return
	}
	
	//节点连接服务事件处理
	transmitNode.SetConnectHandler(func(transmitNode *TransmitNode, nodeInfo *TrustNodeInfo) {
        log.Infof("nodeInfo: %v", nodeInfo)
	})
	
	//节点断开服务事件处理
	transmitNode.SetDisconnectHandler(func(transmitNode *TransmitNode, nodeID string) {
        log.Infof("nodeID: %v", nodeID)
    })
	
	//创建钱包
	transmitNode.CreateWalletViaTrustNode(nodeInfo.NodeID, alias, password, true,
            func(status uint64, msg string, wallet *Wallet) {
                if wallet != nil {
                    log.Infof("wallet: %+v\n", wallet)
                }
            })

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
	
	//发起转账请求
    accountID := "A3Mxhqm65kTgS2ybHLenNrZzZNtLGVobDFYdpc1ge4eK"
    address := "mgCzMJDyJoqa6XE3RSdNGvD5Bi5VTWudRq"
    //可以不传密码，但需要cli的节点执行trustserver时结束钱包
    password := "12345678"  
    sid := uuid.New().String()
    transmitNode.SendTransactionViaTrustNode(nodeInfo.NodeID, accountID, password, sid,
        "", "0.03", address, "0.001", "",
        true, func(status uint64, msg string, successTx []*Transaction, failedRawTxs []*FailedRawTransaction) {

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
		
    //配置汇总
    setting := &SummarySetting{
                "WN84dVZXpgVixsvXnU8jkFWD1qWHp15LpA",
                "A3Mxhqm65kTgS2ybHLenNrZzZNtLGVobDFYdpc1ge4eK",
                "mgCzMJDyJoqa6XE3RSdNGvD5Bi5VTWudRq",
                "1",
                "0.01",
                "0",
                1,
            }
    
            transmitNode.SetSummaryInfoViaTrustNode(nodeInfo.NodeID, setting, true, func(status uint64, msg string) {
                log.Infof("msg:%+v", msg)
            })
    
    //启动后台定时汇总任务
    plain := `
    
    {
        "wallets": [{
            "walletID": "WN84dVZXpgVixsvXnU8jkFWD1qWHp15LpA",
            "password": "12345678",
            "accounts": [{
                "accountID": "A3Mxhqm65kTgS2ybHLenNrZzZNtLGVobDFYdpc1ge4eK"
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

    testServeTransmitNode(func(transmitNode *TransmitNode, nodeInfo *TrustNodeInfo) {

        transmitNode.StartSummaryTaskViaTrustNode(nodeInfo.NodeID, 10, &summaryTask,
            true, func(status uint64, msg string) {
                log.Infof("msg:%+v", msg)
            })
    })
        
    //关闭后台定时汇总任务
    testServeTransmitNode(func(transmitNode *TransmitNode, nodeInfo *TrustNodeInfo) {
    
            transmitNode.StopSummaryTaskViaTrustNode(nodeInfo.NodeID, true, func(status uint64, msg string) {
                log.Infof("msg:%+v", msg)
            })
        })
    
    //更新节点的区块链资料信息
    transmitNode.UpdateInfoViaTrustNode(nodeInfo.NodeID, true, func(status uint64, msg string) {
                log.Infof("msg:%+v", msg)
            })
    
    //追加汇总任务，钱包节点会合拼新任务到现有任务列表，可以不传密码，但需要cli的节点执行trustserver时结束钱包
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
    
    //移除当前执行的汇总任务，根据walletID和accountID进行移除
    transmitNode.RemoveSummaryTaskViaTrustNode(nodeInfo.NodeID,
        "WN84dVZXpgVixsvXnU8jkFWD1qWHp15LpA",
        "A3Mxhqm65kTgS2ybHLenNrZzZNtLGVobDFYdpc1ge4eK",
        true, func(status uint64, msg string) {
            log.Infof("msg:%+v", msg)
        })
    
    //获取当前执行中的汇总任务
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
    
    //获取汇总任务执行日志
    transmitNode.GetSummaryTaskLogViaTrustNode(nodeInfo.NodeID, 0, 200,
        true, func(status uint64, msg string, taskLog []*SummaryTaskLog) {
            log.Infof("msg:%+v", msg)
            for _, r := range taskLog {
                log.Infof("taskLog: %+v", r)
            }

        })
    
    //获取节点本地创建的钱包
    transmitNode.GetLocalWalletListViaTrustNode(nodeInfo.NodeID, 0, 200,
        true, func(status uint64, msg string, wallets []*Wallet) {
            log.Infof("msg:%+v", msg)
            for _, r := range wallets {
                log.Infof("wallet: %+v", r)
            }

        })
```
---