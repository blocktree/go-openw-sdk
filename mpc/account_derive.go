package mpc

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/bnb-chain/tss-lib/crypto"
	"github.com/bnb-chain/tss-lib/crypto/ckd"
	"github.com/bnb-chain/tss-lib/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/tss"

	"github.com/blocktree/openwallet/v2/openwallet"
	"github.com/btcsuite/btcd/chaincfg"
)

// DeriveChildPubFromPath 从 MPC 根公钥 + chainCode + 路径派生子公钥（仅非硬化路径，与 BIP32 一致）。
// 返回 keyDerivationDelta（签名时需传给 RunSignWithKDD）和子公钥（用于生成 AccountID/地址）。
// 不涉及私钥，仅用公钥和 path 即可在服务端/客户端派生。
func DeriveChildPubFromPath(rootPub *crypto.ECPoint, chainCode []byte, path []uint32) (keyDerivationDelta *big.Int, childPub *ecdsa.PublicKey, err error) {
	if rootPub == nil || len(chainCode) < 32 {
		return nil, nil, fmt.Errorf("mpc: rootPub and 32-byte chainCode required")
	}
	ec := tss.S256()
	pk := ecdsa.PublicKey{
		Curve: ec,
		X:     rootPub.X(),
		Y:     rootPub.Y(),
	}
	cc := make([]byte, 32)
	copy(cc, chainCode)
	net := &chaincfg.MainNetParams
	extendedParent := &ckd.ExtendedKey{
		PublicKey:  pk,
		Depth:      0,
		ChildIndex: 0,
		ChainCode:  cc,
		ParentFP:   []byte{0x00, 0x00, 0x00, 0x00},
		Version:    net.HDPrivateKeyID[:],
	}
	delta, extendedChild, err := ckd.DeriveChildKeyFromHierarchy(path, extendedParent, ec.Params().N, ec)
	if err != nil {
		return nil, nil, err
	}
	childPub = &extendedChild.PublicKey
	return delta, childPub, nil
}

// ChainCodeFromKeyID 用 KeyID 生成 32 字节 chainCode，便于无 seed 时仍有一致派生。
func ChainCodeFromKeyID(keyID string) []byte {
	h := sha256.Sum256([]byte(keyID))
	return h[:]
}

// PubKeyToHex 将 ECDSA 公钥转为 65 字节非压缩 hex（04||X||Y），便于与 GenAccountID/地址派生对接。
func PubKeyToHex(pub *ecdsa.PublicKey) string {
	if pub == nil {
		return ""
	}
	const size = 65
	b := make([]byte, size)
	b[0] = 0x04
	copy(b[1:33], Pad32(pub.X.Bytes()))
	copy(b[33:65], Pad32(pub.Y.Bytes()))
	return hex.EncodeToString(b)
}

// PathFromAccountIndex 生成单层非硬化路径 []uint32{accountIndex}，用于 DeriveChildPubFromPath。
// 若需多级路径（如 m/44/60/0/0/index），可传 []uint32{44, 60, 0, 0, index}。
func PathFromAccountIndex(accountIndex uint32) []uint32 {
	return []uint32{accountIndex}
}

// DeriveMPCAccountFromIndex 从 MPC 根 SaveData 派生出第 index 个账户的 AccountID 与公钥 hex。
// 仅依赖根公钥(ECDSAPub)、KeyID 与 index，不需要私钥或种子。
func DeriveMPCAccountFromIndex(
	save *keygen.LocalPartySaveData,
	keyID string,
	index uint32,
) (accountID string, pubHex string, err error) {
	if save == nil || save.ECDSAPub == nil {
		return "", "", fmt.Errorf("mpc: nil save data or ECDSAPub")
	}

	// 1) 根公钥（tss ECPoint -> ecdsa.PublicKey）
	rootPub := save.ECDSAPub

	// 2) 基于 KeyID 生成 chainCode（32 字节），保证同一 KeyID 派生稳定可复现
	chainCode := ChainCodeFromKeyID(keyID)

	// 3) 使用 index 生成路径（当前为单层路径 []uint32{index}）
	path := PathFromAccountIndex(index)

	// 4) 根据路径派生子公钥（同时会返回签名用的 delta，这里建账户时不返回）
	_, childPub, err := DeriveChildPubFromPath(rootPub, chainCode, path)
	if err != nil {
		return "", "", err
	}

	// 5) 子公钥编码为旧系统兼容的 hex 公钥字符串
	pubHex = PubKeyToHex(childPub)

	// 6) 使用原有 openwallet.GenAccountID 算出 AccountID（与旧系统保持一致）
	accountID = openwallet.GenAccountID(pubHex)

	return accountID, pubHex, nil
}

// DeriveMPCAccountFromKeyStore 从本地 MPC keyfile（SaveData）派生账户信息。
// baseDir 对应 NewFileKeyStore(baseDir) 的目录（例如 "keys"）。
// nodeID 为本节点 ID（用于定位 keyfile：{keyID}-{nodeID}.json）。
func DeriveMPCAccountFromKeyStore(baseDir, keyID, nodeID string, index uint32) (accountID string, pubHex string, err error) {
	store := NewFileKeyStore(baseDir)
	save, err := store.Load(keyID, nodeID)
	if err != nil {
		return "", "", err
	}
	return DeriveMPCAccountFromIndex(&save, keyID, index)
}

// DeriveMPCAccountFromRootPubHex 从根公钥 hex（RootPubHex）+ KeyID + index 派生 AccountID 与公钥 hex。
// 适用于服务端：只持有 RootPubHex 与 KeyID，而不持有 SaveData。
func DeriveMPCAccountFromRootPubHex(rootPubHex, keyID string, index uint32) (accountID string, pubHex string, err error) {
	if rootPubHex == "" {
		return "", "", fmt.Errorf("mpc: empty rootPubHex")
	}
	b, err := hex.DecodeString(rootPubHex)
	if err != nil {
		return "", "", fmt.Errorf("decode rootPubHex: %w", err)
	}
	if len(b) != 65 || b[0] != 0x04 {
		return "", "", fmt.Errorf("mpc: invalid rootPubHex format")
	}

	// 解析 04||X||Y
	x := new(big.Int).SetBytes(b[1:33])
	y := new(big.Int).SetBytes(b[33:65])

	ec := tss.S256()
	point, err := crypto.NewECPoint(ec, x, y)
	if err != nil {
		return "", "", fmt.Errorf("mpc: invalid EC point: %w", err)
	}

	chainCode := ChainCodeFromKeyID(keyID)
	path := PathFromAccountIndex(index)

	_, childPub, err := DeriveChildPubFromPath(point, chainCode, path)
	if err != nil {
		return "", "", err
	}

	pubHex = PubKeyToHex(childPub)
	accountID = openwallet.GenAccountIDByHex(pubHex)
	return accountID, pubHex, nil
}
