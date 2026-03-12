package webapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/blocktree/go-openw-sdk/v2/mpc"
	"github.com/blocktree/go-openw-sdk/v2/openwsdk/dto"
	ecc "github.com/godaddy-x/eccrypto"
	"github.com/godaddy-x/freego/node"
	"github.com/godaddy-x/freego/utils"
)

// CreateMPCSignTask 协调多节点完成一次 TSS 签名，返回 64 字节签名的 hex（R||S）。
// keyID 用于计算 walletID，并读取 keyMetaDir/{walletID}.json 获取参与节点列表与门限；msgHashHex 为 32 字节消息哈希的 hex。
func CreateMPCSignTask(walletID, msgHashHex string) (sigHex string, err error) {
	if server == nil {
		return "", errors.New("ws server not initialized")
	}

	// 1) 读取 key 元信息（节点列表、门限、index 映射）
	metaPath := filepath.Join(keyMetaDir, walletID+".json")
	raw, err := os.ReadFile(metaPath)
	if err != nil {
		return "", fmt.Errorf("read key meta failed: %w", err)
	}
	var keyMeta KeyMeta
	if err := json.Unmarshal(raw, &keyMeta); err != nil {
		return "", fmt.Errorf("unmarshal key meta failed: %w", err)
	}
	if keyMeta.KeyID == "" || len(keyMeta.NodeIDs) == 0 {
		return "", errors.New("invalid key meta")
	}

	keyID := keyMeta.KeyID

	// 2) 解析消息哈希
	_, err = mpc.MessageHashFromTxHash(msgHashHex)
	if err != nil {
		return "", fmt.Errorf("invalid msgHashHex: %w", err)
	}

	// 3) 全量节点提交临时公钥：要求 KeyMeta.NodeIDs 中的所有节点当前都在线，且全部参与签名
	nodes := server.GetConnManager().GetAllSubjectDevices()
	online := make(map[string]bool, len(nodes))
	for s := range nodes {
		online[s] = true
	}
	allNodeIDs := make([]string, 0, len(keyMeta.NodeIDs))
	for _, id := range keyMeta.NodeIDs {
		if !online[id] {
			return "", fmt.Errorf("node %s offline for sign", id)
		}
		allNodeIDs = append(allNodeIDs, id)
	}

	taskID := utils.GetUUID(true)
	expiredTime := utils.UnixSecond() + 120

	mpcLogf("CreateMPCSignTask: taskID=%s keyID=%s allNodes=%v threshold=%d (all nodes participate)\n",
		taskID, keyID, allNodeIDs, keyMeta.Threshold)

	// 4) ECDH 临时公钥交换（module = "sign"）
	// 先注册 collector，避免节点快速上报导致事件丢失；并在注册后从 cache 回放一次已存在的公钥。
	timeout := 35
	collector := NewPubkeyCollector(allNodeIDs)
	registerPubkeyCollector("sign", taskID, collector, time.Duration(timeout)*time.Second)
	defer unregisterPubkeyCollector("sign", taskID)

	for _, subject := range allNodeIDs {
		req := &dto.CliMPCTempPublicKeyReq{
			TaskID: taskID,
			Module: "sign",
		}
		if err := server.GetConnManager().SendToSubject(subject, "mpcTempPublicKey", req); err != nil {
			return "", err
		}
	}

	// cache replay：如果有节点在 collector 注册前已上报（或重试上报），这里补一遍 Submit，确保不会丢事件
	for _, subject := range allNodeIDs {
		cacheKey := utils.FNV1a64(utils.AddStr(subject, ":", taskID, ":sign:tempPublicKey"))
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

	signMeta := &MpcKeygenTaskMeta{
		TaskID:      taskID,
		AllNodeIDs:  allNodeIDs,
		SignNodeIDs: nil, // 初始化后在收齐临时公钥后设为全量 allNodeIDs
		Threshold:   keyMeta.Threshold,
		ExpiredTime: expiredTime,
		PublicKey:   collector.GetPubkeys(),
	}

	// 在全量节点都上报临时公钥后，强制所有节点都参与签名（SignNodeIDs = AllNodeIDs）
	signNodeIDs := allNodeIDs
	signMeta.SignNodeIDs = signNodeIDs

	// 将 signMeta 存入 cache 供 handleMpcSignMsg 使用
	metaKey := utils.FNV1a64("mpcSignMeta:" + taskID)
	if err := keyCache.Put(metaKey, signMeta, 600); err != nil {
		return "", err
	}

	// 5) 下发 mpcSignStart（加密）
	startPayload := &dto.CliMPCSignStartRes{
		TaskID:        taskID,
		KeyID:         keyID,
		AllNodeIDs:    allNodeIDs,
		SignNodeIDs:   signNodeIDs,
		Threshold:     keyMeta.Threshold,
		MsgHashHex:    msgHashHex,
		ExpiredTime:   expiredTime,
		PublicKeyPair: make([]dto.CliMPCPublicKeyPair, 0, len(allNodeIDs)),
	}

	// PublicKeyPair 填写全量节点的临时公钥，便于参与节点按任意目标加密
	for _, v := range allNodeIDs {
		startPayload.PublicKeyPair = append(startPayload.PublicKeyPair, dto.CliMPCPublicKeyPair{
			Subject:   v,
			PublicKey: utils.Base64Encode(signMeta.PublicKey[v]),
		})
	}

	for _, subject := range signNodeIDs {
		data, err := utils.JsonMarshal(startPayload)
		if err != nil {
			return "", err
		}
		encrypt, err := ecc.Encrypt(nil, signMeta.PublicKey[subject], data, utils.Str2Bytes(utils.AddStr(taskID, "|", subject, "|mpcSignStart")))
		if err != nil {
			return "", err
		}
		if err := server.GetConnManager().SendToSubject(subject, "mpcSignStart", &dto.CliMPCEncryptData{
			TaskID: taskID,
			Data:   utils.Base64Encode(encrypt),
		}); err != nil {
			return "", err
		}
	}

	// 6) 等待签名结果（事件驱动，无需轮询 cache）
	resultCollector := NewSignResultCollector(signNodeIDs)
	registerSignResultCollector(taskID, resultCollector, time.Duration(timeout)*time.Minute)
	defer unregisterSignResultCollector(taskID)

	// cache replay：如果有节点在 collector 注册前已上报（或重试上报），这里补一遍 Submit，确保不会丢事件
	for _, subject := range signNodeIDs {
		cacheKey := utils.FNV1a64(utils.AddStr("sign:", subject, ":", taskID))
		v, ok, _ := keyCache.Get(cacheKey, nil)
		if ok && v != nil {
			if res, ok := v.(*dto.CliMPCSignResultReq); ok && res != nil {
				resultCollector.Submit(subject, res)
			}
		}
	}

	waitResCtx, cancelRes := context.WithTimeout(context.Background(), time.Duration(timeout-5)*time.Second)
	defer cancelRes()
	if err := resultCollector.Wait(waitResCtx); err != nil {
		return "", fmt.Errorf("timeout waiting for MPC sign result: %w", err)
	}

	results := resultCollector.GetResults()
	var sigHexFirst string
	for _, subject := range signNodeIDs {
		res := results[subject]
		if res == nil {
			return "", errors.New("missing sign result from " + subject)
		}
		if res.Err != "" {
			return "", fmt.Errorf("node %s sign failed: %s", res.NodeID, res.Err)
		}
		if sigHexFirst == "" {
			sigHexFirst = res.SignatureHex
		} else if res.SignatureHex != sigHexFirst {
			return "", errors.New("signature mismatch between nodes")
		}
	}
	if sigHexFirst == "" {
		return "", errors.New("empty signature from nodes")
	}

	// 清理与本次签名任务相关的缓存（签名结果、临时公钥、任务元信息）
	for _, subject := range signNodeIDs {
		resKey := utils.FNV1a64(utils.AddStr("sign:", subject, ":", taskID))
		_ = keyCache.Del(resKey)
		tempPubKey := utils.FNV1a64(utils.AddStr(subject, ":", taskID, ":sign:tempPublicKey"))
		_ = keyCache.Del(tempPubKey)
	}
	metaKey = utils.FNV1a64("mpcSignMeta:" + taskID)
	_ = keyCache.Del(metaKey)

	mpcLogf("CreateMPCSignTask: taskID=%s caches cleared\n", taskID)

	return sigHexFirst, nil
}

// handleMpcSignResult 节点上报 MPC 签名结果，服务端仅用于汇总与一致性校验。
func handleMpcSignResult(ctx context.Context, connCtx *node.ConnectionContext, body []byte) (interface{}, error) {
	req := &dto.CliMPCSignResultReq{}
	if err := utils.JsonUnmarshal(body, req); err != nil {
		return &dto.CliMPCSignResultRes{OK: false, Err: err.Error()}, nil
	}
	subject := connCtx.GetUserIDString()
	if req.NodeID == "" {
		req.NodeID = subject
	}
	cacheKey := utils.FNV1a64(utils.AddStr("sign:", subject, ":", req.TaskID))
	if err := keyCache.Put(cacheKey, req, 300); err != nil {
		return &dto.CliMPCSignResultRes{OK: false, Err: err.Error()}, nil
	}
	// event-driven collector notify (if any)
	submitSignResultToCollector(req.TaskID, subject, req)
	return &dto.CliMPCSignResultRes{OK: true}, nil
}

// handleMpcSignMsg 节点发出的 TSS 签名协议消息，服务端转发给其他参与方（广播或单播）。
func handleMpcSignMsg(ctx context.Context, connCtx *node.ConnectionContext, body []byte) (interface{}, error) {
	req := &dto.CliMPCEncryptData{}
	if err := utils.JsonUnmarshal(body, req); err != nil {
		mpcLogf("handleMpcSignMsg: json unmarshal error: %v\n", err)
		return nil, err
	}

	mpcLogf("handleMpcSignMsg: taskID=%s subject=%s (forward only)\n", req.TaskID, req.Subject)

	if err := server.GetConnManager().SendToSubject(req.Subject, "mpcSignMsg", req); err != nil {
		mpcLogf("handleMpcSignMsg: push to %s FAILED: %v (taskID=%s fromSubject=%s)\n",
			req.TaskID, err, req.TaskID, req.Subject)
	}

	mpcLogf("handleMpcSignMsg: done for taskID=%s subject=%s\n", req.TaskID, req.Subject)
	return &dto.CliMPCResultRes{OK: true}, nil
}
