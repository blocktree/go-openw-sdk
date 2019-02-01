# go-openw-api-sdk

基于OWTP协议库，封装所有与openw-server钱包服务API交互方法。用于集成到go语言开发下的应用方系统。

## 概述

[TOC]

## Build development environment

The requirements to build OpenWallet are:

- Golang version 1.10 or later
- govendor (a third party package management tool)
- xgo (Go CGO cross compiler)
- Properly configured Go language environment
- Golang supported operating system

## 依赖库管理工具govendor

### 安装govendor

```shell

go get -u -v github.com/kardianos/govendor

```

### 使用govendor

```shell

#进入到项目目录
$ cd $GOPATH/src/github.com/blocktree/OpenWallet

#初始化vendor目录
$ govendor init

#查看vendor目录
[root@CC54425A openwallet]# ls
commands  main.go  vendor

#将GOPATH中本工程使用到的依赖包自动移动到vendor目录中
#说明：如果本地GOPATH没有依赖包，先go get相应的依赖包
$ govendor add +external
或使用缩写： govendor add +e

#Go 1.6以上版本默认开启 GO15VENDOREXPERIMENT 环境变量，可忽略该步骤。
#通过设置环境变量 GO15VENDOREXPERIMENT=1 使用vendor文件夹构建文件。
#可以选择 export GO15VENDOREXPERIMENT=1 或 GO15VENDOREXPERIMENT=1 go build 执行编译
$ export GO15VENDOREXPERIMENT=1

# 如果$GOPATH下已更新本地库，可执行命令以下命令，同步更新vendor包下的库
# 例如本地的$GOPATH/github.com/blocktree/下的组织项目更新后，可执行下面命令同步更新vendor
$ govendor update +v

```

### 依赖blocktree本地库

github.com/blocktree/go-owcrypt
github.com/blocktree/go-owcdrivers
github.com/blocktree/OpenWallet
github.com/blocktree/go-openw-server

## 源码编译跨平台工具

### 安装xgo（支持跨平台编译C代码）

[官方github](https://github.com/karalabe/xgo)

xgo的使用依赖docker。并且把要跨平台编译的项目文件加入到File sharing。

```shell

$ go get github.com/karalabe/xgo
...
$ xgo -h
...

```

---

## go-openw-sdk
   
基于OWTP协议库，封装所有与openw-server钱包服务API交互方法。用于集成到go语言开发下的应用方系统。

### 使用教程

```go

    //随机生成一个通信证书
    cert, _ := owtp.NewCertificate(owtp.RandomPrivateKey(), "")

    //配置APISDK参数
	config := &APINodeConfig{
    		AppID:  "8df7420d3917afa0172ea9c85e07ab55",
    		AppKey: "faa14b5e2cf119cd6d38bda45b49eb02b333a1b1ff6f10703acb554011ebfb1e",
    		Host:   "120.78.83.180",
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

通过启动转发服务，控制授信节点的托管钱包，使用于隔离的冷热钱包方案

```go

    err := api.ServeTransmitNode("127.0.0.1:9088")
	if err != nil {
		log.Errorf("ServeTransmitNode error: %v\n", err)
		return
	}

	transmitNode, err := api.TransmitNode()
	if err != nil {
		log.Errorf("TransmitNode error: %v\n", err)
		return
	}
	
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
```
---