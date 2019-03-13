package openwsdk

import (
	"encoding/json"
	"fmt"
	"github.com/blocktree/openwallet/log"
	"github.com/blocktree/openwallet/owtp"
)

type TransmitNode struct {
	node              *owtp.OWTPNode
	config            *APINodeConfig
	disconnectHandler func(transmitNode *TransmitNode, nodeID string)           //托管节点断开连接后的通知
	connectHandler    func(transmitNode *TransmitNode, nodeInfo *TrustNodeInfo) //托管节点连接成功的通知
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
			ctx.Response(nil, owtp.ErrCustomError, err.Error())
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
	sync bool,
	reqFunc func(status uint64, msg string, successTx []*Transaction, failedRawTxs []*FailedRawTransaction),
) error {
	if transmit == nil {
		return fmt.Errorf("TransmitNode is not inited")
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

//StartSummaryTaskViaTrustNode 指定节点，启动汇总任务
func (transmit *TransmitNode) StartSummaryTaskViaTrustNode(
	nodeID string,
	cycleSec int,
	summaryTask *SummaryTask,
	sync bool, reqFunc func(status uint64, msg string)) error {
	if transmit == nil {
		return fmt.Errorf("TransmitNode is not inited")
	}
	params := map[string]interface{}{
		"appID":       transmit.config.AppID,
		"cycleSec":    cycleSec,
		"summaryTask": summaryTask,
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
