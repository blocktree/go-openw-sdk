package webapp

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/blocktree/go-openw-sdk/v2/mpc"
	"github.com/blocktree/go-openw-sdk/v2/openwsdk/dto"
	ecc "github.com/godaddy-x/eccrypto"
	"github.com/godaddy-x/freego/node"
	"github.com/godaddy-x/freego/utils"
)

// mpcKeygenLogMu 串行化 mpc-keygen 相关日志，避免 CreateMPCKeyTask 轮询与 handleMpcKeygenMsg 并发写 stdout 导致交错。
var mpcKeygenLogMu sync.Mutex

func mpcLogf(format string, args ...interface{}) {
	mpcKeygenLogMu.Lock()
	fmt.Printf("[mpc-keygen] "+format, args...)
	mpcKeygenLogMu.Unlock()
}

// MpcKeygenTaskMeta 按 taskID 存的 MPC keygen 任务元信息（NodeIDs、门限等），用于转发 TSS 消息时查表。
type MpcKeygenTaskMeta struct {
	TaskID      string
	NodeIDs     []string
	Threshold   int
	ExpiredTime int64
	PublicKey   map[string][]byte // subject -> temp ECDH public key (raw bytes)
}

// MpcKeygenNodeResult 按 (subject, taskID) 存的节点上报结果，Status 40 表示已上报 SaveData。
type MpcKeygenNodeResult struct {
	TaskID    string
	NodeID    string
	Status    int64  // 10=已下发 start，40=已上报结果
	KeyID     string // 仅用于各节点自报的一致性校验，服务端不落盘 SaveData
	PublicKey string // 节点临时公钥
	Err       string
}

func truncateErr(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// CreateMPCKeyTask 协调多节点完成一次 TSS keygen，轮询直到所有节点上报结果后返回 KeyID。
// 服务端只做「协调 + 校验」，不落盘任何 SaveData：节点各自持久化自己的 LocalPartySaveData。
// 流程：1) 取在线 subject 排序为 nodeIDs  2) 下发 mpcKeygenStart  3) 轮询 mpcKeygenResult 检查状态与 KeyID 一致性  4) 返回 KeyID。
func CreateMPCKeyTask() (keyID string, err error) {
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

	// 发送通知节点提交临时公钥
	for _, subject := range nodeIDs {
		req := &dto.CliMPCTempPublicKeyReq{
			TaskID: taskID,
		}
		if err := server.GetConnManager().SendToSubject(subject, "mpcTempPublicKey", req); err != nil {
			return "", err
		}
	}

	meta := &MpcKeygenTaskMeta{
		TaskID:      taskID,
		NodeIDs:     nodeIDs,
		Threshold:   threshold,
		ExpiredTime: expiredTime,
		PublicKey:   make(map[string][]byte, 5),
	}

	// 轮询检查多节点提交公钥状态
	keyMaxWait := time.After(10 * time.Second)
	keyTicker := time.NewTicker(300 * time.Millisecond)
	defer keyTicker.Stop()

	waitForAll := func() error {
		for {
			select {
			case <-keyMaxWait:
				return errors.New("timeout waiting for all nodes to submit public keys")
			case <-keyTicker.C:
				allReady := true
				for _, subject := range nodeIDs {
					cacheKey := utils.FNV1a64(utils.AddStr(subject, ":", taskID, ":tempPublicKey"))
					v, ok, _ := keyCache.Get(cacheKey, nil)
					if !ok || v == nil {
						allReady = false
						break
					}
					meta.PublicKey[subject] = v.([]byte)
				}
				if allReady {
					return nil
				}
			}
		}
	}

	if err := waitForAll(); err != nil {
		return "", err
	}

	metaKey := utils.FNV1a64("mpcMeta:" + taskID)
	if err := keyCache.Put(metaKey, meta, 600); err != nil {
		return "", err
	}

	startPayload := &dto.CliMPCKeygenStartRes{
		TaskID:      taskID,
		NodeIDs:     nodeIDs,
		Threshold:   threshold,
		ExpiredTime: expiredTime,
	}

	for _, subject := range nodeIDs {
		if err := server.GetConnManager().SendToSubject(subject, "mpcKeygenStart", startPayload); err != nil {
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

	maxWait := time.After(10 * time.Minute)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-maxWait:
			mpcLogf("CreateMPCKeyTask: timeout waiting for nodes, taskID=%s\n", taskID)
			return "", errors.New("timeout waiting for all nodes to submit MPC keygen result")
		case <-ticker.C:
			allDone := true
			var firstKeyID string
			// 汇总当前各节点状态，用于日志
			statusLine := ""
			for _, subject := range nodeIDs {
				cacheKey := utils.FNV1a64(utils.AddStr(subject, taskID))
				v, ok, _ := keyCache.Get(cacheKey, nil)
				if !ok || v == nil {
					allDone = false
					statusLine += subject + "=missing "
					continue
				}
				res := v.(*MpcKeygenNodeResult)
				statusLine += fmt.Sprintf("%s=status:%d ", subject, res.Status)
				if res.KeyID != "" {
					statusLine += "keyID=" + res.KeyID + " "
				}
				if res.Err != "" {
					statusLine += "err=" + truncateErr(res.Err, 64) + " "
				}
				if res.Status != 40 {
					allDone = false
				} else {
					if res.Err != "" {
						return "", errors.New("node " + res.NodeID + " keygen failed: " + res.Err)
					}
					if firstKeyID == "" {
						firstKeyID = res.KeyID
					} else if res.KeyID != firstKeyID {
						return "", errors.New("keyID mismatch between nodes")
					}
				}
			}
			mpcLogf("CreateMPCKeyTask: taskID=%s state: %s\n", taskID, statusLine)
			if !allDone {
				continue
			}
			keyID = firstKeyID
			if keyID == "" {
				return "", errors.New("keyID empty")
			}
			mpcLogf("CreateMPCKeyTask: taskID=%s all nodes done, keyID=%s\n", taskID, keyID)
			return keyID, nil
		}
	}
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
	_, err := ecc.LoadECDHPublicKeyFromBase64(request.PublicKey)
	if err != nil {
		return nil, err
	}
	rawPub, err := base64.StdEncoding.DecodeString(request.PublicKey)
	if err != nil {
		return nil, err
	}
	subject := connCtx.GetUserIDString()
	cacheKey := utils.FNV1a64(utils.AddStr(subject, ":", request.TaskID, ":tempPublicKey"))
	// 缓存原始 bytes，便于后续直接使用
	if err := keyCache.Put(cacheKey, rawPub, 20); err != nil { // 20秒有效
		return nil, err
	}
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
	//nodeRes.SaveDataBase64 = req.SaveDataBase64
	nodeRes.Err = truncateErr(req.Err, 256)
	if err := keyCache.Put(cacheKey, nodeRes, 300); err != nil {
		return &dto.CliMPCKeygenResultRes{OK: false, Err: err.Error()}, nil
	}
	mpcLogf("CreateMPCKeyTask: node reported result taskID=%s node=%s status=40 keyID=%s err=%s\n",
		req.TaskID, subject, req.KeyID, truncateErr(req.Err, 64))
	return &dto.CliMPCKeygenResultRes{OK: true}, nil
}

// handleMpcKeygenMsg 节点发出的 TSS 协议消息，服务端转发给其他参与方（广播或单播）。
func handleMpcKeygenMsg(ctx context.Context, connCtx *node.ConnectionContext, body []byte) (interface{}, error) {
	req := &dto.CliMPCKeygenMsgReq{}
	if err := utils.JsonUnmarshal(body, req); err != nil {
		mpcLogf("handleMpcKeygenMsg: json unmarshal error: %v\n", err)
		return nil, err
	}

	mpcLogf("handleMpcKeygenMsg: taskID=%s fromIndex=%d isBroadcast=%v toNodeIDs=%v\n",
		req.TaskID, req.FromIndex, req.IsBroadcast, req.ToNodeIDs)

	metaKey := utils.FNV1a64("mpcMeta:" + req.TaskID)
	value, ok, _ := keyCache.Get(metaKey, nil)
	if !ok || value == nil {
		mpcLogf("handleMpcKeygenMsg: task meta not found, taskID=%s\n", req.TaskID)
		return nil, errors.New("mpc task not found: " + req.TaskID)
	}
	meta := value.(*MpcKeygenTaskMeta)
	if meta.ExpiredTime < utils.UnixSecond() {
		mpcLogf("handleMpcKeygenMsg: task expired, taskID=%s\n", req.TaskID)
		return nil, errors.New("mpc task expired")
	}

	senderSubject := connCtx.GetUserIDString()
	mpcLogf("handleMpcKeygenMsg: sender subject=%s fromIndex=%d (taskID=%s)\n", senderSubject, req.FromIndex, req.TaskID)

	payload := &dto.CliMPCKeygenMsgRes{
		TaskID:          req.TaskID,
		WireBytesBase64: req.WireBytesBase64,
		FromIndex:       req.FromIndex,
		IsBroadcast:     req.IsBroadcast,
	}

	if req.IsBroadcast {
		var targets []string
		for i, nodeID := range meta.NodeIDs {
			if i == req.FromIndex {
				continue
			}
			targets = append(targets, nodeID)
			mpcLogf("handleMpcKeygenMsg: sending push mpcKeygenMsg -> %s (taskID=%s fromIndex=%d)\n",
				nodeID, req.TaskID, req.FromIndex)
			if err := server.GetConnManager().SendToSubject(nodeID, "mpcKeygenMsg", payload); err != nil {
				mpcLogf("handleMpcKeygenMsg: push to %s FAILED: %v (taskID=%s fromIndex=%d)\n",
					nodeID, err, req.TaskID, req.FromIndex)
			} else {
				mpcLogf("handleMpcKeygenMsg: push to %s OK (taskID=%s fromIndex=%d)\n",
					nodeID, req.TaskID, req.FromIndex)
			}
		}
		mpcLogf("handleMpcKeygenMsg: broadcast fromIndex=%d -> targets=%v (excluded self)\n", req.FromIndex, targets)
	} else {
		for _, nodeID := range req.ToNodeIDs {
			mpcLogf("handleMpcKeygenMsg: sending push mpcKeygenMsg -> %s (taskID=%s fromIndex=%d)\n",
				nodeID, req.TaskID, req.FromIndex)
			if err := server.GetConnManager().SendToSubject(nodeID, "mpcKeygenMsg", payload); err != nil {
				mpcLogf("handleMpcKeygenMsg: push to %s FAILED: %v (taskID=%s fromIndex=%d)\n",
					nodeID, err, req.TaskID, req.FromIndex)
			} else {
				mpcLogf("handleMpcKeygenMsg: push to %s OK (taskID=%s fromIndex=%d)\n",
					nodeID, req.TaskID, req.FromIndex)
			}
		}
	}

	mpcLogf("handleMpcKeygenMsg: done for taskID=%s\n", req.TaskID)
	return map[string]bool{"ok": true}, nil
}
