// Package mpc 提供 MPC 多算法支持的公共类型与常量。
// 具体实现按算法拆分到子包，便于后续支持多种签名算法（如 ECDSA、Ed25519）。
//
//   - mpc/alg_ecdsa：基于 bnb-chain/tss-lib 的 ECDSA (t,n) 门限签名
//   - mpc/alg_ed25519：Ed25519 门限签名（预留，待实现）
package mpc

// Algorithm 表示 MPC 使用的签名算法。
// 用于 KeyMeta、任务参数等处，便于上层按算法选择对应引擎。
type Algorithm string

const (
	// AlgECDSA secp256k1 ECDSA 门限签名（当前由 alg_ecdsa 实现）
	AlgECDSA Algorithm = "ecdsa"
	// AlgEd25519 Ed25519 门限签名（预留，由 alg_ed25519 实现）
	AlgEd25519 Algorithm = "ed25519"
)
