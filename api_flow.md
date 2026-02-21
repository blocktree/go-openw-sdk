# OpenW SDK API 流程文档

本文档基于 `api_flow_test.go` 测试用例，描述了完整的钱包、账户和交易操作流程。

## 架构说明

系统采用双层架构：
- **CLI程序**：本地钱包管理服务，负责私钥存储和交易签名
- **云端OPS系统**：业务处理服务，负责交易构建和广播

## 1. 创建钱包流程

### 流程概述
1. 从CLI程序读取钱包文件列表
2. 上传WalletID信息到云端OPS系统

### 详细步骤

```go
// 步骤1：从CLI获取钱包列表
cliRequestData := dto.CliFindWalletListReq{}
cliResponseData := dto.CliFindWalletListRes{}
cliHttpSDK.PostByAuth("/api/FindWalletList", &cliRequestData, &cliResponseData, true)

// 步骤2：遍历钱包列表，上传到云端
for _, v := range cliResponseData.Result {
    requestData := dto.CreateWalletReq{
        WalletID: v.WalletID,
        Alias:    v.Alias,
        RootPath: v.RootPath,
    }
    responseData := dto.CreateWalletRes{}
    opsHttpSDK.PostByAuth("/api/CreateWallet", &requestData, &responseData, true)
}
```

## 2. 创建账户流程

### 流程概述
1. 指定WalletID发送给CLI程序进行派生AccountID
2. 上传AccountID信息到云端OPS系统

### 详细步骤

```go
// 步骤1：CLI派生账户
cliRequestData := dto.CliCreateAccountReq{
    WalletID:  "VzYK21Vem6WBXHXZmSRYGN4iaE6n2naF6z",
    LastIndex: -1,
    Curve:     3972005888,
}
cliResponseData := dto.CliCreateAccountRes{}
cliHttpSDK.PostByAuth("/api/CreateAccount", &cliRequestData, &cliResponseData, true)

// 步骤2：上传账户信息到云端
requestData := dto.CreateAccountReq{
    WalletID:     cliResponseData.WalletID,
    AccountID:    cliResponseData.AccountID,
    Alias:        "test",
    Symbol:       "BETH",
    PublicKey:    cliResponseData.PublicKey,
    HdPath:       cliResponseData.HdPath,
    ReqSigs:      cliResponseData.ReqSigs,
    AccountIndex: cliResponseData.AccountIndex,
}
responseData := dto.CreateAccountRes{}
opsHttpSDK.PostByAuth("/api/CreateAccount", &requestData, &responseData, true)
```

## 3. 创建交易流程

### 流程概述
1. 业务系统创建交易单保存关键字段（Symbol、Sid、AccountID、To）
2. 发起交易单构建请求到云端OPS系统
3. 业务系统验证交易单数据签名
4. 反序列化交易单对象进行关键字段校验
5. 交易单转发到CLI程序进行签名
6. CLI签名成功后，云端系统广播交易

### 详细步骤

```go
// 步骤1：准备交易参数
symbol := "BETH"
accountID := "8K4oVwL3dLQmLzrsj2zXzbeapCsETNsRVFtykmAKBvp6"
toAddress := "0x4f8abf232ffd006a49a9426ed9a2ab377ce4bdce"
toAmount := "0.1"

// 步骤2：创建交易单
requestData := dto.CreateTradeReq{
    Sid:       utils.GetUUID(true),
    AccountID: accountID,
    Coin: dto.CoinInfo{
        Symbol: symbol,
    },
    To: map[string]string{
        toAddress: toAmount,
    },
}
responseData := dto.CreateTradeRes{}
opsHttpSDK.PostByAuth("/api/CreateTrade", &requestData, &responseData, true)

// 步骤3：验证交易单签名
CheckTxDataSign(opsConfig.AppKey, responseData.TxData)

// 步骤4：反序列化并校验关键字段
txData := responseData.TxData[0]
tx := &openwallet.RawTransaction{}
utils.JsonUnmarshal(utils.Str2Bytes(txData.Data), tx)

// 校验Symbol、AccountID、To等关键字段
if tx.Coin.Symbol != symbol {
    // symbol校验失败
}
if tx.Account.AccountID != accountID {
    // accountID校验失败
}

// 步骤5：CLI签名
cliRequestData := dto.CliSignTransactionReq{
    Data:      txData.Data,
    TradeSign: txData.TradeSign,
}
cliResponseData := dto.CliSignTransactionRes{}
cliHttpSDK.PostByAuth("/api/SignTransaction", &cliRequestData, &cliResponseData, true)

// 步骤6：提交交易广播
txData.SignerList = cliResponseData.SignerList
requestSubmitData := dto.SubmitRawTransactionReq{
    TxData: txData,
}
responseSubmitData := dto.SubmitRawTransactionRes{}
opsHttpSDK.PostByAuth("/api/SubmitTrade", &requestSubmitData, &responseSubmitData, true)
```

## 4. 创建合约交易流程

### 流程概述
创建合约代币交易，流程与普通交易类似，但Coin信息中需指定合约相关参数。

### 关键差异

```go
requestData := dto.CreateTradeReq{
    Sid:       utils.GetUUID(true),
    AccountID: "2TBCLPTaRQpbG6VwWuTNdPPgQtC8o3tzVnqUgJvTXmGs",
    Coin: dto.CoinInfo{
        Symbol:     "BETH",
        IsContract: true,           // 标记为合约交易
        ContractID: "ZA+oTwXimYwVFJ5Tk7ACU6tD+6ycw7u2UsdHLVof8kg=", // 合约ID
    },
    To: map[string]string{
        "0xdAb9c307B8B23A8fD8559f75C71F0694Da30D9F6": "0.2",
    },
}
```

## 5. 创建汇总交易流程

### 流程概述
创建汇总交易，将多个小额转账合并为一个交易。

```go
requestData := dto.CreateSummaryTxReq{
    Sid:             utils.GetUUID(true),
    AccountID:       "2TBCLPTaRQpbG6VwWuTNdPPgQtC8o3tzVnqUgJvTXmGs",
    MinTransfer:     "0",           // 最小转账金额
    RetainedBalance: "0",           // 保留余额
    Address:         "0xa6f4ddc5f8b6b6a07e1e250531f7600daa227138", // 汇总地址
    Coin:            dto.CoinInfo{Symbol: "BETH"},
}
responseData := dto.CreateTradeRes{}
opsHttpSDK.PostByAuth("/api/CreateSummaryTx", &requestData, &responseData, true)
```

## 安全注意事项

1. **私钥安全**：私钥仅存储在CLI程序本地，不会上传到云端
2. **签名验证**：所有交易单都需要业务系统验证签名和关键字段
3. **双重签名**：交易需要CLI程序签名后才能广播
4. **参数校验**：业务系统必须校验Symbol、AccountID、To等关键参数

## API端点总结

| 操作 | CLI端点 | OPS端点 |
|------|---------|---------|
| 获取钱包列表 | `/api/FindWalletList` | - |
| 创建钱包 | - | `/api/CreateWallet` |
| 创建账户 | `/api/CreateAccount` | `/api/CreateAccount` |
| 创建交易 | - | `/api/CreateTrade` |
| 签名交易 | `/api/SignTransaction` | - |
| 提交交易 | - | `/api/SubmitTrade` |
| 创建汇总交易 | - | `/api/CreateSummaryTx` |