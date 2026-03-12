package webapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/blocktree/go-openw-sdk/v2/mpc"
	"github.com/blocktree/go-openw-sdk/v2/openwsdk/dto"
	"github.com/blocktree/openwallet/v2/hdkeystore"
	ecc "github.com/godaddy-x/eccrypto"
	"github.com/godaddy-x/freego/node"
	"github.com/godaddy-x/freego/utils"
)

// mpcKeygenLogMu 串行化 mpc-keygen 相关日志，避免 CreateMPCKeyTask 轮询与 handleMpcKeygenMsg 并发写 stdout 导致交错。
var mpcKeygenLogMu sync.Mutex

// keyMetaDir 服务端记录 walletID 对应节点列表的本地目录。
// 每次 keygen 成功后会写入 keyMetaDir/{walletID}.json。
const keyMetaDir = "mpc_keys_meta"

func mpcLogf(format string, args ...interface{}) {
	mpcKeygenLogMu.Lock()
	fmt.Printf("[mpc-keygen] "+format, args...)
	mpcKeygenLogMu.Unlock()
}

// MpcKeygenTaskMeta 按 taskID 存的 MPC keygen 任务元信息（NodeIDs、门限等），用于转发 TSS 消息时查表。
type MpcKeygenTaskMeta struct {
	TaskID      string
	AllNodeIDs  []string
	SignNodeIDs []string
	Threshold   int
	ExpiredTime int64
	PublicKey   map[string][]byte // subject -> temp ECDH public key (raw bytes)
}

// KeyMeta 用于持久化一把 key 的元信息到本地 JSON 文件（keyMetaDir/{walletID}.json）。
// 方便在服务端配置丢失时，从文件恢复参与节点与门限配置。
type KeyMeta struct {
	WalletID      string         `json:"walletID"`
	KeyID         string         `json:"keyID"`
	RootPubHex    string         `json:"rootPubHex"`    // 65-byte uncompressed pubkey hex (04||X||Y)
	NodeIDs       []string       `json:"nodeIDs"`       // 按 TSS PartyIDs 顺序
	Threshold     int            `json:"threshold"`     // 门限 t（如 2-of-3、3-of-5）
	IndexByNodeID map[string]int `json:"indexByNodeID"` // nodeID -> index（在 NodeIDs 中的下标）
}

// MpcKeygenNodeResult 按 (subject, taskID) 存的节点上报结果，Status 40 表示已上报 SaveData。
type MpcKeygenNodeResult struct {
	TaskID     string
	NodeID     string
	Status     int64  // 10=已下发 start，40=已上报结果
	KeyID      string // 仅用于各节点自报的一致性校验，服务端不落盘 SaveData
	PublicKey  string // 节点临时公钥
	Err        string
	RootPubHex string
}

func truncateErr(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// CreateMPCKeygenTask 协调多节点完成一次 TSS keygen，轮询直到所有节点上报结果后返回 KeyID。
// 服务端只做「协调 + 校验」，不落盘任何 SaveData：节点各自持久化自己的 LocalPartySaveData。
// 流程：1) 取在线 subject 排序为 nodeIDs  2) 下发 mpcKeygenStart  3) 轮询 mpcKeygenResult 检查状态与 KeyID 一致性  4) 返回 KeyID。
func CreateMPCKeygenTask() (keyID string, err error) {
	if server == nil {
		return "", errors.New("ws server not initialized")
	}

	nodes := server.GetConnManager().GetAllSubjectDevices()
	subjects := make([]string, 0, len(nodes))
	for s := range nodes {
		subjects = append(subjects, s)
	}
	sort.Strings(subjects)

	// 注意：TSS 的 party index 由 tss.SortPartyIDs（按 PartyID.Key）决定，
	// 并不等于字符串排序 subjects 的下标。服务端下发/保存的 NodeIDs 必须与 mpc.PartyIDs() 顺序一致，
	// 否则节点计算的 myIndex/fromIndex 会与服务端转发时用的 (meta.NodeIDs, fromIndex) 映射错位。
	partyIDs := mpc.PartyIDs(subjects)
	nodeIDs := make([]string, 0, len(partyIDs))
	for _, pid := range partyIDs {
		nodeIDs = append(nodeIDs, pid.GetId())
	}

	if !utils.CheckInt(len(subjects), 3, 5) {
		return "", errors.New(utils.AddStr("online nodes: ", len(subjects), " (3 or 5 required)"))
	}

	threshold := 2
	if len(subjects) == 5 {
		threshold = 3
	}

	taskID := utils.GetUUID(true)
	expiredTime := utils.UnixSecond() + 120 // 2 分钟，keygen 可能较慢

	mpcLogf("CreateMPCKeyTask: taskID=%s subjects=%v nodeIDs(tss-order)=%v threshold=%d\n", taskID, subjects, nodeIDs, threshold)

	// 发送通知节点提交临时公钥（事件驱动等待，避免轮询）
	timeout := 130
	collector := NewPubkeyCollector(nodeIDs)
	registerPubkeyCollector("keygen", taskID, collector, time.Duration(timeout)*time.Second)
	defer unregisterPubkeyCollector("keygen", taskID)

	for _, subject := range nodeIDs {
		req := &dto.CliMPCTempPublicKeyReq{
			TaskID: taskID,
			Module: "keygen",
		}
		if err := server.GetConnManager().SendToSubject(subject, "mpcTempPublicKey", req); err != nil {
			return "", err
		}
	}

	// cache replay：如果有节点在 collector 注册前已上报（或重试上报），这里补一遍 Submit，确保不会丢事件
	for _, subject := range nodeIDs {
		cacheKey := utils.FNV1a64(utils.AddStr(subject, ":", taskID, ":keygen:tempPublicKey"))
		v, ok, _ := keyCache.Get(cacheKey, nil)
		if ok && v != nil {
			collector.Submit(subject, v.([]byte))
		}
	}

	waitCtx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout-5)*time.Second)
	defer cancel()
	if err := collector.Wait(waitCtx); err != nil {
		return "", fmt.Errorf("timeout waiting for all nodes to submit public keys: %w", err)
	}

	meta := &MpcKeygenTaskMeta{
		TaskID:      taskID,
		AllNodeIDs:  nodeIDs,
		Threshold:   threshold,
		ExpiredTime: expiredTime,
		PublicKey:   collector.GetPubkeys(),
	}

	metaKey := utils.FNV1a64("mpcMeta:" + taskID)
	if err := keyCache.Put(metaKey, meta, 600); err != nil {
		return "", err
	}

	startPayload := &dto.CliMPCKeygenStartRes{
		TaskID:        taskID,
		NodeIDs:       nodeIDs,
		Threshold:     threshold,
		ExpiredTime:   expiredTime,
		PublicKeyPair: make([]dto.CliMPCPublicKeyPair, 0, len(meta.PublicKey)),
	}

	for _, v := range nodeIDs {
		startPayload.PublicKeyPair = append(startPayload.PublicKeyPair, dto.CliMPCPublicKeyPair{
			Subject:   v,
			PublicKey: utils.Base64Encode(meta.PublicKey[v]),
		})
	}

	for _, subject := range nodeIDs {
		data, err := utils.JsonMarshal(startPayload)
		if err != nil {
			return "", err
		}
		encrypt, err := ecc.Encrypt(nil, meta.PublicKey[subject], data, utils.Str2Bytes(utils.AddStr(taskID, "|", subject, "|mpcKeygenStart")))
		if err != nil {
			return "", err
		}

		if err := server.GetConnManager().SendToSubject(subject, "mpcKeygenStart", &dto.CliMPCEncryptData{TaskID: taskID, Data: utils.Base64Encode(encrypt)}); err != nil {
			return "", err
		}
		nodeResult := &MpcKeygenNodeResult{
			TaskID: taskID,
			NodeID: subject,
			Status: 10,
		}
		cacheKey := utils.FNV1a64(utils.AddStr(subject, taskID))
		if err := keyCache.Put(cacheKey, nodeResult, 300); err != nil {
			return "", err
		}
	}

	// 等待 keygen 结果上报（事件驱动，无需轮询 cache）
	resultCollector := NewKeygenResultCollector(nodeIDs)
	registerKeygenResultCollector(taskID, resultCollector, time.Duration(timeout)*time.Second)
	defer unregisterKeygenResultCollector(taskID)

	// cache replay：如果有节点在 collector 注册前已上报（或重试上报），这里补一遍 Submit，确保不会丢事件
	for _, subject := range nodeIDs {
		cacheKey := utils.FNV1a64(utils.AddStr(subject, taskID))
		v, ok, _ := keyCache.Get(cacheKey, nil)
		if ok && v != nil {
			if res, ok := v.(*MpcKeygenNodeResult); ok && res != nil {
				resultCollector.Submit(subject, res)
			}
		}
	}

	waitResCtx, cancelRes := context.WithTimeout(context.Background(), time.Duration(timeout-5)*time.Second)
	defer cancelRes()
	if err := resultCollector.Wait(waitResCtx); err != nil {
		mpcLogf("CreateMPCKeyTask: timeout waiting for nodes, taskID=%s\n", taskID)
		return "", fmt.Errorf("timeout waiting for all nodes to submit MPC keygen result: %w", err)
	}

	results := resultCollector.GetResults()
	var firstKeyID string
	for _, subject := range nodeIDs {
		res := results[subject]
		if res == nil {
			return "", errors.New("missing keygen result from " + subject)
		}
		if res.Err != "" {
			return "", errors.New("node " + res.NodeID + " keygen failed: " + res.Err)
		}
		if firstKeyID == "" {
			firstKeyID = res.KeyID
		} else if res.KeyID != firstKeyID {
			return "", errors.New("keyID mismatch between nodes")
		}
	}
	keyID = firstKeyID
	if keyID == "" {
		return "", errors.New("keyID empty")
	}

	walletID := hdkeystore.ComputeKeyID([]byte(keyID))

	// 基于节点上报的 RootPubHex 校验根公钥一致性
	var rootPubHex string
	for _, subject := range nodeIDs {
		res := results[subject]
		h := res.RootPubHex
		if h == "" {
			return "", fmt.Errorf("missing root pub from %s", subject)
		}
		if rootPubHex == "" {
			rootPubHex = h
		} else if h != rootPubHex {
			return "", errors.New("root pub mismatch between nodes")
		}
	}

	// 将本次 keygen 的元信息持久化到本地 JSON 文件：keyMetaDir/{walletID}.json
	if err := os.MkdirAll(keyMetaDir, 0o700); err != nil {
		return "", fmt.Errorf("create key meta dir failed: %w", err)
	}
	metaPath := filepath.Join(keyMetaDir, walletID+".json")
	indexByNodeID := make(map[string]int, len(nodeIDs))
	for i, id := range nodeIDs {
		indexByNodeID[id] = i
	}
	metaObj := KeyMeta{
		WalletID:      walletID,
		KeyID:         keyID,
		RootPubHex:    rootPubHex,
		NodeIDs:       nodeIDs,
		Threshold:     threshold,
		IndexByNodeID: indexByNodeID,
	}
	metaData, err := json.Marshal(&metaObj)
	if err != nil {
		return "", fmt.Errorf("marshal key meta failed: %w", err)
	}
	if err := os.WriteFile(metaPath, metaData, 0o600); err != nil {
		return "", fmt.Errorf("write key meta file failed: %w", err)
	}

	// 清理与本次任务相关的缓存（节点状态、临时公钥、任务元信息）
	for _, subject := range nodeIDs {
		nodeCacheKey := utils.FNV1a64(utils.AddStr(subject, taskID))
		_ = keyCache.Del(nodeCacheKey)
		tempPubKey := utils.FNV1a64(utils.AddStr(subject, ":", taskID, ":keygen:tempPublicKey"))
		_ = keyCache.Del(tempPubKey)
	}
	_ = keyCache.Del(metaKey)

	mpcLogf("CreateMPCKeyTask: taskID=%s caches cleared\n", taskID)
	mpcLogf("CreateMPCKeyTask: taskID=%s all nodes done, keyID=%s\n", taskID, keyID)
	return keyID, nil
}

// handleTempPublicKey 节点上传 ECDH 临时公钥到服务端
func handleTempPublicKey(ctx context.Context, connCtx *node.ConnectionContext, body []byte) (interface{}, error) {
	request := &dto.CliMPCTempPublicKeyReq{}
	if err := utils.JsonUnmarshal(body, &request); err != nil {
		return nil, err
	}
	if request.TaskID == "" {
		return nil, errors.New("taskID is nil")
	}
	if request.PublicKey == "" {
		return nil, errors.New("public key is nil")
	}
	pub, err := ecc.LoadECDHPublicKeyFromBase64(request.PublicKey)
	if err != nil {
		return nil, err
	}
	subject := connCtx.GetUserIDString()
	cacheKey := utils.FNV1a64(utils.AddStr(subject, ":", request.TaskID, ":", request.Module, ":tempPublicKey"))
	// 缓存原始 bytes，便于后续直接使用
	if err := keyCache.Put(cacheKey, pub.Bytes(), 120); err != nil { // 120秒有效
		return nil, err
	}
	// event-driven collector notify (if any)
	submitPubkeyToCollector(request.Module, request.TaskID, subject, pub.Bytes())
	return &dto.CliMPCTempPublicKeyRes{Success: true}, nil
}

// handleMpcKeygenResult 节点上报 MPC keygen 结果（SaveData base64），服务端写入缓存供 CreateMPCKeyTask 轮询收齐后落盘。
func handleMpcKeygenResult(ctx context.Context, connCtx *node.ConnectionContext, body []byte) (interface{}, error) {
	req := &dto.CliMPCKeygenResultReq{}
	if err := utils.JsonUnmarshal(body, req); err != nil {
		return &dto.CliMPCKeygenResultRes{OK: false, Err: err.Error()}, nil
	}
	subject := connCtx.GetUserIDString()
	cacheKey := utils.FNV1a64(utils.AddStr(subject, req.TaskID))
	value, ok, err := keyCache.Get(cacheKey, nil)
	if err != nil || !ok || value == nil {
		return &dto.CliMPCKeygenResultRes{OK: false, Err: "task not found or expired"}, nil
	}
	nodeRes := value.(*MpcKeygenNodeResult)
	if nodeRes.Status == 40 {
		return &dto.CliMPCKeygenResultRes{OK: true}, nil
	}
	nodeRes.Status = 40
	nodeRes.KeyID = req.KeyID
	nodeRes.RootPubHex = req.RootPubHex
	nodeRes.Err = truncateErr(req.Err, 256)
	if err := keyCache.Put(cacheKey, nodeRes, 300); err != nil {
		return &dto.CliMPCKeygenResultRes{OK: false, Err: err.Error()}, nil
	}
	// event-driven collector notify (if any)
	submitKeygenResultToCollector(req.TaskID, subject, nodeRes)
	mpcLogf("CreateMPCKeyTask: node reported result taskID=%s node=%s status=40 keyID=%s err=%s\n",
		req.TaskID, subject, req.KeyID, truncateErr(req.Err, 64))
	return &dto.CliMPCKeygenResultRes{OK: true}, nil
}

// handleMpcKeygenMsg 节点发出的 TSS 协议消息；节点已按目标加密，服务端只转发不解密。
func handleMpcKeygenMsg(ctx context.Context, connCtx *node.ConnectionContext, body []byte) (interface{}, error) {
	req := &dto.CliMPCEncryptData{}
	if err := utils.JsonUnmarshal(body, req); err != nil {
		mpcLogf("handleMpcKeygenMsg: json unmarshal error: %v\n", err)
		return nil, err
	}

	mpcLogf("handleMpcKeygenMsg: taskID=%s subject=%s (forward only)\n", req.TaskID, req.Subject)

	if err := server.GetConnManager().SendToSubject(req.Subject, "mpcKeygenMsg", req); err != nil {
		mpcLogf("handleMpcKeygenMsg: push to %s FAILED: %v (taskID=%s)\n", req.Subject, err, req.TaskID)
		return nil, err
	}
	mpcLogf("handleMpcKeygenMsg: done taskID=%s subject=%s\n", req.TaskID, req.Subject)
	return &dto.CliMPCResultRes{OK: true}, nil
}
