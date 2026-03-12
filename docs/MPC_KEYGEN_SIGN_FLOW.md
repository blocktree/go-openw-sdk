# MPC Keygen 与 Sign 流程说明

本文档描述当前 **Keygen** 与 **Sign** 模块的完整流程：服务端负责协调与转发，节点运行 TSS 协议；**TSS 协议消息由节点按目标加密后发送，服务端只做按 Subject 转发，不解密、不接触明文**，从而提升安全性。

---

## 1. 架构概览

```
                    服务端（协调 + 转发）
                           |
     ┌─────────────────────┼─────────────────────┐
     |                     |                     |
  节点 A                节点 B                节点 C
  (TSS Party)          (TSS Party)          (TSS Party)
     |                     |                     |
     |  加密(目标=B,C)       |  加密(目标=A,C)       |  加密(目标=A,B)
     |  POST /ws/mpcXxxMsg  |  POST /ws/mpcXxxMsg  |  POST /ws/mpcXxxMsg
     |  Body: {Subject, Data} 每条一个目标          |
     +--------------------->|<--------------------+
     |                     |                     |
     |  只转发：SendToSubject(Subject, "mpcXxxMsg", req)
     |  服务端不解密 Data，看不到 TSS 明文
```

- **服务端**：下发任务（mpcKeygenStart / mpcSignStart）、临时公钥交换协调、**按 Subject 转发** mpcKeygenMsg / mpcSignMsg、轮询结果；不存储 SaveData，不参与 TSS 计算。
- **节点**：持有临时 ECDH 密钥对；收到 Start 后存**其他节点临时公钥**；TSS 产生的每条消息**按目标节点分别加密**，每条请求只带一个目标（Subject + 密文 Data）；收到 Push 时用本节点私钥解密后投递给 TSS party。

---

## 2. 通用组件

### 2.1 临时 ECDH 公钥交换

- **触发**：服务端在创建 Keygen 或 Sign 任务时，向每个参与节点 Push 路由 `mpcTempPublicKey`，Body 为 `CliMPCTempPublicKeyReq{ TaskID, Module }`（Module 为 `"keygen"` 或 `"sign"`）。
- **节点**：收到后生成 ECDH 密钥对，将临时私钥以 `(subject, taskID, module, "tempPrivateKey")` 存入本地 cache；通过 **POST /ws/mpcTempPublicKey** 上报公钥（请求体含 TaskID、Module、PublicKey base64）。
- **服务端**：处理 POST 后把该节点公钥写入 cache（key 含 subject、taskID、module），并轮询直到所有节点公钥收齐，再继续下发 Start。

### 2.2 加密传输格式

- **服务端 → 节点**（Start 等）：Body 为 `CliMPCEncryptData{ TaskID, Data }`（或带 Subject），其中 `Data` 为 base64( ECDH 加密( JSON( payload ), AAD ) )。AAD 格式：`taskID|目标节点ID|路由名`，例如 `taskID|node1|mpcKeygenStart`。
- **节点 → 服务端 → 节点**（TSS 消息）：节点发送的 POST Body 为 `CliMPCEncryptData{ TaskID, Subject, Data }`。`Subject` 为**本条消息的目标节点 ID**；`Data` 为 base64( 用目标节点临时公钥加密的 CliMPCXxxMsgRes )。服务端**不解析 Data**，只根据 `Subject` 将整条 `req` 原样 Push 给对应连接（路由 mpcKeygenMsg 或 mpcSignMsg）。目标节点用本节点临时私钥解密 `Data` 得到 WireBytesBase64、FromIndex、IsBroadcast 等，再投递给 TSS。

---

## 3. Keygen 流程

### 3.1 服务端：CreateMPCKeyTask()

1. 取在线 subject，按 TSS 顺序得到 `nodeIDs`（3 或 5 个），确定 `threshold`（3→2，5→3）。
2. 生成 `taskID`、`expiredTime`；向每个节点 **Push mpcTempPublicKey**（Module=`"keygen"`）。
3. 轮询直到所有节点通过 POST `/ws/mpcTempPublicKey` 上报公钥，写入 `meta.PublicKey[nodeID]`；将 `MpcKeygenTaskMeta` 存入 cache（key `mpcMeta:taskID`）。
4. 构造 `CliMPCKeygenStartRes`（TaskID, NodeIDs, Threshold, ExpiredTime, PublicKeyPair），对每个节点用**该节点临时公钥**加密后 **Push mpcKeygenStart**，Body 为 `CliMPCEncryptData{ TaskID, Data: base64(密文) }`。
5. 轮询各节点上报的 **POST /ws/mpcKeygenResult**（Status=40、KeyID、Err）；校验 KeyID 一致、无 Err。
6. 计算 `walletID = ComputeKeyID([]byte(keyID))`，并将 KeyMeta（WalletID, KeyID, NodeIDs, Threshold, IndexByNodeID）写入 `mpc_keys_meta/{walletID}.json`；清理任务与临时公钥 cache；返回 KeyID。

### 3.2 节点：HandleMpcKeygenStart

1. 收到 **Push mpcKeygenStart**，Body 为 `CliMPCEncryptData`；用本节点在该 task 的**临时私钥**解密 `Data`，AAD 为 `taskID|myNodeID|mpcKeygenStart`，得到 `CliMPCKeygenStartRes`。
2. 将 **其他节点**的临时公钥写入本地 cache：对 `start.PublicKeyPair` 中除自己外的每条，`keyCache.Put( (v.Subject, taskID, "keygen:tempPublicKey"), v.PublicKey )`。
3. 注册 keygen 会话（router + recvCh），启动 delivery 协程；**异步**执行 RunKeygenNodeReal（生成 PreParams、创建 LocalParty、消费 outCh 并 Send、等待 endCh 得到 SaveData）。
4. Keygen 成功后：本地 `FileKeyStore.Save(keyID, myNodeID, saveData)`，再 **POST /ws/mpcKeygenResult** 上报 KeyID；失败则 POST 时带 Err。会话结束时删除本节点临时私钥 cache。

### 3.3 节点：Send（Keygen）— 按目标加密

- 对每条 TSS 消息，根据 `msg.IsBroadcast()` 和 `msg.GetTo()` 得到目标列表 `toNodeIDs`（广播 = 除自己外的所有节点，单播 = GetTo 的节点）。
- **对每个目标节点**：
  - 从 cache 取该目标公钥：`getTempPublicKey("keygen", targetNodeID, taskID)`。
  - 构造 `CliMPCKeygenMsgRes{ TaskID, WireBytesBase64, FromIndex, IsBroadcast }`，JSON 序列化后使用**目标节点临时公钥** ECDH 加密，AAD = `taskID|targetNodeID|mpcKeygenMsg`。
  - **POST /ws/mpcKeygenMsg** 一次，Body 为 `CliMPCEncryptData{ TaskID, Subject: targetNodeID, Data: base64(密文) }`。
- 服务端**从不看到** WireBytesBase64 明文。

### 3.4 服务端：handleMpcKeygenMsg

- 请求体解析为 `CliMPCEncryptData{ TaskID, Subject, Data }`。
- 只做：`SendToSubject(req.Subject, "mpcKeygenMsg", req)`，将整条 body 原样推给目标节点；不查 meta、不解密、不重加密。

### 3.5 节点：DeliverMpcKeygenMsg

- 收到 **Push mpcKeygenMsg**，Body 为 `CliMPCEncryptData`；用本节点临时私钥解密 `Data`（AAD = `taskID|myNodeID|mpcKeygenMsg`），得到 `CliMPCKeygenMsgRes`。
- 根据 TaskID、myNodeID 找到 keygen 会话，将 wireBytes、FromIndex、IsBroadcast 入队；delivery 协程串行调用 `party.Update(parsed)`。

---

## 4. Sign 流程

### 4.1 服务端：CreateMPCSignTask(keyID, msgHashHex)

> **强制全量节点在线且参与签名**：为了保证协议质量与安全性，当前实现要求 `keyMeta.NodeIDs` 中的**所有节点必须在线并参与本次签名**；暂不支持只用子集（例如 2-of-3 只用 2 个节点）完成签名。

1. 从 `mpc_keys_meta/{walletID}.json` 读取 KeyMeta（NodeIDs、Threshold）；其中 `walletID = ComputeKeyID([]byte(keyID))`；校验 msgHashHex。
2. 检查 `NodeIDs` 中的**每个节点**当前是否在线；若有任意一个离线则直接报错，拒绝本次签名请求。
3. 生成 `taskID`、`expiredTime`；向所有节点 **Push mpcTempPublicKey**（Module=`"sign"`），要求全量节点上报临时公钥。
4. 轮询收齐所有节点的临时公钥，写入 `signMeta.PublicKey`；将 signMeta 存入 cache（key `mpcSignMeta:taskID`）。
5. 构造 `CliMPCSignStartRes`（包含 **AllNodeIDs=全量节点**、`SignNodeIDs=AllNodeIDs`、Threshold、MsgHashHex、全量 `PublicKeyPair` 等），对每个节点用其临时公钥加密后 **Push mpcSignStart**。
6. 轮询所有节点 **POST /ws/mpcSignResult**（SignatureHex 或 Err）；校验各节点签名一致；清理 sign 相关 cache；返回 sigHex。

### 4.2 节点：HandleMpcSignStart

1. 收到 **Push mpcSignStart**，解密得到 `CliMPCSignStartRes`；将**其他节点**临时公钥存入 cache（key 含 `v.Subject, taskID, "sign:tempPublicKey"`）。
2. 注册 sign 会话，启动 delivery 协程；**异步**执行 RunSignNodeReal（加载 keyfile、创建 signing LocalParty、Send/Receive、等待签名结果）。
3. 成功则 **POST /ws/mpcSignResult** 上报 SignatureHex；失败则上报 Err。会话结束时删除本节点 sign 临时私钥。

### 4.3 节点：Send（Sign）— 按目标加密

- 与 Keygen 相同模式：对每个目标节点取公钥，构造 `CliMPCSignMsgRes`，用目标公钥加密，**POST /ws/mpcSignMsg** 一次，Body 为 `CliMPCEncryptData{ TaskID, Subject: targetNodeID, Data }`。
- 服务端 **handleMpcSignMsg** 仅解析 TaskID、Subject，然后 `SendToSubject(req.Subject, "mpcSignMsg", req)`，不接触明文。

### 4.4 节点：DeliverMpcSignMsg

- Push body 为 `CliMPCEncryptData`；用本节点 sign 临时私钥解密，得到 `CliMPCSignMsgRes`，入队后由 delivery 协程调用 `party.Update(parsed)`。

---

## 5. 路由与 DTO 汇总

| 方向 | 路由 / 类型 | 说明 |
|------|-------------|------|
| 服务端→节点 | Push `mpcTempPublicKey` | 通知节点提交临时公钥：TaskID, Module |
| 节点→服务端 | POST `/ws/mpcTempPublicKey` | 节点上报临时公钥（请求体含 PublicKey）；服务端写入 cache |
| 服务端→节点 | Push `mpcKeygenStart` | 下发 keygen 任务（Body 为 CliMPCEncryptData，密文内为 CliMPCKeygenStartRes） |
| 节点→服务端 | POST `/ws/mpcKeygenMsg` | 每条 Body 为 CliMPCEncryptData{ TaskID, Subject, Data }，Data=按目标加密的 CliMPCKeygenMsgRes |
| 服务端→节点 | Push `mpcKeygenMsg` | 服务端将收到的 req 原样转发给 req.Subject |
| 节点→服务端 | POST `/ws/mpcKeygenResult` | 上报 KeyID（及可选 Err） |
| 服务端→节点 | Push `mpcSignStart` | 下发 sign 任务（Body 为 CliMPCEncryptData，密文内为 CliMPCSignStartRes） |
| 节点→服务端 | POST `/ws/mpcSignMsg` | 每条 Body 为 CliMPCEncryptData{ TaskID, Subject, Data }，Data=按目标加密的 CliMPCSignMsgRes |
| 服务端→节点 | Push `mpcSignMsg` | 服务端将 req 原样转发给 req.Subject |
| 节点→服务端 | POST `/ws/mpcSignResult` | 上报 SignatureHex 或 Err |

---

## 6. 安全要点

- **节点按目标加密**：TSS 协议消息（WireBytes + FromIndex/IsBroadcast）仅在节点侧用**目标节点的临时公钥**加密，每条 POST 只带一个目标（Subject + Data）。
- **服务端只转发**：handleMpcKeygenMsg / handleMpcSignMsg 不解析 Data、不解密、不重加密，只根据 Subject 把整条 body 推送给对应节点。
- **无明文经服务端**：服务端无法看到 TSS 协议内容，无法还原或篡改各轮消息；仅能知道「谁向谁发了一条密文」用于路由。

---

## 7. 相关代码位置

| 内容 | 路径 |
|------|------|
| 服务端 Keygen/Sign 协调与转发 | `web/websocket2.go`（CreateMPCKeyTask, CreateMPCSignTask, handleMpcKeygenMsg, handleMpcSignMsg, handleMpcKeygenResult, handleMpcSignResult 等） |
| 路由注册 | `web/websocket.go`（/ws/mpcTempPublicKey, /ws/mpcKeygenMsg, /ws/mpcKeygenResult, /ws/mpcSignMsg, /ws/mpcSignResult） |
| 节点 Keygen | `node/mpc_keygen.go`（HandleMpcKeygenStart, Send 按目标加密, DeliverMpcKeygenMsg） |
| 节点 Sign | `node/mpc_sign.go`（HandleMpcSignStart, Send 按目标加密, DeliverMpcSignMsg） |
| 节点公钥交换与 Push 分发 | `node/main.go`（handleTempPublicKey, Push 回调 mpcKeygenStart/mpcKeygenMsg/mpcSignStart/mpcSignMsg） |
| DTO | `openwsdk/dto/cli.go`（CliMPCEncryptData, CliMPCKeygenStartRes, CliMPCKeygenMsgRes, CliMPCSignStartRes, CliMPCSignMsgRes 等） |
