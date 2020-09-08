package openwsdk

import (
	"encoding/json"
	"fmt"
	"github.com/blocktree/openwallet/v2/log"
	"github.com/blocktree/openwallet/v2/openwallet"
	"github.com/blocktree/openwallet/v2/owtp"
)

const (
	/* 汇总任务操作类型 */
	SummaryTaskOperateTypeReset = 0 //重置
	SummaryTaskOperateTypeAdd   = 1 //追加
)

type TransmitNode struct {
	node              *owtp.OWTPNode
	config            *APINodeConfig
	disconnectHandler func(transmitNode *TransmitNode, nodeID string)           //托管节点断开连接后的通知
	connectHandler    func(transmitNode *TransmitNode, nodeInfo *TrustNodeInfo) //托管节点连接成功的通知
	parent            *APINode
}

func NewTransmitNode(config *APINodeConfig) (*TransmitNode, error) {

	if config.ConnectType != owtp.Websocket {
		return nil, fmt.Errorf("Transmit node only support websocket ")
	}

	connectCfg := owtp.ConnectConfig{}
	connectCfg.Address = config.Host
	connectCfg.EnableSSL = config.EnableSSL
	connectCfg.EnableSignature = config.EnableSignature
	connectCfg.ConnectType = config.ConnectType
	node := owtp.NewNode(owtp.NodeConfig{
		Cert:       config.Cert,
		TimeoutSEC: config.TimeoutSEC,
	})

	t := &TransmitNode{
		node:   node,
		config: config,
	}

	node.HandleFunc("newNodeJoin", t.newNodeJoin)

	node.SetCloseHandler(func(n *owtp.OWTPNode, peer owtp.PeerInfo) {
		if t.disconnectHandler != nil {
			t.disconnectHandler(t, peer.ID)
		}
	})

	return t, nil
}

//APINode
func (transmit *TransmitNode) APINode() (*APINode, error) {
	if transmit.parent == nil {
		return nil, fmt.Errorf("transmit node is not inited")
	}
	return transmit.parent, nil
}

//OWTPNode
func (transmit *TransmitNode) OWTPNode() (*owtp.OWTPNode, error) {
	if transmit.node == nil {
		return nil, fmt.Errorf("transmit node is not inited")
	}
	return transmit.node, nil
}

//Listen 启动监听
func (transmit *TransmitNode) Listen() {

	//开启监听
	log.Infof("Transmit node IP %s start to listen [%s] connection...", transmit.config.Host, transmit.config.ConnectType)

	transmit.node.Listen(owtp.ConnectConfig{
		Address:     transmit.config.Host,
		ConnectType: transmit.config.ConnectType,
	})
}

//Close 关闭监听
func (transmit *TransmitNode) Close() {
	transmit.node.Close()
}

//SetConnectHandler 设置托管节点断开连接后的通知
func (transmit *TransmitNode) SetConnectHandler(h func(transmitNode *TransmitNode, nodeInfo *TrustNodeInfo)) {
	transmit.connectHandler = h
}

//SetDisconnectHandler 设置托管节点连接成功的通知
func (transmit *TransmitNode) SetDisconnectHandler(h func(transmitNode *TransmitNode, nodeID string)) {
	transmit.disconnectHandler = h
}

func (transmit *TransmitNode) newNodeJoin(ctx *owtp.Context) {
	if transmit.connectHandler != nil {
		var nodeInfo TrustNodeInfo
		err := json.Unmarshal([]byte(ctx.Params().Get("nodeInfo").Raw), &nodeInfo)
		if err != nil {
			ctx.Response(nil, openwallet.ErrUnknownException, err.Error())
			return
		}
		transmit.connectHandler(transmit, &nodeInfo)
	}

	ctx.Response(nil, owtp.StatusSuccess, "success")
}

//GetTrustNodeInfo 获取授信的托管节点信息
func (transmit *TransmitNode) GetTrustNodeInfo(nodeID string,
	sync bool, reqFunc func(status uint64, msg string, nodeInfo *TrustNodeInfo)) error {

	if transmit == nil {
		return fmt.Errorf("TransmitNode is not inited")
	}

	if p := transmit.node.GetOnlinePeer(nodeID); p == nil {
		return fmt.Errorf("Node ID: %s is not connected ", nodeID)
	}

	params := map[string]interface{}{
		"appID": transmit.config.AppID,
	}

	return transmit.node.Call(nodeID, "getTrustNodeInfo", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		var nodeInfo TrustNodeInfo
		json.Unmarshal([]byte(data.Raw), &nodeInfo)
		reqFunc(resp.Status, resp.Msg, &nodeInfo)
	})
}

//CreateWalletViaTrustNode 指定节点，创建种子托管钱包
func (transmit *TransmitNode) CreateWalletViaTrustNode(nodeID, alias, password string,
	sync bool, reqFunc func(status uint64, msg string, wallet *Wallet)) error {

	if transmit == nil {
		return fmt.Errorf("TransmitNode is not inited")
	}

	if p := transmit.node.GetOnlinePeer(nodeID); p == nil {
		return fmt.Errorf("Node ID: %s is not connected ", nodeID)
	}

	params := map[string]interface{}{
		"appID":    transmit.config.AppID,
		"alias":    alias,
		"password": password,
	}

	return transmit.node.Call(nodeID, "createWalletViaTrustNode", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		var wallet Wallet
		json.Unmarshal([]byte(data.Raw), &wallet)
		reqFunc(resp.Status, resp.Msg, &wallet)
	})
}

//CreateAccountViaTrustNode 指定节点，创建种子托管钱包
func (transmit *TransmitNode) CreateAccountViaTrustNode(
	nodeID, walletID, alias, password, symbol string, sync bool,
	reqFunc func(status uint64, msg string, account *Account, addresses []*Address)) error {
	if transmit == nil {
		return fmt.Errorf("TransmitNode is not inited")
	}

	if p := transmit.node.GetOnlinePeer(nodeID); p == nil {
		return fmt.Errorf("Node ID: %s is not connected ", nodeID)
	}

	params := map[string]interface{}{
		"appID":    transmit.config.AppID,
		"alias":    alias,
		"walletID": walletID,
		"password": password,
		"symbol":   symbol,
	}

	return transmit.node.Call(nodeID, "createAccountViaTrustNode", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		var account Account
		json.Unmarshal([]byte(data.Get("account").Raw), &account)

		var addresses []*Address
		addressArray := data.Get("address")
		if addressArray.IsArray() {
			for _, a := range addressArray.Array() {
				var addr Address
				err := json.Unmarshal([]byte(a.Raw), &addr)
				if err == nil {
					addresses = append(addresses, &addr)
				}
			}
		}

		reqFunc(resp.Status, resp.Msg, &account, addresses)
	})
}

//SendTransactionViaTrustNode 创建转账交易订单
func (transmit *TransmitNode) SendTransactionViaTrustNode(
	nodeID string,
	accountID string,
	password string,
	sid string,
	contractAddress string,
	amount string,
	address string,
	feeRate string,
	memo string,
	extParam string,
	sync bool,
	reqFunc func(status uint64, msg string, successTx []*Transaction, failedRawTxs []*FailedRawTransaction),
) error {
	if transmit == nil {
		return fmt.Errorf("TransmitNode is not inited")
	}

	if p := transmit.node.GetOnlinePeer(nodeID); p == nil {
		return fmt.Errorf("Node ID: %s is not connected ", nodeID)
	}

	params := map[string]interface{}{
		"appID":           transmit.config.AppID,
		"accountID":       accountID,
		"password":        password,
		"sid":             sid,
		"contractAddress": contractAddress,
		"amount":          amount,
		"address":         address,
		"feeRate":         feeRate,
		"memo":            memo,
		"extParam":        extParam,
	}

	return transmit.node.Call(nodeID, "sendTransactionViaTrustNode", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		failedRawTxs := make([]*FailedRawTransaction, 0)
		failedArray := data.Get("failure")
		if failedArray.IsArray() {
			for _, failed := range failedArray.Array() {
				var failedRawTx FailedRawTransaction
				err := json.Unmarshal([]byte(failed.Raw), &failedRawTx)
				if err == nil {
					failedRawTxs = append(failedRawTxs, &failedRawTx)
				}
			}
		}

		var txs []*Transaction
		successArray := data.Get("success")
		if successArray.IsArray() {
			for _, a := range successArray.Array() {
				var tx Transaction
				err := json.Unmarshal([]byte(a.Raw), &tx)
				if err == nil {
					txs = append(txs, &tx)
				}
			}
		}

		reqFunc(resp.Status, resp.Msg, txs, failedRawTxs)
	})
}

//SetSummaryInfoViaTrustNode 指定节点，设置汇总信息
func (transmit *TransmitNode) SetSummaryInfoViaTrustNode(
	nodeID string,
	summarySetting *SummarySetting,
	sync bool, reqFunc func(status uint64, msg string)) error {
	if transmit == nil {
		return fmt.Errorf("TransmitNode is not inited")
	}

	if p := transmit.node.GetOnlinePeer(nodeID); p == nil {
		return fmt.Errorf("Node ID: %s is not connected ", nodeID)
	}

	params := map[string]interface{}{
		"appID":          transmit.config.AppID,
		"summarySetting": summarySetting,
	}

	return transmit.node.Call(nodeID, "setSummaryInfoViaTrustNode", params, sync, func(resp owtp.Response) {
		reqFunc(resp.Status, resp.Msg)
	})
}

//FindSummaryInfoByWalletIDViaTrustNode 指定节点，获取汇总设置信息
func (transmit *TransmitNode) FindSummaryInfoByWalletIDViaTrustNode(
	nodeID string,
	walletID string,
	sync bool, reqFunc func(status uint64, msg string, summarySettings []*SummarySetting)) error {
	if transmit == nil {
		return fmt.Errorf("TransmitNode is not inited")
	}

	if p := transmit.node.GetOnlinePeer(nodeID); p == nil {
		return fmt.Errorf("Node ID: %s is not connected ", nodeID)
	}

	params := map[string]interface{}{
		"appID":    transmit.config.AppID,
		"walletID": walletID,
	}

	return transmit.node.Call(nodeID, "findSummaryInfoByWalletIDViaTrustNode", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		var summaryInfoList []*SummarySetting
		if data.IsArray() {
			for _, a := range data.Array() {
				var sumInfo SummarySetting
				err := json.Unmarshal([]byte(a.Raw), &sumInfo)
				if err == nil {
					summaryInfoList = append(summaryInfoList, &sumInfo)
				}
			}
		}

		reqFunc(resp.Status, resp.Msg, summaryInfoList)
	})
}

// StartSummaryTaskViaTrustNode 指定节点，启动汇总任务
// nodeID 节点ID
// cycleSec 任务周期间隔
// summaryTask 汇总任务
// operateType 操作类型：0：重置，1：追加
func (transmit *TransmitNode) StartSummaryTaskViaTrustNode(
	nodeID string,
	cycleSec int,
	summaryTask *SummaryTask,
	operateType int,
	sync bool, reqFunc func(status uint64, msg string)) error {
	if transmit == nil {
		return fmt.Errorf("TransmitNode is not inited")
	}

	if p := transmit.node.GetOnlinePeer(nodeID); p == nil {
		return fmt.Errorf("Node ID: %s is not connected ", nodeID)
	}

	params := map[string]interface{}{
		"appID":       transmit.config.AppID,
		"cycleSec":    cycleSec,
		"summaryTask": summaryTask,
		"operateType": operateType,
	}

	return transmit.node.Call(nodeID, "startSummaryTaskViaTrustNode", params, sync, func(resp owtp.Response) {
		reqFunc(resp.Status, resp.Msg)
	})
}

//StopSummaryTaskViaTrustNode 指定节点，停止汇总任务
func (transmit *TransmitNode) StopSummaryTaskViaTrustNode(
	nodeID string,
	sync bool, reqFunc func(status uint64, msg string)) error {
	if transmit == nil {
		return fmt.Errorf("TransmitNode is not inited")
	}

	if p := transmit.node.GetOnlinePeer(nodeID); p == nil {
		return fmt.Errorf("Node ID: %s is not connected ", nodeID)
	}

	params := map[string]interface{}{
		"appID": transmit.config.AppID,
	}

	return transmit.node.Call(nodeID, "stopSummaryTaskViaTrustNode", params, sync, func(resp owtp.Response) {
		reqFunc(resp.Status, resp.Msg)
	})
}

//UpdateInfo UpdateInfoViaTrustNode，更新主链信息和合约资料
func (transmit *TransmitNode) UpdateInfoViaTrustNode(
	nodeID string,
	sync bool, reqFunc func(status uint64, msg string)) error {
	if transmit == nil {
		return fmt.Errorf("TransmitNode is not inited")
	}

	if p := transmit.node.GetOnlinePeer(nodeID); p == nil {
		return fmt.Errorf("Node ID: %s is not connected ", nodeID)
	}

	params := map[string]interface{}{
		"appID": transmit.config.AppID,
	}

	return transmit.node.Call(nodeID, "updateInfoViaTrustNode", params, sync, func(resp owtp.Response) {
		reqFunc(resp.Status, resp.Msg)
	})
}

//AppendSummaryTaskViaTrustNode 指定节点，追加汇总任务
func (transmit *TransmitNode) AppendSummaryTaskViaTrustNode(
	nodeID string,
	summaryTask *SummaryTask,
	sync bool, reqFunc func(status uint64, msg string)) error {
	if transmit == nil {
		return fmt.Errorf("TransmitNode is not inited")
	}

	if p := transmit.node.GetOnlinePeer(nodeID); p == nil {
		return fmt.Errorf("Node ID: %s is not connected ", nodeID)
	}

	params := map[string]interface{}{
		"appID":       transmit.config.AppID,
		"summaryTask": summaryTask,
	}

	return transmit.node.Call(nodeID, "appendSummaryTaskViaTrustNode", params, sync, func(resp owtp.Response) {
		reqFunc(resp.Status, resp.Msg)
	})
}

//RemoveSummaryTaskViaTrustNode 指定节点，移除汇总任务
func (transmit *TransmitNode) RemoveSummaryTaskViaTrustNode(
	nodeID string,
	walletID string,
	accountID string,
	sync bool, reqFunc func(status uint64, msg string)) error {
	if transmit == nil {
		return fmt.Errorf("TransmitNode is not inited")
	}

	if p := transmit.node.GetOnlinePeer(nodeID); p == nil {
		return fmt.Errorf("Node ID: %s is not connected ", nodeID)
	}

	params := map[string]interface{}{
		"appID":     transmit.config.AppID,
		"walletID":  walletID,
		"accountID": accountID,
	}

	return transmit.node.Call(nodeID, "removeSummaryTaskViaTrustNode", params, sync, func(resp owtp.Response) {
		reqFunc(resp.Status, resp.Msg)
	})
}

//GetCurrentSummaryTaskViaTrustNode 指定节点，获取当前的执行中的汇总任务
func (transmit *TransmitNode) GetCurrentSummaryTaskViaTrustNode(
	nodeID string,
	sync bool, reqFunc func(status uint64, msg string, summaryTask *SummaryTask)) error {
	if transmit == nil {
		return fmt.Errorf("TransmitNode is not inited")
	}

	if p := transmit.node.GetOnlinePeer(nodeID); p == nil {
		return fmt.Errorf("Node ID: %s is not connected ", nodeID)
	}

	params := map[string]interface{}{
		"appID": transmit.config.AppID,
	}

	return transmit.node.Call(nodeID, "getCurrentSummaryTaskViaTrustNode", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		var summaryTask SummaryTask
		json.Unmarshal([]byte(data.Raw), &summaryTask)
		reqFunc(resp.Status, resp.Msg, &summaryTask)
	})
}

//GetSummaryTaskLogViaTrustNode 指定节点，获取汇总日志列表
func (transmit *TransmitNode) GetSummaryTaskLogViaTrustNode(
	nodeID string,
	offset int,
	limit int,
	sync bool, reqFunc func(status uint64, msg string, taskLog []*SummaryTaskLog)) error {
	if transmit == nil {
		return fmt.Errorf("TransmitNode is not inited")
	}

	if p := transmit.node.GetOnlinePeer(nodeID); p == nil {
		return fmt.Errorf("Node ID: %s is not connected ", nodeID)
	}

	params := map[string]interface{}{
		"appID":  transmit.config.AppID,
		"offset": offset,
		"limit":  limit,
	}

	return transmit.node.Call(nodeID, "getSummaryTaskLogViaTrustNode", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		var taskLog []*SummaryTaskLog
		json.Unmarshal([]byte(data.Raw), &taskLog)
		reqFunc(resp.Status, resp.Msg, taskLog)
	})
}

//GetLocalWalletListViaTrustNode 指定节点，获取该节点创建的钱包
func (transmit *TransmitNode) GetLocalWalletListViaTrustNode(
	nodeID string,
	sync bool, reqFunc func(status uint64, msg string, wallets []*Wallet)) error {
	if transmit == nil {
		return fmt.Errorf("TransmitNode is not inited")
	}

	if p := transmit.node.GetOnlinePeer(nodeID); p == nil {
		return fmt.Errorf("Node ID: %s is not connected ", nodeID)
	}

	params := map[string]interface{}{
		"appID": transmit.config.AppID,
	}

	return transmit.node.Call(nodeID, "getLocalWalletListViaTrustNode", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		var list []*Wallet
		json.Unmarshal([]byte(data.Raw), &list)
		reqFunc(resp.Status, resp.Msg, list)
	})
}

//GetTrustAddressListViaTrustNode 指定节点，获取信任地址列表
func (transmit *TransmitNode) GetTrustAddressListViaTrustNode(
	nodeID string,
	symbol string,
	sync bool, reqFunc func(status uint64, msg string, trustAddressList []*TrustAddress, enableTrustAddress bool)) error {
	if transmit == nil {
		return fmt.Errorf("TransmitNode is not inited")
	}

	if p := transmit.node.GetOnlinePeer(nodeID); p == nil {
		return fmt.Errorf("Node ID: %s is not connected ", nodeID)
	}

	params := map[string]interface{}{
		"appID":  transmit.config.AppID,
		"symbol": symbol,
	}

	return transmit.node.Call(nodeID, "getTrustAddressListViaTrustNode", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		var (
			list               []*TrustAddress
			enableTrustAddress bool
		)
		trustAddressList := data.Get("trustAddressList")
		json.Unmarshal([]byte(trustAddressList.Raw), &list)
		enableTrustAddress = data.Get("enableTrustAddress").Bool()
		reqFunc(resp.Status, resp.Msg, list, enableTrustAddress)
	})
}

//SignTransactionViaTrustNode 指定节点，签名交易单
func (transmit *TransmitNode) SignTransactionViaTrustNode(
	nodeID string,
	walletID string,
	rawTx *RawTransaction,
	password string,
	sync bool,
	reqFunc func(status uint64, msg string, signedRawTx *RawTransaction),
) error {
	if transmit == nil {
		return fmt.Errorf("TransmitNode is not inited")
	}

	if p := transmit.node.GetOnlinePeer(nodeID); p == nil {
		return fmt.Errorf("Node ID: %s is not connected ", nodeID)
	}

	params := map[string]interface{}{
		"appID":    transmit.config.AppID,
		"walletID": walletID,
		"password": password,
		"rawTx":    rawTx,
	}

	return transmit.node.Call(nodeID, "signTransactionViaTrustNode", params, sync, func(resp owtp.Response) {

		if resp.Status == owtp.StatusSuccess {
			data := resp.JsonData()
			jsonRawTx := data.Get("signedRawTx")

			var rawTx RawTransaction
			err := json.Unmarshal([]byte(jsonRawTx.Raw), &rawTx)
			if err != nil {
				reqFunc(openwallet.ErrUnknownException, err.Error(), nil)
				return
			}

			reqFunc(resp.Status, resp.Msg, &rawTx)
		} else {
			reqFunc(resp.Status, resp.Msg, nil)
		}

	})
}

// TriggerABIViaTrustNode 触发ABI上链调用
// @param nodeID 必填 节点ID
// @param accountID 必填 账户ID
// @param password 可选 钱包解锁密码
// @param contractAddress 必填 合约地址
// @param contractABI 可选 ABI定义
// @param amount 可选 主币数量
// @param feeRate 可选 自定义手续费率
// @param abiParam 可选 ABI参数组
// @param raw 可选 原始交易单
// @param rawType 可选 原始交易单编码类型，0：hex字符串，1：json字符串，2：base64字符串
// @param sync 必填 是否同步线程
//@param reqFunc 必填 回调函数处理
func (transmit *TransmitNode) TriggerABIViaTrustNode(
	nodeID string,
	accountID string,
	password string,
	sid string,
	contractAddress string,
	contractABI string,
	amount string,
	feeRate string,
	abiParam []string,
	raw string,
	rawType uint64,
	awaitResult bool,
	sync bool,
	reqFunc func(status uint64, msg string, receipt *SmartContractReceipt),
) error {
	if transmit == nil {
		return fmt.Errorf("TransmitNode is not inited")
	}

	if p := transmit.node.GetOnlinePeer(nodeID); p == nil {
		return fmt.Errorf("Node ID: %s is not connected ", nodeID)
	}

	params := map[string]interface{}{
		"appID":           transmit.config.AppID,
		"accountID":       accountID,
		"password":        password,
		"sid":             sid,
		"contractAddress": contractAddress,
		"contractABI":     contractABI,
		"amount":          amount,
		"abiParam":        abiParam,
		"feeRate":         feeRate,
		"raw":             raw,
		"rawType":         rawType,
		"awaitResult":     awaitResult,
	}

	return transmit.node.Call(nodeID, "triggerABIViaTrustNode", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		var receipt SmartContractReceipt
		err := json.Unmarshal([]byte(data.Raw), &receipt)
		if err != nil {
			reqFunc(openwallet.ErrUnknownException, err.Error(), nil)
			return
		}

		reqFunc(resp.Status, resp.Msg, &receipt)
	})
}

// SignHashViaTrustNode 通过节点签名哈希消息
// @param walletID 必填 节点ID
// @param accountID 必填 钱包ID
// @param message 必填 账户ID
// @param password 可选 钱包解锁密码
// @param address 必填 地址
// @param symbol 可选 主链标识
// @param hdPath 可选 子密钥路径
// @param sync 必填 是否同步线程
//@param reqFunc 必填 回调函数处理
func (transmit *TransmitNode) SignHashViaTrustNode(
	nodeID string,
	walletID string,
	accountID string,
	address string,
	message string,
	password string,
	symbol string,
	hdPath string,
	sync bool,
	reqFunc func(status uint64, msg string, signature string),
) error {
	if transmit == nil {
		return fmt.Errorf("TransmitNode is not inited")
	}

	if p := transmit.node.GetOnlinePeer(nodeID); p == nil {
		return fmt.Errorf("Node ID: %s is not connected ", nodeID)
	}

	params := map[string]interface{}{
		"appID":     transmit.config.AppID,
		"walletID":  walletID,
		"accountID": accountID,
		"address":   address,
		"message":   message,
		"password":  password,
		"symbol":    symbol,
		"hdPath":    hdPath,
	}

	return transmit.node.Call(nodeID, "signHashViaTrustNode", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		signature := data.Get("signature").String()
		reqFunc(resp.Status, resp.Msg, signature)
	})
}
