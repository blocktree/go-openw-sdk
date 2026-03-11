# MPC 测试用例详细教程

本文说明 `mpc` 包中测试用例的含义、如何运行，以及每一步在做什么。

---

## 一、测试文件与用例概览

| 测试函数 | 作用 | 耗时 | 说明 |
|----------|------|------|------|
| `TestPartyIDs` | 校验 PartyID 生成与排序 | 毫秒级 | 不跑 TSS 协议，只测工具函数 |
| `TestKeygenAndSignInProcess` | 完整 2-of-3 keygen + 一次签名 | 约 1～3 分钟 | 单机内存内跑完 keygen 和 sign |

---

## 二、环境与运行方式

### 2.1 前置条件

- Go 1.26+（与项目 go.mod 一致）
- 已拉取依赖：`go mod tidy` 或 `go get github.com/bnb-chain/tss-lib@v1.5.0`

### 2.2 只跑「快」测试（推荐先跑）

不跑耗时的 keygen，只测 PartyID：

```bash
cd /path/to/go-openw-sdk
go test -v ./mpc/ -run TestPartyIDs
```

预期：几秒内结束，输出类似：

```
=== RUN   TestPartyIDs
--- PASS: TestPartyIDs (0.00s)
PASS
```

### 2.3 跑完整 keygen + sign 测试

会生成 PreParams 并跑 2-of-3 keygen 和一次签名，**耗时约 1～3 分钟**：

```bash
go test -v -timeout=5m ./mpc/ -run TestKeygenAndSignInProcess
```

注意：

- **不要** 加 `-short`，否则该测试会被跳过。
- 第一次运行会现场生成 Paillier 与安全素数（PreParams），最慢；后续若改用预生成 PreParams 会快很多。

### 2.4 跑全部 mpc 测试

```bash
go test -v -timeout=5m ./mpc/
```

若在 CI 里希望跳过慢测试，可加 `-short`：

```bash
go test -v -short ./mpc/
```

此时只会执行 `TestPartyIDs`，`TestKeygenAndSignInProcess` 会被跳过。

---

## 三、TestPartyIDs 在测什么？

### 3.1 代码在做什么

```go
ids := mpc.PartyIDs([]string{"node1", "node2", "node3"})
```

- 输入：节点 ID 字符串列表（例如 `["node1","node2","node3"]`）。
- 内部：对 ID 列表**排序**，再对每个 ID 用 SHA256 生成一个 `*big.Int` 作为 TSS 的 key，并调用 `tss.NewPartyID(id, id, key)`，最后按 key 排序得到 `tss.SortedPartyIDs`。
- 输出：与 TSS 协议要求一致的、**排序稳定**的 PartyID 列表，供 keygen/sign 使用。

### 3.2 断言含义

- `len(ids) == 3`：参与方数量正确。
- `ids[0].Index != ids[1].Index`：排序后每方有唯一下标（0、1、2），协议用 Index 做路由。

### 3.3 为何重要

Keygen/Sign 里「谁是谁」必须全网一致，否则协议会错。`PartyIDs(nodeIDs)` 保证：只要 `nodeIDs` 相同（顺序可任意，内部会排序），得到的 PartyID 顺序一致，便于服务端与各节点对齐。

---

## 四、TestKeygenAndSignInProcess 分步说明

该测试分两阶段：**Keygen（门限密钥生成）** 和 **Sign（对一条消息做门限签名）**。全程在**单进程**内用内存路由（`InProcessRouter`）模拟三节点。

### 4.1 准备：节点列表与路由

```go
nodeIDs := []string{"n1", "n2", "n3"}
threshold := 2
errCh := make(chan *tss.Error, 8)
router := &mpc.InProcessRouter{ErrCh: errCh}
```

- **nodeIDs**：3 个参与方，对应 keygen 后的 3 份 SaveData。
- **threshold = 2**：2-of-3 门限，即任意 2 方即可签名（本测试用 3 方都参与 sign）。
- **errCh**：TSS 协议报错会写到这里。
- **InProcessRouter**：内存版 `MessageRouter`，把某一方发出的消息按 `GetTo()`/`IsBroadcast()` 转给其他方，用于单机测试。

### 4.2 第一阶段：RunKeygen（密钥生成）

```go
result, err := mpc.RunKeygen(nodeIDs, threshold, router, mpc.KeygenConfig{
    PreParamsTimeout: 1 * time.Minute,
}, nil)
```

含义简述：

1. **RunKeygen** 内部会：
   - 用 `PartyIDs(nodeIDs)` 得到排序后的 3 个 PartyID；
   - 为每方创建 `keygen.LocalParty`（2-of-3 参数）；
   - 若未传入 `preParams`，则在 **PreParamsTimeout** 内为每方生成 PreParams（Paillier + 安全素数），**这里最耗时**；
   - 启动 3 个 party，通过 `router` 交换多轮消息，直到 keygen 完成。
2. **返回值**：
   - `result.SaveData`：长度为 n，**`SaveData[i]` 对应排序后第 i 方（PartyIDs 下标 i）的私密数据**，顺序与 keygen 内部 party 下标一致，不能给其他方。
   - `result.KeyID`：由公钥算出的标识（例如 SHA256(ECDSAPub) 的 hex），与业务里的「钱包/key ID」对应。

测试中的断言：

- `len(result.SaveData) == 3`：三方都成功完成 keygen。
- `result.KeyID != ""`：公钥已正确生成并算出 KeyID。

### 4.3 构造待签名消息哈希

```go
msgHash := new(big.Int).SetBytes([]byte("hello world 32 bytes hash!!!!!!"))
if msgHash.BitLen() > 256 {
    msgHash = new(big.Int).SetBytes(msgHash.Bytes()[:32])
}
```

- TSS 签名的输入是「消息哈希」的 `*big.Int`，且需 **&lt; 曲线阶 N**。
- 这里用固定 32 字节字符串模拟一条哈希；真实场景应使用链上/业务规定的哈希（如 `Keccak256(tx)` 或 `SHA256(tx)`），并用 `mpc.MessageHashFromTxHash(hexTxHash)` 转成 `*big.Int`。

### 4.4 第二阶段：RunSign（门限签名）

```go
sigResult, err := mpc.RunSign(nodeIDs, result.SaveData, msgHash, router)
```

含义简述：

1. **RunSign** 会：
   - 再次用 `PartyIDs(nodeIDs)` 得到与 keygen 一致的顺序；
   - 对每方用 `keygen.BuildLocalSaveDataSubset(keys[i], sortedIDs)` 得到本次签名参与方的 save 子集；
   - 创建 3 个 `signing.LocalParty`，用同一 `msgHash` 和同一 router 跑 TSS 签名协议；
   - 收集签名结果，聚合成最终 (R, S)。
2. **返回值**：
   - `sigResult.Signature`：64 字节，R||S（secp256k1），可直接用于链上或 hex 输出；
   - `sigResult.R`、`sigResult.S`：`*big.Int`，便于校验或兼容不同格式；
   - `sigResult.Recovery`：以太坊风格 recovery id（若需要）。

测试中的断言：

- `len(sigResult.Signature) == 64`：secp256k1 下 R、S 各 32 字节。

### 4.5 流程小结（2-of-3 单机）

```
[Keygen]
  n1, n2, n3 各自生成 PreParams（若无预生成）
       ↓
  多轮 TSS keygen 消息经 InProcessRouter 在 n1↔n2↔n3 间转发
       ↓
  每方得到 SaveData[i]，共同公钥对应 result.KeyID

[Sign]
  同一 nodeIDs + result.SaveData + msgHash
       ↓
  多轮 TSS sign 消息经同一 router 转发
       ↓
  得到 64 字节 Signature (R||S)
```

---

## 五、常见问题与注意点

### 5.1 超时

- Keygen 默认 5 分钟超时（见 `keygen.go`），Sign 默认 2 分钟。
- 首次 keygen 因要生成 PreParams，可能接近 1～2 分钟，属正常。
- 若机器较慢，可适当增大 `go test -timeout=5m`。

### 5.2 PreParams 想加速

可预先为每个 party 生成一次 PreParams 并保存，测试时以 `preParams []*keygen.LocalPreParams` 形式传入 `RunKeygen`，并设 `PreParamsTimeout: 0`，则 keygen 阶段会明显变快（仅跑协议，不生成素数）。

### 5.3 曲线与链

当前测试（及 mpc 实现）使用的是 **secp256k1**（BTC/ETH 等）。签名结果 64 字节 R||S 可直接用于这些链；若链使用其他曲线或编码，需在业务层再做一次转换。

### 5.4 Sign 报错 "failed to calculate Alice_end or Alice_end_wc"

若 keygen 阶段从 `endCh` 收集 SaveData 时**按完成顺序**追加，而未按 party 下标排序，则 RunSign 时会把「甲方的 save」误当成「乙方的 save」用，协议在 round 3 左右会报错（culprits 指向其他方）。修复方式：keygen 收集时用 `save.OriginalIndex()` 得到该 save 对应的 party 下标，放入 `SaveData[idx]`，保证 **SaveData[i] 始终对应第 i 方**。当前实现已按此修复。

### 5.5 和真实多节点的关系

- 测试里 **三方都在同一进程**，用 `InProcessRouter` 模拟网络。
- 真实部署时：每个节点只跑**一个** party，消息通过 WebSocket/HTTP 由服务端转发；需实现 `MessageRouter` 的另一个实现（例如 WsRouter），在 `Send` 里把消息发到对应节点，在 `Receive` 里把收到的字节交给本节点 party 的 `Update`。

---

## 六、快速命令速查

```bash
# 只跑 PartyIDs（快）
go test -v ./mpc/ -run TestPartyIDs

# 跑完整 keygen+sign（慢，约 1～3 分钟）
go test -v -timeout=5m ./mpc/ -run TestKeygenAndSignInProcess

# 跑全部 mpc 测试（含上面两个）
go test -v -timeout=5m ./mpc/

# CI：跳过慢测试
go test -v -short ./mpc/
```

按上述顺序跑一遍，再结合本文对每个断言和步骤的说明，即可把测试用例当「最小可运行示例」来理解和扩展。
