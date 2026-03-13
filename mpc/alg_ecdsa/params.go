// Package alg_ecdsa 基于 bnb-chain/tss-lib 的 ECDSA (t,n) 门限签名实现。
// keygen 与 sign 均不重建完整私钥，供 mpc 上层按算法选用。
package alg_ecdsa

import (
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"sort"

	"github.com/bnb-chain/tss-lib/tss"
)

// PartyIDs 从一组节点 ID（如 WebSocket subject）生成排序后的 tss.PartyID 列表。
// 每个 id 会对应一个唯一的 key（基于 id 的哈希），保证多方排序稳定。
func PartyIDs(nodeIDs []string) tss.SortedPartyIDs {
	if len(nodeIDs) == 0 {
		return nil
	}
	// 复制并排序，保证顺序稳定
	ids := make([]string, len(nodeIDs))
	copy(ids, nodeIDs)
	sort.Strings(ids)

	out := make(tss.UnSortedPartyIDs, len(ids))
	h := sha256.New()
	for i, id := range ids {
		h.Reset()
		h.Write([]byte(id))
		keyBytes := h.Sum(nil)
		key := new(big.Int).SetBytes(keyBytes)
		out[i] = tss.NewPartyID(id, id, key)
	}
	return tss.SortPartyIDs(out)
}

// Parameters 为某一方构建 tss.Parameters。
// partyIndex 为该方在 sortedIDs 中的下标（0..n-1），threshold 为门限 t。
func Parameters(sortedIDs tss.SortedPartyIDs, partyIndex int, threshold int) *tss.Parameters {
	if partyIndex < 0 || partyIndex >= len(sortedIDs) || threshold < 1 || threshold > len(sortedIDs) {
		return nil
	}
	ctx := tss.NewPeerContext(sortedIDs)
	return tss.NewParameters(tss.S256(), ctx, sortedIDs[partyIndex], len(sortedIDs), threshold)
}

// KeyIDFromSaveData 从 keygen 的 LocalPartySaveData 导出公钥标识（与 HD keyID 概念一致）。
// 使用 ECDSA 公钥点的字节做 SHA256 再 hex，便于与现有 keyID 对接。
func KeyIDFromSaveData(pubX, pubY *big.Int) string {
	if pubX == nil || pubY == nil {
		return ""
	}
	h := sha256.New()
	h.Write(Pad32(pubX.Bytes()))
	h.Write(Pad32(pubY.Bytes()))
	return hex.EncodeToString(h.Sum(nil))
}

// Pad32 将 b 填充或截断为 32 字节（前导零）。
func Pad32(b []byte) []byte {
	if len(b) >= 32 {
		return b[:32]
	}
	out := make([]byte, 32)
	copy(out[32-len(b):], b)
	return out
}
