# MPC 下「无 seed」的 AccountID / 地址派生

原先流程是：**seed → 派生 AccountID → 再派生地址**。MPC 没有单点私钥/seed，改为用**根公钥 + chainCode + 路径**在服务端/客户端派生，私钥始终以分片形式存在各节点，不参与“账户/地址”的展示。

---

## 1. 概念对应

| 原流程（有 seed） | MPC 流程（无 seed） |
|------------------|---------------------|
| seed（主密钥） | **KeyID** + **根公钥**（SaveData 里的 ECDSAPub）+ **chainCode**（32 字节，如 `ChainCodeFromKeyID(KeyID)`） |
| 按路径派生子密钥 | **DeriveChildPubFromPath**(根公钥, chainCode, path) → 得到 **keyDerivationDelta** 和 **子公钥** |
| AccountID = GenAccountID(子公钥) | 子公钥用 **PubKeyToHex** 等编码后，仍用现有 **GenAccountID** 生成 AccountID |
| 地址 = 链规则(子公钥) | 不变，仍由子公钥按链规则算地址（如 ETH/BTC） |
| 用子密钥签名 | 用**根 SaveData** + **keyDerivationDelta** + **子公钥** 调 **RunSignWithKDD** 做签名 |

---

## 2. 初始化时（keygen 后）

- 得到 **KeyID** 和每节点的 **SaveData**。
- **根公钥**：任意一份 SaveData 的 `save.ECDSAPub`（所有方一致）。
- **chainCode**：无 seed 时可用 `mpc.ChainCodeFromKeyID(KeyID)` 得到 32 字节，与 KeyID 绑定、可复现。

---

## 3. 派生 Account / 地址（给客户端用）

```go
// 根公钥：从已加载的 SaveData 取（例如 SaveData[0].ECDSAPub）
rootPub := saveData.ECDSAPub
chainCode := mpc.ChainCodeFromKeyID(keyID)

// 账户索引 → 路径（仅非硬化，如 0, 1, 2…）
path := mpc.PathFromAccountIndex(accountIndex)

delta, childPub, err := mpc.DeriveChildPubFromPath(rootPub, chainCode, path)
if err != nil { ... }

// 与现有逻辑对接：公钥编码 → AccountID / 地址
pubKeyHex := mpc.PubKeyToHex(childPub)
accountID := openwallet.GenAccountID(pubKeyHex)  // 或你们现有的 GenAccountID(publicKey)
// 地址：按链规则从 childPub 生成（如 Keccak256 后取后 20 字节等）
```

- **HdPath**：可按你们现有规则用 path 拼字符串（例如 `"m/44/60/0/0/" + strconv.FormatUint(uint64(accountIndex), 10)`），仅用于展示或索引，不参与密码学。
- 同一 **KeyID + path + chainCode** 每次派生产生的 **delta / 子公钥** 一致，因此 **AccountID / 地址** 稳定。

---

## 4. 签名该账户的交易

- 仍用**根的 SaveData**（各节点存的 keygen 结果），不存“子密钥”。
- 对应该账户的 **path** 再算一次 **delta** 和 **childPub**（与派生 Account/地址时相同 path）。
- 调用 **RunSignWithKDD**，用根 keys + 该 path 的 delta + childPub：

```go
delta, childPub, _ := mpc.DeriveChildPubFromPath(rootPub, chainCode, path)
sig, err := mpc.RunSignWithKDD(nodeIDs, keys, msgHash, delta, childPub, router)
```

这样签出来的就是该“子账户”对应的签名，和以前用 seed 派生该路径再签名的效果一致。

---

## 5. 小结

- **没有 seed**：根密钥 = KeyID + 根公钥(ECDSAPub) + chainCode；子“账户” = 路径派生出的子公钥 + 同一 path 的 delta。
- **AccountID / 地址**：只依赖**子公钥**，用现有 GenAccountID 和链规则即可，无需私钥。
- **签名**：用根 SaveData + **RunSignWithKDD**(msgHash, delta, childPub, …)，由 TSS 用分片私钥 + delta 生成该子密钥的签名。
