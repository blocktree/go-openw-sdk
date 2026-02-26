# OpenW SDK 完整指南

## 概述

OpenW SDK 采用双层架构设计，为区块链应用提供安全的密钥管理和交易签名服务：

- **CLI程序**：本地钱包管理，负责私钥存储和交易签名
- **云端OPS系统**：业务处理服务，负责交易构建和广播

## 快速开始

### 编译运行
```bash
# 编译CLI程序
go build -o cli-app main.go

# 启动交互式控制台（使用默认配置）
./cli-app

# 或指定自定义配置文件
./cli-app -c myconfig.yaml
```

### 主要功能
1. **Create Wallet** - 创建新的加密钱包
2. **Unlock Wallet** - 解锁现有钱包进行操作
3. **Generate ECDSA** - 生成ECDSA密钥对
4. **Start Service** - 启动HTTP签名服务
5. **Exit** - 退出程序

### 命令行参数
- `-c`：指定配置文件路径（默认：`config.yaml`）
- `-h`：显示帮助信息

**示例：**
```bash
# 使用默认配置
./cli-app

# 使用自定义配置
./cli-app -c production.yaml

# 查看帮助
./cli-app -h
```

### 重要提醒
交易签名需要先启动HTTP服务，服务将在后台运行并监听本地端口。

## 核心功能

### 1. 钱包管理

#### 创建钱包
```bash
# 在CLI交互界面中选择 "Create Wallet"
# 1. 输入别名（字母数字组合）
# 2. 输入密码（最少8字符，两次确认）
# 3. 系统自动创建加密钱包文件
```

#### 解锁钱包

**交互式操作：**
1. 在主菜单选择 **"Unlock Wallet"** (快捷键: 2)

2. 系统显示钱包列表，选择要解锁的钱包：
   ```
   testwallet1 (VzYK21Vem6WBXHXZmSRYGN4iaE6n2naF6z)
   File: testwallet1-VzYK21Vem6WBXHXZmSRYGN4iaE6n2naF6z.key
   ```

3. 输入密码（无回显输入）：
   ```
   Enter password (input is HIDDEN):
   ```

4. 解锁成功后显示：
   ```
   ✅ Wallet unlocked successfully!
   Press Enter to return to main menu...
   ```

**安全特性：**
- 密码在锁定内存中处理，解锁后立即销毁
- 种子密钥加密存放内存，用于后续签名操作
- 程序退出时自动清理所有敏感数据

### 2. 账户管理

#### 创建账户
```go
// CLI派生新账户
cliRequestData := dto.CliCreateAccountReq{
    WalletID:  "VzYK21Vem6WBXHXZmSRYGN4iaE6n2naF6z",
    LastIndex: -1,
    Curve:     3972005888,
}
cliResponseData := dto.CliCreateAccountRes{}
cliHttpSDK.PostByAuth("/api/CreateAccount", &cliRequestData, &cliResponseData, true)

// 云端注册账户信息
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

### 3. 交易处理

#### 前置条件：启动签名服务
```bash
# 编译并启动CLI程序
go build -o cli-app main.go
./cli-app

# 在交互界面中选择 "Start Service"
# 服务将在后台运行，监听 http://127.0.0.1:9422
```

#### 标准交易流程

##### 1. 创建交易单
```go
// 生成交易单ID并构建交易
sid := utils.GetUUID(true)
requestData := dto.CreateTradeReq{
    Sid:       sid,
    AccountID: "8K4oVwL3dLQmLzrsj2zXzbeapCsETNsRVFtykmAKBvp6",
    Coin: dto.CoinInfo{Symbol: "BETH"},
    To: map[string]string{"0x4f8abf232ffd006a49a9426ed9a2ab377ce4bdce": "0.1"},
}
responseData := dto.CreateTradeRes{}
opsHttpSDK.PostByAuth("/api/CreateTrade", &requestData, &responseData, true)
```

##### 2. 验证交易单
```bash
# 验证云端签名，检查关键字段：symbol、sid、accountID、to、contractID
```

##### 3. CLI签名
```go
// CLI进行交易签名
cliRequestData := dto.CliSignTransactionReq{
    Type: 0, Data: txData.Data, TradeSign: txData.TradeSign,
}
cliResponseData := dto.CliSignTransactionRes{}
cliHttpSDK.PostByAuth("/api/SignTransaction", &cliRequestData, &cliResponseData, true)
```

##### 4. 广播交易
```go
// 更新签名并广播
txData.SignerList = cliResponseData.SignerList
requestSubmitData := dto.SubmitRawTransactionReq{TxData: txData}
responseSubmitData := dto.SubmitRawTransactionRes{}
opsHttpSDK.PostByAuth("/api/SubmitTrade", &requestSubmitData, &responseSubmitData, true)
```

#### 特殊交易类型

##### 合约交易
```go
// 设置 IsContract: true 和 ContractID
requestData := dto.CreateTradeReq{
    Sid: utils.GetUUID(true),
    AccountID: "2TBCLPTaRQpbG6VwWuTNdPPgQtC8o3tzVnqUgJvTXmGs",
    Coin: dto.CoinInfo{Symbol: "BETH", IsContract: true, ContractID: "..."},
    To: map[string]string{"0xdAb9c307B8B23A8fD8559f75C71F0694Da30D9F6": "0.2"},
}
```

##### 汇总交易
```go
// 批量转账交易
requestData := dto.CreateSummaryTxReq{
    Sid: utils.GetUUID(true), AccountID: "...",
    MinTransfer: "0", RetainedBalance: "0", Address: "...",
    Coin: dto.CoinInfo{Symbol: "BETH"},
}
opsHttpSDK.PostByAuth("/api/CreateSummaryTx", &requestData, &responseData, true)
```

## 安全保障

### 密码生命周期
```
解锁密码流程：
1. 用户输入密码 → 2. 锁定到安全内存 → 3. 用于解锁钱包种子 → 4. 立即销毁密码内存块
   ↑                        ↑                        ↑                        ↑
输入无回显               防止内存泄露             单次解锁使用             安全即时清理

钱包解锁后：
5. 种子密钥加密存放内存 → 6. 签名操作使用 → 7. 程序退出时清理种子内存
   ↑                            ↑                        ↑
加密保护内存                 本地签名                自动清理
```

### 私钥安全
- ✅ 私钥以加密钱包文件形式存储在本地磁盘
- ✅ 所有签名操作在本地内存中完成
- ✅ 从不通过网络传输私钥、种子或密码

### 密钥轮换机制
SDK请求配置支持自动密钥轮换，确保通信安全：

**轮换策略：**
- ✅ **最低间隔**：15秒一次密钥轮换
- ✅ **向前兼容**：新密钥生成后，老密钥在过渡期仍有效
- ✅ **渐进替换**：避免单点故障导致全部密钥泄露
- ✅ **自动管理**：SDK内部自动处理密钥生命周期

**安全优势：**
- 🔒 **时间限制泄露**：即使某个密钥泄露，也仅在15秒窗口内有效
- 🔄 **持续更新**：密钥定期轮换，降低长期泄露风险
- 🛡️ **容错设计**：支持密钥过渡期，确保服务连续性
- ⚡ **性能优化**：轮换间隔可配置，平衡安全与性能

**配置示例：**
```yaml
# config.yaml
jwt:
  cli_main:
    token_key_rotation: 15s  # 密钥轮换间隔
    token_overlap: 30s       # 过渡期时长
```
- ✅ 业务系统必须验证交易单签名
- ✅ 校验关键字段：Symbol、Sid、AccountID、To、ContractID
- ✅ 使用白名单/黑名单控制交易目标

### 并发控制
- ✅ 敏感操作使用原子锁防止并发
- ✅ 密码内存操作使用安全清理机制

### 网络安全
- ✅ 敏感API仅允许本地访问
- ✅ 支持远程白名单配置
- ✅ 所有通信使用加密传输

## API 参考

| 操作类型 | CLI端点 | OPS端点 | 说明 |
|---------|---------|---------|------|
| 钱包管理 | `/api/FindWalletList` | `/api/CreateWallet` | 钱包注册 |
| 账户管理 | `/api/CreateAccount` | `/api/CreateAccount` | 账户创建 |
| 交易处理 | `/api/SignTransaction` | `/api/CreateTrade` | 交易构建 |
| 交易广播 | - | `/api/SubmitTrade` | 交易广播 |
| 合约交易 | - | `/api/CreateTrade` | 合约转账 |
| 汇总交易 | - | `/api/CreateSummaryTx` | 批量转账 |

## 部署架构

### 推荐部署方式
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   业务系统      │────│   云端OPS       │────│   区块链网络    │
│   (Business)    │    │   (Cloud OPS)   │    │   (Blockchain)  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                        │
         └────────────────────────┘
                  │
         ┌─────────────────┐
         │   CLI程序       │
         │   (Local CLI)   │
         └─────────────────┘
```

### 交互流程
1. **业务系统** → **云端OPS**：请求交易构建
2. **云端OPS** → **CLI程序**：请求交易签名
3. **CLI程序** → **云端OPS**：返回签名结果
4. **云端OPS** → **区块链**：广播交易

这种架构确保了私钥的安全性，同时提供了便捷的业务集成能力。