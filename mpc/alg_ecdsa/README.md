# mpc — TSS 多方门限签名

基于 `github.com/bnb-chain/tss-lib` 的 (t,n) 门限 ECDSA 封装，用于替代原有 SSS 分片方案：**keygen 与 sign 全程不重建完整私钥**。

## 结构概览

- **params.go** — 从节点 ID 列表生成排序后的 `PartyIDs`、构建 `tss.Parameters`、从公钥算 `KeyID`。
- **transport.go** — `MessageRouter` 接口：在各方之间转发 TSS 协议消息；`InProcessRouter` 为单机内存实现（测试/demo）。
- **keygen.go** — `RunKeygen`：跑完整 TSS keygen，得到每方一份 `LocalPartySaveData` 和统一 `KeyID`。
- **sign.go** — `RunSign`：对给定消息哈希做 TSS 签名，得到 R、S 及 64 字节签名；`MessageHashFromTxHash` 将 hex 交易哈希转为 `*big.Int`。
- **storage.go** — `KeyStore` 接口与 `KeygenResultByNode`，便于按节点持久化或下发 keygen 结果。
- **init_example.go** — 密钥初始化封装：`InitKey`、`InitKeyRequest/Response`、`ThresholdFromWalletMode`、`ExampleInitKey`，可直接对接 walletMode 3/5。
- **keystore_file.go** — 基于文件的 `KeyStore` 实现：`Save(keyID, nodeID, data)` 写入 `BaseDir/keyID/nodeID.json`，`Load` 读回并恢复曲线；`LoadAllForKey(keyID, nodeIDs)` 一次加载某 key 下所有节点 SaveData，用于重启后签名。
- **account_derive.go** — 无 seed 时的账户/地址派生：`DeriveChildPubFromPath`(根公钥, chainCode, path) 得到 keyDerivationDelta 与子公钥；`ChainCodeFromKeyID`、`PubKeyToHex`、`PathFromAccountIndex`；签名派生账户用 `RunSignWithKDD`。

## 使用方式

### 0. 密钥初始化（推荐入口）

与现有 walletMode（3=2-of-3，5=3-of-5）一致时，可直接用 `InitKey`：

```go
req := mpc.InitKeyRequest{
    NodeIDs:    []string{"node-1", "node-2", "node-3"}, // 与 sharding 的 subject 一致
    WalletMode: 3, // 2-of-3
}
config := mpc.KeygenConfig{PreParamsTimeout: time.Minute}
resp, err := mpc.InitKey(req, config, nil)
if err != nil { /* ... */ }

// resp.KeyID 写库；resp.KeyByNode[nodeID] 为该节点的 SaveData，需加密下发给节点或存 KeyStore
for nodeID, data := range resp.KeyByNode {
    _ = keyStore.Save(resp.KeyID, nodeID, data) // 或通过 shardingPost 通道加密下发
}
```

门限由 `WalletMode` 决定：`mpc.ThresholdFromWalletMode(3)==2`，`ThresholdFromWalletMode(5)==3`。完整示例见 `init_example.go` 中的 `ExampleInitKey`。

**重要**：keygen 只需做一次。之后只要各节点持有一份自己的 SaveData，每次签名时用同一 KeyID 对应的 SaveData + 消息哈希调用 `RunSign` 即可得到最终签名，无需再次初始化；除非换参与节点、新建另一把钥匙或 SaveData 丢失且不足门限。

**保存与重启加载**：可用 `FileKeyStore`（`NewFileKeyStore(dir)`）按节点 `Save(keyID, nodeID, data)`；重启后用 `Load(keyID, nodeID)` 或 `LoadAllForKey(keyID, nodeIDs)` 读回，得到与 keygen 时同序的 `[]keygen.LocalPartySaveData`，再传给 `RunSign`。

**SaveData 保密**：SaveData 内含 Paillier 私钥、秘密份额等，等同于私钥级别，必须保密。`FileKeyStore` 当前为明文 JSON + 0600 权限，仅适合测试或可信环境。生产建议：落盘前对内容加密（如 AES-GCM + 密钥由密码/KMS 派生）、存于加密卷或严格权限目录；若下发到节点，必须走加密通道（如现有 ECDH），节点侧再加密存或存安全模块。

**无 seed 时的 AccountID/地址**：MPC 没有单点 seed，用「根公钥(ECDSAPub) + chainCode + 路径」派生。`DeriveChildPubFromPath` 得到子公钥与 keyDerivationDelta；子公钥用于 GenAccountID/地址；签名该账户时用 `RunSignWithKDD`(keys, msgHash, delta, childPub, router)。详见 `mpc/docs/ACCOUNT_AND_ADDRESS.md`。

### 1. 单机测试（内存路由）

```go
nodeIDs := []string{"node1", "node2", "node3"}
threshold := 2
router := &mpc.InProcessRouter{ErrCh: make(chan *tss.Error, 16)}

result, err := mpc.RunKeygen(nodeIDs, threshold, router, mpc.KeygenConfig{
    PreParamsTimeout: time.Minute,
}, nil)
// result.SaveData[i] 对应 nodeIDs[i]，result.KeyID 为公钥标识

msgHash, _ := mpc.MessageHashFromTxHash(hexTxHash)
sig, err := mpc.RunSign(nodeIDs, result.SaveData, msgHash, router)
// sig.Signature 为 64 字节 R||S，可用于链上
```

### 2. 服务端 + 多节点（WebSocket 路由）

- 服务端维护「keygen/sign 会话」与当前轮次，收到某节点的 TSS 消息后，根据 `msg.GetTo()` / `msg.IsBroadcast()` 转发给其他节点。
- 实现 `MessageRouter`：
  - `Send(fromIndex, msg)`：将 `msg.WireBytes()`、`fromIndex`、`msg.IsBroadcast()` 及目标下标发给对应节点（或广播）。
  - `Receive(toIndex, wireBytes, fromIndex, isBroadcast)`：节点收到后，用 `tss.ParseWireMessage(wireBytes, sortedIDs[fromIndex], isBroadcast)` 得到 `ParsedMessage`，再调用该方 `party.Update(parsed)`。
- Keygen 结束后，用 `KeygenResultByNode(nodeIDs, result)` 得到「节点 ID -> SaveData」，再经 ECDH 等加密下发给各节点存储（可复用现有 shardingPost 通道）。
- 签名时由服务端发起 sign 会话，收集参与签名的节点及其 `LocalPartySaveData`，跑 `RunSign(signingNodeIDs, keys, msgHash, wsRouter)`；若各 party 分布在节点上，则需在节点侧运行 `signing.LocalParty` 并通过同一 `MessageRouter` 收发消息。

### 3. 与现有流程的对应

| 原 SSS 流程           | MPC 对应                          |
|----------------------|-----------------------------------|
| CreateShardingTask   | RunKeygen + 下发 SaveData 到各节点 |
| shardingPre/Post     | MessageRouter.Send/Receive（WS 转发） |
| SignSharedTransaction | RunSign(signingNodeIDs, keys, msgHash, router) |
| SignRawTransactionExtractLocker(seed) | 不再重建 seed；用 TSS 签名结果填 txSigner |

## 曲线与门限

- 当前使用 **secp256k1**（`tss.S256()`），与 BTC/ETH 等一致。
- 门限：2-of-3、3-of-5 等由 `RunKeygen`/`RunSign` 的 `threshold` 与 `nodeIDs` 长度决定。

## 依赖

- `github.com/bnb-chain/tss-lib`（已在 go.mod 中；btcd 通过 replace 固定到 v0.22.1 以兼容 tss-lib 的 btcec 导入）。
