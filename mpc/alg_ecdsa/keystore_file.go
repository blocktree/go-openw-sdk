package alg_ecdsa

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bnb-chain/tss-lib/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/tss"
)

// FileKeyStore 基于目录的 KeyStore 实现：每个 (keyID, nodeID) 存为一份 JSON 文件。
// 目录结构：BaseDir / {keyID}-{nodeID}.json
// 重启后通过 Load(keyID, nodeID) 即可恢复该节点的 SaveData。
//
// 安全说明：SaveData 含 Paillier 私钥、秘密份额等，等同于私钥级别，必须保密。
// - 当前实现为明文 JSON + 文件权限 0600，仅适合测试或可信环境。
// - 生产建议：落盘前加密（如 AES-GCM，密钥由密码/KMS 派生），或存于加密卷/权限严格目录；
//   若下发到节点，必须走加密通道（如现有 ECDH shardingPost），节点侧再加密存或存安全模块。
type FileKeyStore struct {
	BaseDir string
}

// NewFileKeyStore 创建基于目录的 KeyStore，baseDir 建议用绝对路径或相对程序工作目录。
func NewFileKeyStore(baseDir string) *FileKeyStore {
	return &FileKeyStore{BaseDir: baseDir}
}

// sanitize 去掉会破坏路径的字符，避免路径遍历。
func sanitize(s string) string {
	s = strings.ReplaceAll(s, string(filepath.Separator), "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	if s == "" {
		return "_"
	}
	return s
}

func (f *FileKeyStore) path(keyID, nodeID string) string {
	return filepath.Join(f.BaseDir, sanitize(keyID)+"-"+sanitize(nodeID)+".json")
}

// Save 将 data 序列化为 JSON 写入 BaseDir/{keyID}-{nodeID}.json（新规则）。
// 文件权限 0700(目录)/0600(文件)。内容为明文，敏感；生产环境应对内容加密或使用加密存储。
func (f *FileKeyStore) Save(keyID, nodeID string, data keygen.LocalPartySaveData) error {
	path := f.path(keyID, nodeID)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	raw, err := json.Marshal(&data)
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0600)
}

// Load 从 BaseDir/{keyID}-{nodeID}.json 读取并反序列化，并恢复曲线（SetCurve）。
// 重启后对每个 (keyID, nodeID) 调用 Load 即可得到该节点的 SaveData，用于 RunSign。
func (f *FileKeyStore) Load(keyID, nodeID string) (keygen.LocalPartySaveData, error) {
	path := f.path(keyID, nodeID)
	raw, err := os.ReadFile(path)
	if err != nil {
		return keygen.LocalPartySaveData{}, err
	}
	return DecodeSaveDataFromJSON(raw)
}

// DecodeSaveDataFromJSON 从 JSON 字节反序列化 LocalPartySaveData 并恢复曲线。
// 用于服务端收到节点上报的 SaveData（如 base64 解码后）后解码并落盘。
func DecodeSaveDataFromJSON(raw []byte) (keygen.LocalPartySaveData, error) {
	var data keygen.LocalPartySaveData
	if err := json.Unmarshal(raw, &data); err != nil {
		return keygen.LocalPartySaveData{}, err
	}
	if data.ECDSAPub != nil {
		data.ECDSAPub.SetCurve(tss.S256())
	}
	for _, p := range data.BigXj {
		if p != nil {
			p.SetCurve(tss.S256())
		}
	}
	return data, nil
}

// LoadAllForKey 加载某一 keyID 下所有已存在的 nodeID 的 SaveData。
// nodeIDs 为当前参与该 key 的节点列表（如配置或注册表），只加载存在的文件。
// 返回 map[nodeID]SaveData 以及按 nodeIDs 顺序的 keys 切片（用于 RunSign）。
func (f *FileKeyStore) LoadAllForKey(keyID string, nodeIDs []string) (byNode map[string]keygen.LocalPartySaveData, keysInOrder []keygen.LocalPartySaveData, err error) {
	byNode = make(map[string]keygen.LocalPartySaveData)
	keysInOrder = make([]keygen.LocalPartySaveData, 0, len(nodeIDs))
	sortedIDs := PartyIDs(nodeIDs)
	for i := range sortedIDs {
		nodeID := sortedIDs[i].GetId()
		data, err := f.Load(keyID, nodeID)
		if err != nil {
			return nil, nil, fmt.Errorf("load key %s node %s: %w", keyID, nodeID, err)
		}
		byNode[nodeID] = data
		keysInOrder = append(keysInOrder, data)
	}
	return byNode, keysInOrder, nil
}
