package main

import (
	"crypto/ecdsa"
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

// ============ 新增：早期消息缓存 ============
var (
	typeEarlyKeygenBucket = struct{}{}
	earlyKeygenMessages   = make(map[string]earlyKeygenBucket) // key = taskID|myNodeID
	earlyKeygenMessagesMu sync.Mutex
	maxEarlyMessages      = 512 // 与 recvCh 同级，避免早期丢关键消息
	earlyMsgTTL           = 2 * time.Minute
)

type earlyKeygenBucket struct {
	items     []recvItem
	createdAt time.Time
}

func cleanupExpiredEarlyKeygenMessagesLocked(now time.Time) {
	for k, b := range earlyKeygenMessages {
		if now.Sub(b.createdAt) > earlyMsgTTL {
			delete(earlyKeygenMessages, k)
		}
	}
}

// ============ 原有类型保持不变 ============
type recvItem struct {
	WireBytes   []byte
	FromIndex   int
	IsBroadcast bool
}

type keygenSession struct {
	router    *wsKeygenRouter
	recvCh    chan recvItem
	errCh     chan<- error
	recvCount uint32
	mu        sync.Mutex
	closed    bool
}

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

// ============ Session 管理（改造 register 函数） ============
var (
	keygenSessions   = make(map[string]*keygenSession)
	keygenSessionsMu sync.RWMutex
)

func keygenSessionKey(taskID, nodeID string) string {
	return taskID + "|" + nodeID
}

// registerKeygenSession now replays early messages
func registerKeygenSession(taskID, nodeID string, s *keygenSession) {
	key := keygenSessionKey(taskID, nodeID)

	// Step 1: 取出并清除该 session 的早期消息
	var replayItems []recvItem
	earlyKeygenMessagesMu.Lock()
	cleanupExpiredEarlyKeygenMessagesLocked(time.Now())
	if b, ok := earlyKeygenMessages[key]; ok {
		replayItems = b.items
		delete(earlyKeygenMessages, key)
	}
	earlyKeygenMessagesMu.Unlock()

	// Step 2: 注册 session
	keygenSessionsMu.Lock()
	keygenSessions[key] = s
	keygenSessionsMu.Unlock()

	// Step 3: 回放早期消息
	for _, item := range replayItems {
		if !s.enqueue(item) {
			fmt.Printf("[mpc-keygen] replay early msg failed (session closed) task=%s node=%s\n", taskID, nodeID)
		} else {
			fmt.Printf("[mpc-keygen] replayed early msg task=%s node=%s fromIndex=%d\n", taskID, nodeID, item.FromIndex)
		}
	}
}

func unregisterKeygenSession(taskID, nodeID string) {
	// 同时清理可能残留的早期消息
	earlyKeygenMessagesMu.Lock()
	delete(earlyKeygenMessages, keygenSessionKey(taskID, nodeID))
	earlyKeygenMessagesMu.Unlock()

	keygenSessionsMu.Lock()
	defer keygenSessionsMu.Unlock()
	delete(keygenSessions, keygenSessionKey(taskID, nodeID))
}

func getKeygenSession(taskID, nodeID string) *keygenSession {
	keygenSessionsMu.RLock()
	defer keygenSessionsMu.RUnlock()
	return keygenSessions[keygenSessionKey(taskID, nodeID)]
}

// ============ 消息投递逻辑（不变） ============
func runKeygenDelivery(s *keygenSession) {
	fmt.Printf("[mpc-keygen] task=%s myIndex=%d delivery goroutine started\n", s.router.taskID, s.router.myIndex)

	var earlyMsgs []recvItem

	for item := range s.recvCh {
		if item.FromIndex == s.router.myIndex {
			continue
		}

		if s.router.party != nil {
			// 先回放缓存的早期协议消息
			for _, early := range earlyMsgs {
				processMessage(s, early)
			}
			earlyMsgs = nil
			processMessage(s, item)
			continue
		}

		// party 未就绪：缓存（最多512条）
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

// ============ DeliverMpcKeygenMsg（改造：支持缓存早期消息） ============
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

	taskID := res.TaskID
	sessionKey := keygenSessionKey(taskID, myNodeID)

	s := getKeygenSession(taskID, myNodeID)
	if s == nil {
		// ❗ Session 不存在：缓存为早期消息
		wireBytes, err := base64.StdEncoding.DecodeString(res.WireBytesBase64)
		if err != nil {
			fmt.Println("[mpc-keygen] Deliver: base64 decode error =", err)
			return err
		}

		earlyKeygenMessagesMu.Lock()
		now := time.Now()
		cleanupExpiredEarlyKeygenMessagesLocked(now)
		if b, exists := earlyKeygenMessages[sessionKey]; exists && len(b.items) >= maxEarlyMessages {
			earlyKeygenMessagesMu.Unlock()
			fmt.Printf("[mpc-keygen] Deliver: dropped early msg (buffer full) task=%s node=%s\n", taskID, myNodeID)
			return nil
		}
		item := recvItem{
			WireBytes:   wireBytes,
			FromIndex:   res.FromIndex,
			IsBroadcast: res.IsBroadcast,
		}
		b := earlyKeygenMessages[sessionKey]
		if b.createdAt.IsZero() {
			b.createdAt = now
		}
		b.items = append(b.items, item)
		earlyKeygenMessages[sessionKey] = b
		earlyKeygenMessagesMu.Unlock()

		fmt.Printf("[mpc-keygen] Deliver: cached early msg task=%s node=%s fromIndex=%d\n", taskID, myNodeID, res.FromIndex)
		return nil
	}

	// 己方消息不处理
	if res.FromIndex == s.router.myIndex {
		fmt.Printf("[mpc-keygen] Deliver: dropped (own) task=%s myIndex=%d fromIndex=%d\n",
			taskID, s.router.myIndex, res.FromIndex)
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
		fmt.Println("[mpc-keygen] Deliver: session already closed for task", taskID)
		return nil
	}
	fmt.Printf("[mpc-keygen] Deliver: enqueued myIndex=%d fromIndex=%d task=%s\n", s.router.myIndex, res.FromIndex, taskID)
	return nil
}

// ============ 以下是你原有的业务逻辑（未改动，仅保留上下文） ============

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

	session := getKeygenSession(taskID, myNodeID)
	if session == nil {
		return keygen.LocalPartySaveData{}, "", errors.New("session disappeared during keygen")
	}

	session.router.sortedIDs = sortedIDs
	session.router.myIndex = myIndex
	session.router.party = party
	session.router.wsClient = wsClient

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

func SubmitKeygenResult(wsClient *sdk.SocketSDK, taskID, nodeID, keyID string, saveData keygen.LocalPartySaveData) error {
	var rootPubHex string
	if saveData.ECDSAPub != nil {
		pub := &ecdsa.PublicKey{
			Curve: tss.S256(),
			X:     saveData.ECDSAPub.X(),
			Y:     saveData.ECDSAPub.Y(),
		}
		rootPubHex = mpc.PubKeyToHex(pub)
	}
	req := &dto.CliMPCKeygenResultReq{
		TaskID:     taskID,
		NodeID:     nodeID,
		KeyID:      keyID,
		RootPubHex: rootPubHex,
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

	for _, v := range start.PublicKeyPair {
		if v.Subject == myNodeID {
			continue
		}
		cacheKey := utils.FNV1a64(utils.AddStr(v.Subject, ":", start.TaskID, ":keygen:tempPublicKey"))
		if err := keyCache.Put(cacheKey, v.PublicKey, 600); err != nil {
			return errors.New("handleTempPublicKey put tempPublicKey error: " + err.Error())
		}
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

	recvCh := make(chan recvItem, 512)
	errCh := make(chan error, 4)

	routerStub := &wsKeygenRouter{
		taskID:    start.TaskID,
		subject:   myNodeID,
		sortedIDs: sortedIDs,
		myIndex:   myIndex,
		wsClient:  wsClient,
	}

	session := &keygenSession{
		router: routerStub,
		recvCh: recvCh,
		errCh:  errCh,
	}
	// 先启动 delivery，再注册/回放，减少回放时塞满队列的概率
	go runKeygenDelivery(session)
	registerKeygenSession(start.TaskID, myNodeID, session) // 👈 注册时会自动回放早期消息！

	go func() {
		defer func() {
			session.close()
			unregisterKeygenSession(start.TaskID, myNodeID)
			keygenTempPrk := utils.FNV1a64(utils.AddStr(myNodeID, ":", start.TaskID, ":keygen:tempPrivateKey"))
			_ = keyCache.Del(keygenTempPrk)
			for _, v := range start.NodeIDs {
				signTempPub := utils.FNV1a64(utils.AddStr(v, ":", start.TaskID, ":keygen:tempPublicKey"))
				_ = keyCache.Del(signTempPub)
			}
		}()

		saveData, keyID, err := RunKeygenNodeReal(start.TaskID, start.NodeIDs, myNodeID, start.Threshold, wsClient)
		if err != nil {
			fmt.Printf("[mpc-keygen] node=%s task=%s failed: %v\n", myNodeID, start.TaskID, err)
			_ = submitKeygenResultErr(wsClient, start.TaskID, myNodeID, err.Error())
			return
		}

		fmt.Printf("[mpc-keygen] node=%s task=%s succeeded, keyID=%s, saving local share and submitting result\n",
			myNodeID, start.TaskID, keyID)

		baseDir := fmt.Sprintf("keys")
		store := mpc.NewFileKeyStore(baseDir)
		if err := store.Save(keyID, myNodeID, saveData); err != nil {
			fmt.Printf("[mpc-keygen] node=%s task=%s save local share failed: %v\n", myNodeID, start.TaskID, err)
			_ = submitKeygenResultErr(wsClient, start.TaskID, myNodeID, "save local share failed: "+err.Error())
			return
		}

		if err := SubmitKeygenResult(wsClient, start.TaskID, myNodeID, keyID, saveData); err != nil {
			fmt.Printf("[mpc-keygen] node=%s task=%s submit result failed: %v\n", myNodeID, start.TaskID, err)
			_ = submitKeygenResultErr(wsClient, start.TaskID, myNodeID, "submit result failed: "+err.Error())
		}
	}()

	return nil
}

type wsKeygenRouter struct {
	taskID    string
	myIndex   int
	subject   string
	sortedIDs tss.SortedPartyIDs
	party     tss.Party
	wsClient  *sdk.SocketSDK
}

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
	} else {
		for _, v := range r.sortedIDs {
			if v.GetId() == r.subject {
				continue
			}
			toNodeIDs = append(toNodeIDs, v.GetId())
		}
	}

	for _, targetNodeID := range toNodeIDs {
		payload := &dto.CliMPCKeygenMsgRes{
			TaskID:          r.taskID,
			WireBytesBase64: base64.StdEncoding.EncodeToString(wireBytes),
			FromIndex:       fromIndex,
			IsBroadcast:     msg.IsBroadcast(),
		}
		data, err := utils.JsonMarshal(payload)
		if err != nil {
			return err
		}
		publicKey, err := getTempPublicKey("keygen", targetNodeID, r.taskID)
		if err != nil {
			return err
		}
		if len(publicKey) == 0 {
			fmt.Printf("[mpc-keygen] Send: task=%s no public key for target %s, skip\n", r.taskID, targetNodeID)
			continue
		}
		encrypt, err := ecc.Encrypt(nil, publicKey, data, utils.Str2Bytes(utils.AddStr(r.taskID, "|", targetNodeID, "|mpcKeygenMsg")))
		if err != nil {
			return err
		}
		if err := r.wsClient.SendWebSocketMessage("/ws/mpcKeygenMsg", &dto.CliMPCEncryptData{
			TaskID:  r.taskID,
			Subject: targetNodeID,
			Data:    utils.Base64Encode(encrypt),
		}, &dto.CliMPCResultRes{}, true, true, 60); err != nil {
			fmt.Printf("[mpc-keygen] Send: task=%s fromIndex=%d to %s rpc error=%v\n", r.taskID, fromIndex, targetNodeID, err)
			return err
		}
	}
	return nil
}

func (r *wsKeygenRouter) Receive(toIndex int, wireBytes []byte, fromIndex int, isBroadcast bool) error {
	if r.party == nil || fromIndex < 0 || fromIndex >= len(r.sortedIDs) {
		return nil
	}
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
