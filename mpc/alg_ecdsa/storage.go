package alg_ecdsa

import (
	"github.com/bnb-chain/tss-lib/ecdsa/keygen"
)

// KeyStore 用于按 keyID + nodeID 存取各节点的 keygen 结果。
// 实现方可将 SaveData 持久化到文件/DB，或经加密后下发给节点（与现有 shardingPost 类似）。
type KeyStore interface {
	Save(keyID, nodeID string, data keygen.LocalPartySaveData) error
	Load(keyID, nodeID string) (keygen.LocalPartySaveData, error)
}

// KeygenResultByNode 将 KeygenResult 按 nodeID 拆成「节点 -> SaveData」便于下发或存储。
// 注意：SaveData 顺序与 PartyIDs(nodeIDs) 的排序一致，故按 sorted 后的 party Id 映射。
func KeygenResultByNode(nodeIDs []string, result *KeygenResult) map[string]keygen.LocalPartySaveData {
	if result == nil || len(result.SaveData) != len(nodeIDs) {
		return nil
	}
	sortedIDs := PartyIDs(nodeIDs)
	out := make(map[string]keygen.LocalPartySaveData, len(sortedIDs))
	for i := range sortedIDs {
		out[sortedIDs[i].GetId()] = result.SaveData[i]
	}
	return out
}
