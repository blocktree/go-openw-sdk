package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/blocktree/go-openw-sdk/v2/mpc"
	"github.com/blocktree/go-openw-sdk/v2/openwsdk/dto"
	"github.com/bnb-chain/tss-lib/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/tss"
	ecc "github.com/godaddy-x/eccrypto"
	"github.com/godaddy-x/freego/utils"
	"github.com/godaddy-x/freego/utils/sdk"
)

// RunKeygenNodeReal 是实际执行 KeyGen 的函数（由 HandleMpcKeygenStart 异步调用）
func RunKeygenNodeReal(taskID string, nodeIDs []string, myNodeID string, threshold int, wsClient *sdk.SocketSDK) (saveData keygen.LocalPartySaveData, keyID string, err error) {
	sortedIDs := mpc.PartyIDs(nodeIDs)
	myIndex := -1
	for i := range sortedIDs {
		if sortedIDs[i].GetId() == myNodeID {
			myIndex = i
			break
		}
	}
	if myIndex < 0 {
		return keygen.LocalPartySaveData{}, "", errors.New("myNodeID not in nodeIDs")
	}

	params := mpc.Parameters(sortedIDs, myIndex, threshold)
	if params == nil {
		return keygen.LocalPartySaveData{}, "", errors.New("mpc: invalid parameters")
	}

	fmt.Printf("[mpc-keygen] node=%s task=%s: generating preparams...\n", myNodeID, taskID)
	preParams, err := keygen.GeneratePreParams(90*time.Second, 2)
	if err != nil {
		return keygen.LocalPartySaveData{}, "", fmt.Errorf("preparams: %w", err)
	}
	fmt.Printf("[mpc-keygen] node=%s task=%s: preparams generated\n", myNodeID, taskID)

	outCh := make(chan tss.Message, 8)
	endCh := make(chan keygen.LocalPartySaveData, 1)
	errCh := make(chan error, 4)

	party := keygen.NewLocalParty(params, outCh, endCh, *preParams)

	// >>> 关键：获取已注册的 session 并补全 router <<<
	session := getKeygenSession(taskID, myNodeID)
	if session == nil {
		return keygen.LocalPartySaveData{}, "", errors.New("session disappeared during keygen")
	}

	// 填充 router 的缺失字段
	session.router.sortedIDs = sortedIDs
	session.router.myIndex = myIndex
	session.router.party = party
	session.router.wsClient = wsClient

	// 启动 outCh 消费协程
	go func() {
		for msg := range outCh {
			if sendErr := session.router.Send(myIndex, msg); sendErr != nil {
				select {
				case errCh <- sendErr:
				default:
				}
			}
		}
	}()

	if startErr := party.Start(); startErr != nil {
		return keygen.LocalPartySaveData{}, "", startErr
	}

	fmt.Printf("[mpc-keygen] node=%s task=%s: party started, waiting for messages and result\n", myNodeID, taskID)

	keygenTimeout := 10 * time.Minute
	deadline := time.After(keygenTimeout)
	for {
		select {
		case e := <-errCh:
			return keygen.LocalPartySaveData{}, "", e
		case save := <-endCh:
			if save.ECDSAPub != nil {
				keyID = mpc.KeyIDFromSaveData(save.ECDSAPub.X(), save.ECDSAPub.Y())
			}
			return save, keyID, nil
		case <-deadline:
			return keygen.LocalPartySaveData{}, "", errors.New("mpc keygen timeout")
		}
	}
}

// SubmitKeygenResult 将本节点 keygen 得到的 KeyID 上报服务端（不再上传 SaveData，仅用于一致性与状态判断）。
func SubmitKeygenResult(wsClient *sdk.SocketSDK, taskID, nodeID, keyID string) error {
	req := &dto.CliMPCKeygenResultReq{
		TaskID: taskID,
		NodeID: nodeID,
		KeyID:  keyID,
	}
	var res dto.CliMPCKeygenResultRes
	if err := wsClient.SendWebSocketMessage("/ws/mpcKeygenResult", req, &res, true, true, 30); err != nil {
		return err
	}
	if !res.OK {
		return errors.New("server rejected result: " + res.Err)
	}
	return nil
}

// submitKeygenResultErr 仅上报错误信息，供 keygen 失败时让服务端结束轮询。
// errMsg 会截断至 maxErrMsgLen，避免把大段 JSON/响应体写入服务端 cache 污染日志。
const maxErrMsgLen = 256

func submitKeygenResultErr(wsClient *sdk.SocketSDK, taskID, nodeID, errMsg string) error {
	if len(errMsg) > maxErrMsgLen {
		errMsg = errMsg[:maxErrMsgLen] + "..."
	}
	req := &dto.CliMPCKeygenResultReq{
		TaskID: taskID,
		NodeID: nodeID,
		Err:    errMsg,
	}
	var res dto.CliMPCKeygenResultRes
	_ = wsClient.SendWebSocketMessage("/ws/mpcKeygenResult", req, &res, true, true, 30)
	return nil
}

// HandleMpcKeygenStart 处理服务端下发的 mpcKeygenStart Push：
// 提前注册会话以接收早期 TSS 消息，再异步执行耗时的 KeyGen。
func HandleMpcKeygenStart(wsClient *sdk.SocketSDK, myNodeID, router string, body []byte) error {
	if len(body) == 0 {
		return nil
	}
	var decrypt dto.CliMPCEncryptData
	if err := utils.JsonUnmarshal(body, &decrypt); err != nil {
		return err
	}
	prk, err := getTempPrivateKey("keygen", myNodeID, decrypt.TaskID)
	if err != nil {
		return err
	}
	if prk == nil {
		return errors.New("temp prk is nil")
	}
	msg, err := ecc.Decrypt(prk, utils.Base64Decode(decrypt.Data), utils.Str2Bytes(utils.AddStr(decrypt.TaskID, "|", myNodeID, "|mpcKeygenStart")), nil)
	if err != nil {
		return err
	}
	var start dto.CliMPCKeygenStartRes
	if err := utils.JsonUnmarshal(msg, &start); err != nil {
		return err
	}
	if start.ExpiredTime > 0 && start.ExpiredTime < utils.UnixSecond() {
		return errors.New("mpc keygen task expired")
	}

	fmt.Printf("[mpc-keygen] node=%s task=%s start, threshold=%d, nodes=%v\n",
		myNodeID, start.TaskID, start.Threshold, start.NodeIDs)

	sortedIDs := mpc.PartyIDs(start.NodeIDs)
	myIndex := -1
	for i := range sortedIDs {
		if sortedIDs[i].GetId() == myNodeID {
			myIndex = i
			break
		}
	}
	if myIndex < 0 {
		return errors.New("mpc keygen task myIndex invalid")
	}

	// === 第一步：立即注册会话（即使 party 还没创建）===
	recvCh := make(chan recvItem, 512)
	errCh := make(chan error, 4)

	// 创建一个占位 router（party 暂为 nil）
	routerStub := &wsKeygenRouter{
		taskID:    start.TaskID,
		subject:   myNodeID,
		sortedIDs: sortedIDs,
		myIndex:   myIndex,
		wsClient:  wsClient,
		// sortedIDs 和 myIndex 稍后在 RunKeygenNodeReal 中填充
		// party 先为 nil，delivery 协程会跳过处理直到 party 就绪
	}

	session := &keygenSession{
		router: routerStub,
		recvCh: recvCh,
		errCh:  errCh,
	}
	registerKeygenSession(start.TaskID, myNodeID, session)

	// 启动 delivery 协程（可安全处理 party == nil 的情况）
	go runKeygenDelivery(session)

	// === 第二步：异步执行真正的 KeyGen（含 PreParams 生成）===
	go func() {
		defer func() {
			session.close()
			unregisterKeygenSession(start.TaskID, myNodeID)
			keygenTempPrk := utils.FNV1a64(utils.AddStr(myNodeID, ":", start.TaskID, ":keygen:tempPrivateKey"))
			_ = keyCache.Del(keygenTempPrk)
		}()

		saveData, keyID, err := RunKeygenNodeReal(start.TaskID, start.NodeIDs, myNodeID, start.Threshold, wsClient)
		if err != nil {
			fmt.Printf("[mpc-keygen] node=%s task=%s failed: %v\n", myNodeID, start.TaskID, err)
			_ = submitKeygenResultErr(wsClient, start.TaskID, myNodeID, err.Error())
			return
		}

		fmt.Printf("[mpc-keygen] node=%s task=%s succeeded, keyID=%s, saving local share and submitting result\n",
			myNodeID, start.TaskID, keyID)

		// 节点本地持久化自己的份额（keyfile）
		baseDir := fmt.Sprintf("keys")
		store := mpc.NewFileKeyStore(baseDir)
		if err := store.Save(keyID, myNodeID, saveData); err != nil {
			fmt.Printf("[mpc-keygen] node=%s task=%s save local share failed: %v\n", myNodeID, start.TaskID, err)
			_ = submitKeygenResultErr(wsClient, start.TaskID, myNodeID, "save local share failed: "+err.Error())
			return
		}

		if err := SubmitKeygenResult(wsClient, start.TaskID, myNodeID, keyID); err != nil {
			fmt.Printf("[mpc-keygen] node=%s task=%s submit result failed: %v\n", myNodeID, start.TaskID, err)
			_ = submitKeygenResultErr(wsClient, start.TaskID, myNodeID, "submit result failed: "+err.Error())
		}
	}()

	return nil
}

// wsKeygenRouter 节点侧基于 WebSocket 的 MessageRouter：Send 通过 POST /ws/mpcKeygenMsg 发给服务端中继，Receive 由 mpcKeygenMsg Push 回调时调用。
type wsKeygenRouter struct {
	taskID    string
	myIndex   int
	subject   string
	sortedIDs tss.SortedPartyIDs
	party     tss.Party
	wsClient  *sdk.SocketSDK
}

// Send 将本节点 party 产生的消息编码后 POST 到服务端，由服务端转发给其他节点。
func (r *wsKeygenRouter) Send(fromIndex int, msg tss.Message) error {
	wireBytes, _, err := msg.WireBytes()
	if err != nil {
		return err
	}
	fmt.Printf("[mpc-keygen] Send: task=%s fromIndex=%d isBroadcast=%v len=%d\n",
		r.taskID, fromIndex, msg.IsBroadcast(), len(wireBytes))

	var toNodeIDs []string
	if !msg.IsBroadcast() {
		for _, pid := range msg.GetTo() {
			toNodeIDs = append(toNodeIDs, pid.GetId())
		}
		fmt.Printf("[mpc-keygen] Send: task=%s fromIndex=%d toNodeIDs=%v\n",
			r.taskID, fromIndex, toNodeIDs)
	}
	req := &dto.CliMPCKeygenMsgReq{
		TaskID:          r.taskID,
		WireBytesBase64: base64.StdEncoding.EncodeToString(wireBytes),
		FromIndex:       fromIndex,
		IsBroadcast:     msg.IsBroadcast(),
		ToNodeIDs:       toNodeIDs,
	}
	var res map[string]interface{}
	if err := r.wsClient.SendWebSocketMessage("/ws/mpcKeygenMsg", req, &res, true, true, 60); err != nil {
		fmt.Printf("[mpc-keygen] Send: task=%s fromIndex=%d rpc error=%v\n", r.taskID, fromIndex, err)
		return err
	}
	return nil
}

// Receive 在收到服务端转发的 mpcKeygenMsg 时调用，将消息解析后投递给本节点 party.Update。
// 注意：Update 必须在单 goroutine 内串行调用，不能并发（tss-lib 非并发安全），故实际投递由 delivery 协程完成。
func (r *wsKeygenRouter) Receive(toIndex int, wireBytes []byte, fromIndex int, isBroadcast bool) error {
	if r.party == nil || fromIndex < 0 || fromIndex >= len(r.sortedIDs) {
		return nil
	}
	// 不应收到己方消息（协议只处理其他方）；若因服务端/SDK 回显收到则忽略，否则会破坏协议状态
	if fromIndex == r.myIndex {
		return nil
	}
	fromPartyID := r.sortedIDs[fromIndex]
	parsed, err := tss.ParseWireMessage(wireBytes, fromPartyID, isBroadcast)
	if err != nil {
		fmt.Printf("[mpc-keygen] Receive: task=%s parse error fromIndex=%d: %v\n",
			r.taskID, fromIndex, err)
		return err
	}
	_, err = r.party.Update(parsed)
	if err != nil {
		if err.Error() != "Error is nil" {
			fmt.Printf("[mpc-keygen] Receive: task=%s Update error fromIndex=%d: %v\n",
				r.taskID, fromIndex, err)
		}
	}
	return err
}

// recvItem 供 delivery 协程串行投递的一条消息。
type recvItem struct {
	WireBytes   []byte
	FromIndex   int
	IsBroadcast bool
}

// keygenSession 当前节点的一次 keygen 会话，供 mpcKeygenMsg Push 查找并投递消息。
// 投递通过 recvCh 串行化，避免多路 Push 并发调用 party.Update（tss-lib 非并发安全）。
type keygenSession struct {
	router    *wsKeygenRouter
	recvCh    chan recvItem
	errCh     chan<- error
	recvCount uint32 // 已投递的 Receive 条数，用于诊断协议是否在推进
	mu        sync.Mutex
	closed    bool
}

// enqueue 将一条消息放入投递队列；若会话已关闭或队列满则返回 false（持锁时仅非阻塞 send，避免死锁）。
func (s *keygenSession) enqueue(item recvItem) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return false
	}
	select {
	case s.recvCh <- item:
		return true
	default:
		fmt.Printf("[mpc-keygen] Deliver: recvCh full, dropping message fromIndex=%d (task=%s)\n",
			item.FromIndex, s.router.taskID)
		return false
	}
}

func (s *keygenSession) getRecvCount() uint32 { return atomic.LoadUint32(&s.recvCount) }

// close 关闭投递队列并标记会话结束，delivery 协程会随之退出。
func (s *keygenSession) close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	close(s.recvCh)
	s.mu.Unlock()
}

func runKeygenDelivery(s *keygenSession) {
	fmt.Printf("[mpc-keygen] task=%s myIndex=%d delivery goroutine started\n", s.router.taskID, s.router.myIndex)

	// 新增：缓存早期消息
	var earlyMsgs []recvItem

	for item := range s.recvCh {
		if item.FromIndex == s.router.myIndex {
			continue
		}

		// 如果 party 已就绪，直接处理
		if s.router.party != nil {
			// 先处理缓存的消息（按顺序）
			for _, early := range earlyMsgs {
				processMessage(s, early)
			}
			earlyMsgs = nil // 清空

			// 再处理当前消息
			processMessage(s, item)
			continue
		}

		// party 未就绪：缓存消息（最多缓存 512 条，防内存爆炸）
		if len(earlyMsgs) < 512 {
			earlyMsgs = append(earlyMsgs, item)
			fmt.Printf("[mpc-keygen] task=%s cached early msg fromIndex=%d (total=%d)\n",
				s.router.taskID, item.FromIndex, len(earlyMsgs))
		} else {
			fmt.Printf("[mpc-keygen] task=%s dropped early msg (buffer full) fromIndex=%d\n",
				s.router.taskID, item.FromIndex)
		}
	}
}

// 抽取消息处理逻辑
func processMessage(s *keygenSession, item recvItem) {
	fmt.Printf("[mpc-keygen] task=%s myIndex=%d before Update fromIndex=%d\n",
		s.router.taskID, s.router.myIndex, item.FromIndex)
	err := s.router.Receive(s.router.myIndex, item.WireBytes, item.FromIndex, item.IsBroadcast)
	fmt.Printf("[mpc-keygen] task=%s myIndex=%d after Update fromIndex=%d err=%v\n",
		s.router.taskID, s.router.myIndex, item.FromIndex, err)
	c := atomic.AddUint32(&s.recvCount, 1)
	if c <= 10 || c%20 == 0 {
		fmt.Printf("[mpc-keygen] task=%s myIndex=%d recvCount=%d fromIndex=%d\n",
			s.router.taskID, s.router.myIndex, c, item.FromIndex)
	}
	if err != nil && err.Error() != "Error is nil" {
		select {
		case s.errCh <- err:
		default:
		}
	}
}

var (
	keygenSessions   = make(map[string]*keygenSession)
	keygenSessionsMu sync.RWMutex
)

func keygenSessionKey(taskID, nodeID string) string {
	return taskID + "|" + nodeID
}

// registerKeygenSession 注册 (taskID,nodeID) 对应的 keygen 会话（允许单进程多节点）。
func registerKeygenSession(taskID, nodeID string, s *keygenSession) {
	keygenSessionsMu.Lock()
	defer keygenSessionsMu.Unlock()
	keygenSessions[keygenSessionKey(taskID, nodeID)] = s
}

// unregisterKeygenSession  keygen 结束后移除指定 (taskID,nodeID) 的会话。
func unregisterKeygenSession(taskID, nodeID string) {
	keygenSessionsMu.Lock()
	defer keygenSessionsMu.Unlock()
	delete(keygenSessions, keygenSessionKey(taskID, nodeID))
}

// getKeygenSession 由 mpcKeygenMsg 回调根据 (taskID,nodeID) 取会话并投递消息。
func getKeygenSession(taskID, nodeID string) *keygenSession {
	keygenSessionsMu.RLock()
	defer keygenSessionsMu.RUnlock()
	return keygenSessions[keygenSessionKey(taskID, nodeID)]
}

// DeliverMpcKeygenMsg 由 main 的 Push 回调调用：根据 body 中的 taskID 找到会话，将消息放入投递队列由 delivery 协程串行 Update。
func DeliverMpcKeygenMsg(wsClient *sdk.SocketSDK, myNodeID, router string, body []byte) error {
	if len(body) == 0 {
		return nil
	}
	var decrypt dto.CliMPCEncryptData
	if err := utils.JsonUnmarshal(body, &decrypt); err != nil {
		return err
	}
	prk, err := getTempPrivateKey("keygen", myNodeID, decrypt.TaskID)
	if err != nil {
		return err
	}
	if prk == nil {
		return errors.New("temp prk is nil")
	}
	msg, err := ecc.Decrypt(prk, utils.Base64Decode(decrypt.Data), utils.Str2Bytes(utils.AddStr(decrypt.TaskID, "|", myNodeID, "|mpcKeygenMsg")), nil)
	if err != nil {
		return err
	}
	var res dto.CliMPCKeygenMsgRes
	if err := utils.JsonUnmarshal(msg, &res); err != nil {
		fmt.Println("[mpc-keygen] Deliver: json error =", err)
		return err
	}
	// 单进程多节点时，需要用 (taskID,myNodeID) 精确找到当前节点的会话，避免被其他节点覆盖。
	s := getKeygenSession(res.TaskID, myNodeID)
	if s == nil || s.router == nil {
		fmt.Println("[mpc-keygen] Deliver: no session for task", res.TaskID)
		return nil
	}

	fmt.Printf("[mpc-keygen] Deliver: accepted task=%s myIndex=%d fromIndex=%d routerSubject=%s currentSubject=%s\n",
		res.TaskID, s.router.myIndex, res.FromIndex, s.router.subject, myNodeID)

	// 己方消息不入队（多为 POST 响应被当 Push 回显），避免占满 recvCh 且协议不收己方
	if res.FromIndex == s.router.myIndex {
		fmt.Printf("[mpc-keygen] Deliver: dropped (own) task=%s myIndex=%d fromIndex=%d\n",
			res.TaskID, s.router.myIndex, res.FromIndex)
		return nil
	}
	wireBytes, err := base64.StdEncoding.DecodeString(res.WireBytesBase64)
	if err != nil {
		fmt.Println("[mpc-keygen] Deliver: base64 error =", err)
		return err
	}
	item := recvItem{
		WireBytes:   wireBytes,
		FromIndex:   res.FromIndex,
		IsBroadcast: res.IsBroadcast,
	}
	if !s.enqueue(item) {
		fmt.Println("[mpc-keygen] Deliver: session already closed for task", res.TaskID)
		return nil
	}
	// 首条入队时打日志，便于确认 Push 是否到达节点（首轮每节点应收 2 条）
	fmt.Printf("[mpc-keygen] Deliver: enqueued myIndex=%d fromIndex=%d task=%s\n", s.router.myIndex, res.FromIndex, res.TaskID)
	return nil
}
