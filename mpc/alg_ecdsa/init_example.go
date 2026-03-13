// 本文件演示 MPC 密钥初始化流程：如何跑 keygen、保存/下发 SaveData、与门限/节点数的关系。
// 可直接复制到业务代码中按需修改（如接 Web 接口、持久化、PreParams 预生成等）。
package alg_ecdsa

import (
	"fmt"
	"sort"
	"time"

	"github.com/bnb-chain/tss-lib/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/tss"
)

// InitKeyRequest 初始化密钥的入参（可由 HTTP/WebSocket 传入）。
type InitKeyRequest struct {
	// NodeIDs 参与 keygen 的节点 ID 列表（如 WebSocket subject），至少 2 个；3 或 5 对应 2-of-3 / 3-of-5。
	NodeIDs []string
	// WalletMode 与现有配置一致：3 = 2-of-3，5 = 3-of-5；用于确定 threshold。
	WalletMode int64
}

// InitKeyResponse 初始化密钥的返回（KeyID 可写库，SaveData 需按节点下发或持久化）。
type InitKeyResponse struct {
	KeyID     string
	KeyByNode map[string]keygen.LocalPartySaveData // nodeID -> 该节点的 SaveData，用于下发或存 KeyStore
}

// ThresholdFromWalletMode 根据 WalletMode 返回门限 t。
func ThresholdFromWalletMode(mode int64) int {
	switch mode {
	case 3:
		return 2 // 2-of-3
	case 5:
		return 3 // 3-of-5
	default:
		return 0
	}
}

// InitKey 执行一次 TSS keygen，生成门限密钥并返回 KeyID 与按节点划分的 SaveData。
// 使用内存 InProcessRouter，适用于「所有 party 在同一进程」的场景（如测试或服务端集中跑 keygen）。
func InitKey(req InitKeyRequest, config KeygenConfig, preParams []*keygen.LocalPreParams) (*InitKeyResponse, error) {
	if len(req.NodeIDs) < 2 {
		return nil, fmt.Errorf("mpc: need at least 2 nodes, got %d", len(req.NodeIDs))
	}
	threshold := ThresholdFromWalletMode(req.WalletMode)
	if threshold == 0 {
		return nil, fmt.Errorf("mpc: unsupported walletMode %d (use 3 or 5)", req.WalletMode)
	}
	if len(req.NodeIDs) == 3 && threshold != 2 {
		return nil, fmt.Errorf("mpc: 3 nodes require threshold 2 (walletMode=3)")
	}
	if len(req.NodeIDs) == 5 && threshold != 3 {
		return nil, fmt.Errorf("mpc: 5 nodes require threshold 3 (walletMode=5)")
	}

	errCh := make(chan *tss.Error, len(req.NodeIDs)*2)
	router := &InProcessRouter{ErrCh: errCh}

	result, err := RunKeygen(req.NodeIDs, threshold, router, config, preParams)
	if err != nil {
		return nil, err
	}

	byNode := KeygenResultByNode(req.NodeIDs, result)
	return &InitKeyResponse{KeyID: result.KeyID, KeyByNode: byNode}, nil
}

// ExampleInitKey 演示一次完整的密钥初始化（可直接在测试或 main 里调用）。
func ExampleInitKey() {
	// 1) 参与节点 ID（与现有 sharding 的 subject 一致即可）
	nodeIDs := []string{"node-1", "node-2", "node-3"}
	// 2) 门限配置：3 节点用 2-of-3
	req := InitKeyRequest{NodeIDs: nodeIDs, WalletMode: 3}
	config := KeygenConfig{
		PreParamsTimeout: 1 * time.Minute,
		Concurrency:      0,
	}
	// 3) 可选：预生成 PreParams 以加速（首次可传 nil，会现场生成，较慢）
	var preParams []*keygen.LocalPreParams

	resp, err := InitKey(req, config, preParams)
	if err != nil {
		fmt.Println("init key failed:", err)
		return
	}

	fmt.Println("KeyID:", resp.KeyID)
	// 4) 按节点保存到本地（重启后可 Load 回来）
	keyStore := NewFileKeyStore("./mpc_keys") // 或绝对路径
	for nodeID, data := range resp.KeyByNode {
		if err := keyStore.Save(resp.KeyID, nodeID, data); err != nil {
			fmt.Println("save failed:", nodeID, err)
			return
		}
	}
}

// ExampleLoadAndSign 演示重启后如何加载 SaveData 并完成一次签名。
func ExampleLoadAndSign() {
	keyID := "26a48599f88d40e12669700f3c84e969e06d8a6748c89338b8435eb2b7ea670a"
	nodeIDs := []string{"node-1", "node-2", "node-3"}
	keyStore := NewFileKeyStore("./mpc_keys")

	// 从磁盘加载该 key 下所有节点的 SaveData（顺序与 PartyIDs 一致，可直接用于 RunSign）
	_, keysInOrder, err := keyStore.LoadAllForKey(keyID, nodeIDs)
	if err != nil {
		fmt.Println("load keys failed:", err)
		return
	}

	// 用加载的 keys 做一次签名（msgHash 需为待签名的 32 字节哈希的 big.Int）
	// msgHash, _ := MessageHashFromTxHash(hexTxHash)
	// sig, err := RunSign(nodeIDs, keysInOrder, msgHash, router)
	_ = keysInOrder
}

// SortedNodeIDs 返回与 PartyIDs 顺序一致的节点 ID 列表（用于签名时保证顺序）。
func SortedNodeIDs(nodeIDs []string) []string {
	ids := make([]string, len(nodeIDs))
	copy(ids, nodeIDs)
	sort.Strings(ids)
	return ids
}
