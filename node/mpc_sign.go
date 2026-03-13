package main

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/blocktree/go-openw-sdk/v2/mpc"
	"github.com/blocktree/go-openw-sdk/v2/mpc/alg_ecdsa"
	"github.com/blocktree/go-openw-sdk/v2/openwsdk/dto"
	"github.com/bnb-chain/tss-lib/common"
	"github.com/bnb-chain/tss-lib/ecdsa/signing"
	"github.com/bnb-chain/tss-lib/tss"
	ecc "github.com/godaddy-x/eccrypto"
	"github.com/godaddy-x/freego/utils"
	"github.com/godaddy-x/freego/utils/sdk"
)

// ============ 新增：sign 早期消息缓存（避免 mpcSignMsg 先于 session 注册到达） ============
var (
	earlySignMessages   = make(map[string]earlySignBucket) // key = taskID|myNodeID
	earlySignMessagesMu sync.Mutex
	maxEarlySignMsgs    = 512
	earlySignMsgTTL     = 2 * time.Minute
)

type earlySignBucket struct {
	items     []recvItem
	createdAt time.Time
}

func cleanupExpiredEarlySignMessagesLocked(now time.Time) {
	for k, b := range earlySignMessages {
		if now.Sub(b.createdAt) > earlySignMsgTTL {
			delete(earlySignMessages, k)
		}
	}
}

func logSignf(format string, args ...interface{}) {
	fmt.Printf("[mpc-sign] "+format, args...)
}

// RunSignNodeRealByAlg 按算法执行本节点的签名逻辑（由 HandleMpcSignStart 异步调用）。
// 当前仅实现 ECDSA（AlgECDSA），后续可根据 Algorithm 扩展 Ed25519 等。
func RunSignNodeRealByAlg(
	start dto.CliMPCSignStartRes,
	myNodeID string,
	msgHash *big.Int,
	wsClient *sdk.SocketSDK,
) (signatureHex string, err error) {
	switch mpc.Algorithm(start.Algorithm) {
	case mpc.AlgECDSA:
		return runSignNodeRealECDSA(start, myNodeID, msgHash, wsClient)
	default:
		return "", fmt.Errorf("unsupported MPC algorithm for sign on node: %s", mpc.Algorithm(start.Algorithm))
	}
}

// runSignNodeRealECDSA 为 secp256k1 ECDSA 的具体实现。
// 使用本地 keyfile (keyID-nodeID.json) 中的 LocalPartySaveData，通过 WS 路由与其他节点完成一次 TSS 签名。
func runSignNodeRealECDSA(
	start dto.CliMPCSignStartRes,
	myNodeID string,
	msgHash *big.Int,
	wsClient *sdk.SocketSDK,
) (signatureHex string, err error) {
	sortedIDs := alg_ecdsa.PartyIDs(start.AllNodeIDs)
	myIndex := -1
	for i := range sortedIDs {
		if sortedIDs[i].GetId() == myNodeID {
			myIndex = i
			break
		}
	}
	if myIndex < 0 {
		return "", errors.New("myNodeID not in nodeIDs")
	}

	params := alg_ecdsa.Parameters(sortedIDs, myIndex, start.Threshold)
	if params == nil {
		return "", errors.New("mpc: invalid sign parameters")
	}

	// 从本地 keystore 加载本节点的 SaveData
	store := alg_ecdsa.NewFileKeyStore("keys")
	saveData, err := store.Load(start.KeyID, myNodeID)
	if err != nil {
		return "", fmt.Errorf("load local SaveData failed: %w", err)
	}

	outCh := make(chan tss.Message, 8)
	endCh := make(chan common.SignatureData, 1)
	errCh := make(chan error, 4)

	party := signing.NewLocalParty(msgHash, params, saveData, outCh, endCh)

	// 获取已注册的 session 并补全 router
	session := getSignSession(start.TaskID, myNodeID)
	if session == nil {
		return "", errors.New("sign session disappeared during RunSignNodeReal")
	}

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
		return "", startErr
	}

	logSignf("node=%s task=%s: party started, waiting for messages and result\n", myNodeID, start.TaskID)

	signTimeout := 2 * time.Minute
	deadline := time.After(signTimeout)
	for {
		select {
		case e := <-errCh:
			return "", e
		case sigData := <-endCh:
			// 聚合签名数据为 64 字节 R||S hex
			r := new(big.Int).SetBytes(sigData.GetR())
			s := new(big.Int).SetBytes(sigData.GetS())
			sig := append(alg_ecdsa.Pad32(r.Bytes()), alg_ecdsa.Pad32(s.Bytes())...)
			return hex.EncodeToString(sig), nil
		case <-deadline:
			return "", errors.New("mpc sign timeout")
		}
	}
}

// HandleMpcSignStart 处理服务端下发的 mpcSignStart Push：
// 解密 payload -> CliMPCSignStartRes，注册 sign 会话并异步执行 RunSignNodeReal。
func HandleMpcSignStart(wsClient *sdk.SocketSDK, myNodeID, router string, body []byte) error {
	if len(body) == 0 {
		return nil
	}
	var decrypt dto.CliMPCEncryptData
	if err := utils.JsonUnmarshal(body, &decrypt); err != nil {
		return err
	}
	prk, err := getTempPrivateKey("sign", myNodeID, decrypt.TaskID)
	if err != nil {
		return err
	}
	if prk == nil {
		return errors.New("temp prk is nil")
	}
	msg, err := ecc.Decrypt(prk, utils.Base64Decode(decrypt.Data), utils.Str2Bytes(utils.AddStr(decrypt.TaskID, "|", myNodeID, "|mpcSignStart")), nil)
	if err != nil {
		return err
	}
	var start dto.CliMPCSignStartRes
	if err := utils.JsonUnmarshal(msg, &start); err != nil {
		return err
	}
	if start.ExpiredTime > 0 && start.ExpiredTime < utils.UnixSecond() {
		return errors.New("mpc sign task expired")
	}

	for _, v := range start.PublicKeyPair {
		cacheKey := utils.FNV1a64(utils.AddStr(v.Subject, ":", start.TaskID, ":sign:tempPublicKey"))
		if err := keyCache.Put(cacheKey, v.PublicKey, 600); err != nil {
			return errors.New("handleTempPublicKey put tempPrivateKey error: " + err.Error())
		}
	}

	logSignf("node=%s task=%s start, alg=%s keyID=%s threshold=%d, allNodes=%v signNodes=%v\n",
		myNodeID, start.TaskID, start.Algorithm, start.KeyID, start.Threshold, start.AllNodeIDs, start.SignNodeIDs)

	sortedIDs := alg_ecdsa.PartyIDs(start.AllNodeIDs)
	myIndex := -1
	for i := range sortedIDs {
		if sortedIDs[i].GetId() == myNodeID {
			myIndex = i
			break
		}
	}
	if myIndex < 0 {
		return errors.New("mpc sign task myIndex invalid")
	}

	// 解析消息哈希
	msgHash, err := alg_ecdsa.MessageHashFromTxHash(start.MsgHashHex)
	if err != nil {
		return fmt.Errorf("invalid msgHashHex: %w", err)
	}

	// 注册 sign 会话
	recvCh := make(chan recvItem, 512)
	errCh := make(chan error, 4)

	routerStub := &wsSignRouter{
		taskID:      start.TaskID,
		subject:     myNodeID,
		sortedIDs:   sortedIDs,
		myIndex:     myIndex,
		wsClient:    wsClient,
		allNodeIDs:  start.AllNodeIDs,
		signNodeIDs: start.SignNodeIDs,
	}

	session := &signSession{
		router: routerStub,
		recvCh: recvCh,
		errCh:  errCh,
	}
	// 先启动 delivery，再注册/回放，减少回放时塞满队列的概率
	go runSignDelivery(session)
	registerSignSession(start.TaskID, myNodeID, session)

	// 异步执行签名
	go func() {
		defer func() {
			session.close()
			unregisterSignSession(start.TaskID, myNodeID)
			signTempPrk := utils.FNV1a64(utils.AddStr(myNodeID, ":", start.TaskID, ":sign:tempPrivateKey"))
			_ = keyCache.Del(signTempPrk)
			for _, v := range start.AllNodeIDs {
				signTempPub := utils.FNV1a64(utils.AddStr(v, ":", start.TaskID, ":sign:tempPublicKey"))
				_ = keyCache.Del(signTempPub)
			}
		}()

		sigHex, err := RunSignNodeRealByAlg(start, myNodeID, msgHash, wsClient)
		nodeID := myNodeID
		if err != nil {
			logSignf("node=%s task=%s failed: %v\n", myNodeID, start.TaskID, err)
			req := &dto.CliMPCSignResultReq{
				TaskID: start.TaskID,
				NodeID: nodeID,
				KeyID:  start.KeyID,
				Err:    err.Error(),
			}
			var res dto.CliMPCSignResultRes
			_ = wsClient.SendWebSocketMessage("/ws/mpcSignResult", req, &res, true, true, 30)
			return
		}

		logSignf("node=%s task=%s succeeded, keyID=%s, signature=%s\n",
			myNodeID, start.TaskID, start.KeyID, sigHex)

		req := &dto.CliMPCSignResultReq{
			TaskID:       start.TaskID,
			NodeID:       nodeID,
			KeyID:        start.KeyID,
			SignatureHex: sigHex,
		}
		var res dto.CliMPCSignResultRes
		if err := wsClient.SendWebSocketMessage("/ws/mpcSignResult", req, &res, true, true, 30); err != nil {
			logSignf("node=%s task=%s submit sign result failed: %v\n", myNodeID, start.TaskID, err)
		}
	}()

	return nil
}

// wsSignRouter 节点侧基于 WebSocket 的 MessageRouter（签名版）
type wsSignRouter struct {
	taskID      string
	myIndex     int
	subject     string
	sortedIDs   tss.SortedPartyIDs
	allNodeIDs  []string
	signNodeIDs []string
	party       tss.Party
	wsClient    *sdk.SocketSDK
}

// Send 将本节点 party 产生的消息编码后 POST 到服务端，由服务端转发给其他节点。
func (r *wsSignRouter) Send(fromIndex int, msg tss.Message) error {
	var toNodeIDs []string
	if !msg.IsBroadcast() {
		for _, pid := range msg.GetTo() {
			toNodeIDs = append(toNodeIDs, pid.GetId())
		}
	} else {
		for _, v := range r.signNodeIDs {
			if v == r.subject { // 跳过自身
				continue
			}
			toNodeIDs = append(toNodeIDs, v)
		}
	}
	logSignf("Send: task=%s fromIndex=%d toNodeIDs=%v\n",
		r.taskID, fromIndex, toNodeIDs)

	wireBytes, _, err := msg.WireBytes()
	if err != nil {
		return err
	}
	logSignf("Send: node=%s task=%s fromIndex=%d isBroadcast=%v len=%d\n",
		r.subject, r.taskID, fromIndex, msg.IsBroadcast(), len(wireBytes))

	// 这里开始发送给服务端转发
	for _, v := range toNodeIDs {

		payload := &dto.CliMPCSignMsgRes{
			TaskID:          r.taskID,
			WireBytesBase64: base64.StdEncoding.EncodeToString(wireBytes),
			FromIndex:       fromIndex,
			IsBroadcast:     msg.IsBroadcast(),
		}
		data, err := utils.JsonMarshal(payload)
		if err != nil {
			return err
		}
		publicKey, err := getTempPublicKey("sign", v, r.taskID)
		if err != nil {
			return err
		}
		encrypt, err := ecc.Encrypt(nil, publicKey, data, utils.Str2Bytes(utils.AddStr(r.taskID, "|", v, "|mpcSignMsg")))
		if err != nil {
			return err
		}
		if err := r.wsClient.SendWebSocketMessage("/ws/mpcSignMsg", &dto.CliMPCEncryptData{Data: utils.Base64Encode(encrypt), TaskID: r.taskID, Subject: v}, &dto.CliMPCResultRes{}, true, true, 60); err != nil {
			logSignf("Send: task=%s fromIndex=%d rpc error=%v\n", r.taskID, fromIndex, err)
			return err
		}
	}
	return nil
}

// Receive 在收到服务端转发的 mpcSignMsg 时调用，将消息解析后投递给本节点 party.Update。
func (r *wsSignRouter) Receive(toIndex int, wireBytes []byte, fromIndex int, isBroadcast bool) error {
	if r.party == nil || fromIndex < 0 || fromIndex >= len(r.sortedIDs) {
		return nil
	}
	if fromIndex == r.myIndex {
		return nil
	}
	fromPartyID := r.sortedIDs[fromIndex]
	parsed, err := tss.ParseWireMessage(wireBytes, fromPartyID, isBroadcast)
	if err != nil {
		logSignf("Receive: task=%s parse error fromIndex=%d: %v\n",
			r.taskID, fromIndex, err)
		return err
	}
	_, err = r.party.Update(parsed)
	if err != nil {
		if err.Error() != "Error is nil" {
			logSignf("Receive: task=%s Update error fromIndex=%d: %v\n",
				r.taskID, fromIndex, err)
		}
	}
	return err
}

// signSession 当前节点的一次 sign 会话，供 mpcSignMsg Push 查找并投递消息。
type signSession struct {
	router    *wsSignRouter
	recvCh    chan recvItem
	errCh     chan<- error
	recvCount uint32
	mu        sync.Mutex
	closed    bool
}

func (s *signSession) enqueue(item recvItem) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return false
	}
	select {
	case s.recvCh <- item:
		return true
	default:
		logSignf("Deliver: recvCh full, dropping message fromIndex=%d (task=%s)\n",
			item.FromIndex, s.router.taskID)
		return false
	}
}

func (s *signSession) close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	close(s.recvCh)
	s.mu.Unlock()
}

var (
	signSessions   = make(map[string]*signSession)
	signSessionsMu sync.RWMutex
)

func signSessionKey(taskID, nodeID string) string {
	return taskID + "|" + nodeID
}

func registerSignSession(taskID, nodeID string, s *signSession) {
	key := signSessionKey(taskID, nodeID)

	// Step 1: 取出并清除该 session 的早期消息
	var replayItems []recvItem
	earlySignMessagesMu.Lock()
	cleanupExpiredEarlySignMessagesLocked(time.Now())
	if b, ok := earlySignMessages[key]; ok {
		replayItems = b.items
		delete(earlySignMessages, key)
	}
	earlySignMessagesMu.Unlock()

	// Step 2: 注册 session
	signSessionsMu.Lock()
	signSessions[key] = s
	signSessionsMu.Unlock()

	// Step 3: 回放早期消息
	for _, item := range replayItems {
		if !s.enqueue(item) {
			logSignf("replay early msg failed (session closed) task=%s node=%s\n", taskID, nodeID)
		} else {
			logSignf("replayed early msg task=%s node=%s fromIndex=%d\n", taskID, nodeID, item.FromIndex)
		}
	}
}

func unregisterSignSession(taskID, nodeID string) {
	// 同时清理可能残留的早期消息
	earlySignMessagesMu.Lock()
	delete(earlySignMessages, signSessionKey(taskID, nodeID))
	earlySignMessagesMu.Unlock()

	signSessionsMu.Lock()
	defer signSessionsMu.Unlock()
	delete(signSessions, signSessionKey(taskID, nodeID))
}

func getSignSession(taskID, nodeID string) *signSession {
	signSessionsMu.RLock()
	defer signSessionsMu.RUnlock()
	return signSessions[signSessionKey(taskID, nodeID)]
}

// runSignDelivery 串行处理收到的签名消息，确保 party.Update 不并发。
func runSignDelivery(s *signSession) {
	logSignf("task=%s myIndex=%d delivery goroutine started\n", s.router.taskID, s.router.myIndex)

	var earlyMsgs []recvItem

	for item := range s.recvCh {
		if item.FromIndex == s.router.myIndex {
			continue
		}

		if s.router.party != nil {
			for _, early := range earlyMsgs {
				processSignMessage(s, early)
			}
			earlyMsgs = nil

			processSignMessage(s, item)
			continue
		}

		if len(earlyMsgs) < 512 {
			earlyMsgs = append(earlyMsgs, item)
			logSignf("task=%s cached early msg fromIndex=%d (total=%d)\n",
				s.router.taskID, item.FromIndex, len(earlyMsgs))
		} else {
			logSignf("task=%s dropped early msg (buffer full) fromIndex=%d\n",
				s.router.taskID, item.FromIndex)
		}
	}
}

func processSignMessage(s *signSession, item recvItem) {
	logSignf("task=%s myIndex=%d before Update fromIndex=%d\n",
		s.router.taskID, s.router.myIndex, item.FromIndex)
	err := s.router.Receive(s.router.myIndex, item.WireBytes, item.FromIndex, item.IsBroadcast)
	logSignf("task=%s myIndex=%d after Update fromIndex=%d err=%v\n",
		s.router.taskID, s.router.myIndex, item.FromIndex, err)
	c := atomic.AddUint32(&s.recvCount, 1)
	if c <= 10 || c%20 == 0 {
		logSignf("task=%s myIndex=%d recvCount=%d fromIndex=%d\n",
			s.router.taskID, s.router.myIndex, c, item.FromIndex)
	}
	if err != nil && err.Error() != "Error is nil" {
		select {
		case s.errCh <- err:
		default:
		}
	}
}

// DeliverMpcSignMsg 由 main 的 Push 回调调用：根据 body 中的 taskID 找到会话，将消息放入投递队列由 delivery 协程串行 Update。
func DeliverMpcSignMsg(wsClient *sdk.SocketSDK, myNodeID, router string, body []byte) error {
	if len(body) == 0 {
		return nil
	}
	var decrypt dto.CliMPCEncryptData
	if err := utils.JsonUnmarshal(body, &decrypt); err != nil {
		return err
	}
	prk, err := getTempPrivateKey("sign", myNodeID, decrypt.TaskID)
	if err != nil {
		return err
	}
	if prk == nil {
		return errors.New("temp prk is nil")
	}
	msg, err := ecc.Decrypt(prk, utils.Base64Decode(decrypt.Data), utils.Str2Bytes(utils.AddStr(decrypt.TaskID, "|", myNodeID, "|mpcSignMsg")), nil)
	if err != nil {
		return err
	}
	var res dto.CliMPCSignMsgRes
	if err := utils.JsonUnmarshal(msg, &res); err != nil {
		logSignf("Deliver: json error = %v\n", err)
		return err
	}

	s := getSignSession(res.TaskID, myNodeID)
	if s == nil || s.router == nil {
		// Session 不存在：缓存为早期消息
		wireBytes, err := base64.StdEncoding.DecodeString(res.WireBytesBase64)
		if err != nil {
			logSignf("Deliver: base64 error = %v\n", err)
			return err
		}
		sessionKey := signSessionKey(res.TaskID, myNodeID)

		earlySignMessagesMu.Lock()
		now := time.Now()
		cleanupExpiredEarlySignMessagesLocked(now)
		if b, exists := earlySignMessages[sessionKey]; exists && len(b.items) >= maxEarlySignMsgs {
			earlySignMessagesMu.Unlock()
			logSignf("Deliver: dropped early msg (buffer full) task=%s node=%s\n", res.TaskID, myNodeID)
			return nil
		}
		item := recvItem{
			WireBytes:   wireBytes,
			FromIndex:   res.FromIndex,
			IsBroadcast: res.IsBroadcast,
		}
		b := earlySignMessages[sessionKey]
		if b.createdAt.IsZero() {
			b.createdAt = now
		}
		b.items = append(b.items, item)
		earlySignMessages[sessionKey] = b
		earlySignMessagesMu.Unlock()

		logSignf("Deliver: cached early msg task=%s node=%s fromIndex=%d\n", res.TaskID, myNodeID, res.FromIndex)
		return nil
	}

	logSignf("Deliver: enqueuing to session task=%s myIndex=%d fromIndex=%d\n",
		res.TaskID, s.router.myIndex, res.FromIndex)

	if res.FromIndex == s.router.myIndex {
		logSignf("Deliver: dropped (own) task=%s myIndex=%d fromIndex=%d\n",
			res.TaskID, s.router.myIndex, res.FromIndex)
		return nil
	}
	wireBytes, err := base64.StdEncoding.DecodeString(res.WireBytesBase64)
	if err != nil {
		logSignf("Deliver: base64 error = %v\n", err)
		return err
	}
	item := recvItem{
		WireBytes:   wireBytes,
		FromIndex:   res.FromIndex,
		IsBroadcast: res.IsBroadcast,
	}
	if !s.enqueue(item) {
		logSignf("Deliver: session already closed for task %s\n", res.TaskID)
		return nil
	}
	logSignf("Deliver: enqueued myIndex=%d fromIndex=%d task=%s\n", s.router.myIndex, res.FromIndex, res.TaskID)
	return nil
}
