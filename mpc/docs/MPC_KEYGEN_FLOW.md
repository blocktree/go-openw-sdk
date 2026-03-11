# MPC 初始密钥创建流程（多节点轮询）

本文档描述「服务端协调 + 多节点参与」的 TSS keygen 流程：服务端下发任务、轮询收齐各节点结果后返回 KeyID；**SaveData 只在各节点本地持久化**。

---

## 1. 流程概览

```
服务端                                    节点 1 .. N
   |                                          |
   |  1) 取在线 subject，排序为 nodeIDs        |
   |  2) 写任务元信息到 cache                 |
   |  3) Push「mpcKeygenStart」─────────────> |  收到后启动本地 TSS keygen
   |     (TaskID, NodeIDs, Threshold)          |  用 MessageRouter 经 WS 收发消息
   |                                          |
   |  <────────── POST /ws/mpcKeygenMsg ────── |  协议消息中继
   |  ────────── Push「mpcKeygenMsg」───────> |  其他节点 Update(parsed)
   |  ... 多轮 ...                             |
   |                                          |
   |  <────────── POST /ws/mpcKeygenResult ─── |  keygen 完成后上报 KeyID + 状态
   |  4) 轮询 cache 直到所有节点 Status=40    |  （只做一致性校验，不落盘 SaveData）
   |  5) 返回 KeyID                           |
```

- **服务端**：只做任务下发、TSS 消息中继、结果轮询与落盘，不参与 TSS 计算。
- **节点**：各自运行 TSS keygen 的 LocalParty，通过 WebSocket 收发协议消息，完成后上报本节点的 `LocalPartySaveData`（base64）。

---

## 2. 服务端流程

### 2.1 入口：CreateMPCKeyTask()

位置：`web/websocket2.go`

1. **校验**：`server != nil`，在线节点数 3 或 5。
2. **节点列表**：`server.GetConnManager().GetAllSubjectDevices()`，key 即为 subject（节点 ID），排序得到稳定 `nodeIDs`。
3. **门限**：3 节点 → threshold=2；5 节点 → threshold=3。
4. **任务 ID**：`taskID = utils.GetUUID(true)`，`expiredTime = now + 120s`。
5. **写 cache**：
   - `keyCache.Put("mpcMeta:"+taskID, MpcKeygenTaskMeta{TaskID, NodeIDs, Threshold, ExpiredTime}, 300)`
   - 对每个 subject：`keyCache.Put(subject+taskID, MpcKeygenNodeResult{TaskID, NodeID, Status:10}, 300)`
6. **下发**：对每个 subject Push 路由 `mpcKeygenStart`，body = `CliMPCKeygenStartRes{TaskID, NodeIDs, Threshold, ExpiredTime}`。
7. **轮询**：每 500ms 检查所有 `(subject, taskID)` 的 cache 是否 `Status == 40`；90s 超时。
8. **收齐后**：
   - 校验各节点 KeyID 一致、无 Err。
9. **返回**：KeyID（各节点各自持久化自己的 SaveData，服务端不落盘）。

### 2.2 节点上报结果：handleMpcKeygenResult

- **路由**：`POST /ws/mpcKeygenResult`
- **请求体**：`CliMPCKeygenResultReq{TaskID, NodeID, KeyID, SaveDataBase64, Err}`
- **逻辑**：用 `connCtx.GetUserIDString()` 得到 subject，查 cache `(subject, req.TaskID)`；若已 Status=40 直接返回 OK；否则更新为 Status=40 并写入 KeyID、SaveDataBase64、Err，再 Put 回 cache。
- **响应**：`CliMPCKeygenResultRes{OK, Err}`。

### 2.3 TSS 消息中继：handleMpcKeygenMsg

- **路由**：`POST /ws/mpcKeygenMsg`
- **请求体**：`CliMPCKeygenMsgReq{TaskID, WireBytesBase64, FromIndex, IsBroadcast, ToNodeIDs}`
- **逻辑**：根据 TaskID 从 cache 取出 `MpcKeygenTaskMeta`（NodeIDs、ExpiredTime）；若过期返回错误；否则构造 `CliMPCKeygenMsgRes`，若 `IsBroadcast` 则向除 FromIndex 外的所有 NodeIDs Push `mpcKeygenMsg`，否则向 `ToNodeIDs` 逐一 Push。
- **响应**：`{ "ok": true }`。

---

## 3. 节点侧流程（保存各自的份额）

节点程序（如 `node/main.go`）需完成以下逻辑。

### 3.1 订阅 Push：mpcKeygenStart

- 在 WebSocket 的 Push 回调里识别路由 `mpcKeygenStart`。
- Body 为 `CliMPCKeygenStartRes`：`TaskID, NodeIDs, Threshold, ExpiredTime`。
- 本节点 ID 即当前 subject（如 `cliConfig.Source`），在 `NodeIDs` 中的下标即为本方的 `partyIndex`。

### 3.2 启动 Keygen

- 使用 `mpc.PartyIDs(NodeIDs)` 得到 `SortedPartyIDs`，用 `mpc.Parameters(sortedIDs, partyIndex, Threshold)` 得到 params。
- 实现一个 **基于 WS 的 MessageRouter**：
  - **Send(fromIndex, msg)**：对 `msg` 调 `WireBytes()` 得到字节，base64 编码后 POST `/ws/mpcKeygenMsg`，body 为 `CliMPCKeygenMsgReq{TaskID, WireBytesBase64, FromIndex: fromIndex, IsBroadcast: msg.IsBroadcast(), ToNodeIDs: ...}`（若单播则填目标 nodeID 列表）。
  - **Receive**：由下方「接收 mpcKeygenMsg」触发，调 `tss.ParseWireMessage` 再 `party.Update(parsed)`。
- 创建 keygen LocalParty（可现场生成 PreParams 或使用预生成），`party.Start()`，在 goroutine 中消费 `outCh` 并调用 router.Send。
- 从 `endCh` 收到本节点的 `LocalPartySaveData` 后进入 3.4。

### 3.3 订阅 Push：mpcKeygenMsg

- 在 Push 回调里识别路由 `mpcKeygenMsg`。
- Body 为 `CliMPCKeygenMsgRes`：`TaskID, WireBytesBase64, FromIndex, IsBroadcast`。
- Base64 解码得到 wireBytes，用 `tss.ParseWireMessage(wireBytes, fromPartyID, isBroadcast)` 得到 parsed，再调用本节点当前 keygen party 的 `Update(parsed)`。

### 3.4 本地持久化 + 上报结果：POST /ws/mpcKeygenResult

- Keygen 成功后，在**节点本地**使用 `FileKeyStore` 将该节点的 `LocalPartySaveData` 落盘：
  - `store := mpc.NewFileKeyStore("mpc_keys")`
  - `store.Save(keyID, myNodeID, saveData)`
- 然后再将 `saveData` 用 `json.Marshal` + `base64.StdEncoding.EncodeToString` 变为 `SaveDataBase64`。
- KeyID 可由 `mpc.KeyIDFromSaveData(save.ECDSAPub.X(), save.ECDSAPub.Y())` 得到。
- 请求体：`CliMPCKeygenResultReq{TaskID, NodeID: subject, KeyID, SaveDataBase64, Err: ""}`；失败时填 `Err`。
- 收到 `CliMPCKeygenResultRes{OK: true}` 即表示服务端已记录，随后服务端轮询收齐后仅作一致性校验并返回 KeyID，不落盘 SaveData。

---

## 4. WebSocket 路由与 DTO 汇总

| 方向       | 路由 / 类型        | 说明 |
|------------|--------------------|------|
| 服务端→节点 | Push `mpcKeygenStart`  | 下发任务：TaskID, NodeIDs, Threshold, ExpiredTime |
| 节点→服务端 | POST `/ws/mpcKeygenMsg`   | TSS 协议消息中继：WireBytesBase64, FromIndex, IsBroadcast, ToNodeIDs |
| 服务端→节点 | Push `mpcKeygenMsg`  | 转发的 TSS 消息：TaskID, WireBytesBase64, FromIndex, IsBroadcast |
| 节点→服务端 | POST `/ws/mpcKeygenResult` | 上报 keygen 结果：TaskID, NodeID, KeyID, SaveDataBase64, Err |

DTO 定义见 `openwsdk/dto/cli.go`：

- **CliMPCKeygenStartRes**：TaskID, NodeIDs, Threshold, ExpiredTime  
- **CliMPCKeygenResultReq**：TaskID, NodeID, KeyID, SaveDataBase64, Err  
- **CliMPCKeygenResultRes**：OK, Err  
- **CliMPCKeygenMsgReq**：TaskID, WireBytesBase64, FromIndex, IsBroadcast, ToNodeIDs  
- **CliMPCKeygenMsgRes**：TaskID, WireBytesBase64, FromIndex, IsBroadcast  

---

## 5. 如何触发（服务端）

- **CLI 菜单**：在 tview 主菜单选择 **「Create MPC Key (TSS)」**（快捷键 7），会调用 `CreateMPCKeyTask()`；需先启动 WebSocket 服务，并保证 3 或 5 个节点在线。
- **代码调用**：`keyID, err := webapp.CreateMPCKeyTask()`（在已启动 WS 服务的前提下）。

---

## 6. 相关代码位置

| 内容           | 路径 |
|----------------|------|
| 服务端协调逻辑 | `web/websocket2.go`（CreateMPCKeyTask, handleMpcKeygenResult, handleMpcKeygenMsg） |
| 路由注册       | `web/websocket.go` 的 `NewSocket()` 中注册 `/ws/mpcKeygenResult`、`/ws/mpcKeygenMsg` |
| MPC 落盘与解码 | `mpc/keystore_file.go`（FileKeyStore.Save, DecodeSaveDataFromJSON，**仅在各节点本地使用**） |
| 节点程序入口   | `node/main.go`（需增加 mpcKeygenStart / mpcKeygenMsg / mpcKeygenResult 处理） |
| TSS keygen 核心 | `mpc/keygen.go`（RunKeygen）, `mpc/params.go`（PartyIDs, Parameters）, `mpc/transport.go`（MessageRouter） |

---

## 7. 与 SSS 流程的对比

| 步骤         | SSS（CreateShardingTask）     | MPC（CreateMPCKeyTask）           |
|--------------|------------------------------|-----------------------------------|
| 节点数量     | 3 或 5                       | 3 或 5                            |
| 服务端生成   | 随机 seed，SSS 分片          | 不生成密钥，只协调                |
| 下发内容     | 加密分片（shardingPost）     | 任务参数（mpcKeygenStart）        |
| 节点间交互   | 无                           | TSS 协议消息经服务端中继          |
| 结果         | 各节点持有一份分片           | 各节点持有一份 SaveData（本地 mpc_keys），服务端仅记录 KeyID/节点信息 |

---

## 8. 为何未生成 keyfile（协议未收敛）

若所有节点始终处于 status=10、无人上报 KeyID，说明 **TSS keygen 协议未收敛**：没有节点收到 `SaveData`，也就不会写 keyfile、不会 POST mpcKeygenResult。

**可能原因：**

1. **消息未足量到达**：某节点 Send 失败（如 POST /ws/mpcKeygenMsg 超时/错误），其他节点收不到该方消息，协议卡住。
2. **消息顺序/时序**：tss-lib 按轮推进，若某方过早发出下一轮消息而对方尚未收齐上一轮，可能不收敛（当前实现按到达顺序投递，未按轮缓冲）。
3. **并发 Update 已修复**：此前 Push 并发调 `party.Update` 会导致状态错乱，已改为单 goroutine 串行投递（`runKeygenDelivery` + `recvCh`）。

**诊断：**

- 节点日志中每投递 20 条消息会打一次：`[mpc-keygen] task=... myIndex=%d recvCount=%d (protocol progress)`。若三节点 recvCount 持续增长且数量级接近，说明消息在流动；若某节点明显偏少或长时间不增，可能是该节点收不到或 Send 失败。
- 超时时会打：`keygen timeout ... (recvCount=%d)`。对比三节点超时时的 recvCount，若都很大且接近，多半是协议逻辑/顺序问题；若某节点很小，多半是网络或中继问题。
- 若出现 `recvCh full, dropping message`，说明投递队列满（delivery 跟不上或 Update 阻塞），可能需增大缓冲或排查 Update 是否卡住。

---

## 9. subject 与 fromIndex 错位（Push 收不到 / 只收到己方）

若服务端日志里 **sender subject** 与 **fromIndex** 对不上（例如 fromIndex=1 但 sender subject=node3），说明「连接在服务端的身份」与「节点身份」错位：

- `meta.NodeIDs = [node1, node2, node3]`，下标 0=node1、1=node2、2=node3。
- 节点发 POST 时带的是自己的 **fromIndex = myIndex**（node1 发 0，node2 发 1，node3 发 2）。
- 服务端用 **connCtx.GetUserIDString()** 得到当前连接的 subject（即「这条连接是谁」）。

**正确情况**：发 fromIndex=0 的请求应来自 subject=node1，发 fromIndex=1 的应来自 subject=node2，发 fromIndex=2 的应来自 subject=node3。

**若错位**（例如 fromIndex=1 却 sender=node3）：则 SendToSubject(node1/node2/node3) 会发到「错误」的连接，导致有的节点收不到对方消息、只收到己方回显。

**处理**：保证每个节点进程用**各自配置**（如 cli_node1.json 的 `"source":"node1"`、cli_node2.json 的 `"source":"node2"`、cli_node3.json 的 `"source":"node3"`），且服务端在**建连/登录时用该 source（或等价字段）作为 subject** 登记连接，不要用连接顺序、clientNo 等会混用的标识。这样 GetUserIDString() 与节点身份一致，Push 才能正确送达。
