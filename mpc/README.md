# mpc — 多算法 MPC 门限签名

本目录按**算法**拆分子包，便于后续同时支持多种签名算法（如 ECDSA、Ed25519），上层 WebSocket 协议与协调逻辑可复用，仅底层引擎按算法切换。

## 目录结构

| 路径 | 说明 |
|------|------|
| **mpc/** | 根包：公共类型与常量（如 `Algorithm`：`AlgECDSA`、`AlgEd25519`），无具体实现。 |
| **mpc/alg_ecdsa/** | ECDSA (t,n) 门限签名，基于 `bnb-chain/tss-lib`。keygen、sign、账户派生、KeyStore 等完整实现。 |
| **mpc/alg_ed25519/** | Ed25519 门限签名（预留）。当前仅占位，后续可接入支持 Ed25519 的 MPC 库。 |

## 使用方式

- **当前仅 ECDSA**：业务侧（node、application、web）直接引用 `github.com/blocktree/go-openw-sdk/v2/mpc/alg_ecdsa`，使用 `PartyIDs`、`Parameters`、`RunKeygen`、`RunSign`、`NewFileKeyStore`、`DeriveMPCAccountFromRootPubHex` 等。
- **算法标识**：KeyMeta、任务参数等可携带 `mpc.AlgECDSA` / `mpc.AlgEd25519`，便于上层按算法选择对应引擎。
- **后续扩展 Ed25519**：在 `alg_ed25519` 中实现与协议层一致的「消息进/出」接口，上层根据 `Algorithm` 调用对应子包即可。

## 协议与引擎解耦

- 传输层（WebSocket）：只关心「不透明消息 + TaskID + Subject」，不解析 TSS 明文。
- 各算法子包负责：本算法的 keygen/sign 协议消息格式、SaveData 结构、账户派生规则；对外提供与当前 DTO 一致的「按目标加密发送 / 解密接收」对接方式。

详见各子包内 README（如 `alg_ecdsa/README.md`）。
