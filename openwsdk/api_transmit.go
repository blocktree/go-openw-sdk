package openwsdk

import (
	"encoding/json"
	"fmt"
	"github.com/blocktree/OpenWallet/log"
	"github.com/blocktree/OpenWallet/owtp"
)

type TransmitNode struct {
	node              *owtp.OWTPNode
	config            *APINodeConfig
	disconnectHandler func(n *TrusteeshipNode) //托管节点断开连接后的通知
	connectHandler    func(n *TrusteeshipNode) //托管节点连接成功的通知
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
	connectCfg.Timeout = config.Timeout
	node := owtp.NewNode(owtp.NodeConfig{
		Cert: config.Cert,
	})

	t := &TransmitNode{
		node:   node,
		config: config,
	}
	return t, nil
}

//Listen 启动监听
func (transmit *TransmitNode) Listen() {

	//开启监听
	log.Infof("Transmit node port: %s start to listen [%s] connection...", transmit.config.Host, transmit.config.ConnectType)

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
func (transmit *TransmitNode) SetConnectHandler(h func(n *TrusteeshipNode)) {
	transmit.connectHandler = h
}

//SetDisconnectHandler 设置托管节点连接成功的通知
func (transmit *TransmitNode) SetDisconnectHandler(h func(n *TrusteeshipNode)) {
	transmit.disconnectHandler = h
}

//nodeJoin 节点加入
func (transmit *TransmitNode) nodeJoin(ctx *owtp.Context) {
	//:托管节点加入
	var trustNode TrusteeshipNode
	err := json.Unmarshal([]byte(ctx.Params().Get("nodeInfo").Raw), &trustNode)
	if err != nil {
		log.Errorf("new node joining failed, unexpected error: %v", err)
		return
	}
	transmit.connectHandler(&trustNode)
}

//nodeLeave 节点离开
func (transmit *TransmitNode) nodeLeave(ctx *owtp.Context) {
	//:托管节点离开
	var trustNode TrusteeshipNode
	err := json.Unmarshal([]byte(ctx.Params().Get("nodeInfo").Raw), &trustNode)
	if err != nil {
		log.Errorf("exist node leaving failed, unexpected error: %v", err)
		return
	}
	transmit.disconnectHandler(&trustNode)
}

//CreateTrusteeshipWallet 指定节点，创建种子托管钱包
func (transmit *TransmitNode) CreateTrusteeshipWallet(nodeID, alias, password string,
	sync bool, reqFunc func(status uint64, msg string, wallet *Wallet)) error {

	if transmit == nil {
		return fmt.Errorf("TransmitNode is not inited")
	}

	params := map[string]interface{}{
		"appID":    transmit.config.AppID,
		"alias":    alias,
		"password": password,
	}

	return transmit.node.Call(nodeID, "createWallet", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		var wallet Wallet
		json.Unmarshal([]byte(data.Raw), &wallet)
		reqFunc(resp.Status, resp.Msg, &wallet)
	})
}

//CreateTrusteeshipAccount 指定节点，创建种子托管钱包
func (transmit *TransmitNode) CreateTrusteeshipAccount(
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

	return transmit.node.Call(nodeID, "createAccount", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		var account Account
		json.Unmarshal([]byte(data.Get("account").Raw), &account)

		var addresses []*Address
		addressArray := data.Get("address").Array()
		for _, a := range addressArray {
			var addr Address
			err := json.Unmarshal([]byte(a.Raw), &addr)
			if err == nil {
				addresses = append(addresses, &addr)
			}
		}

		reqFunc(resp.Status, resp.Msg, &account, addresses)
	})
}

//SendTransaction 创建转账交易订单
func (transmit *TransmitNode) SendTransaction(
	nodeID string,
	accountID string,
	password string,
	sid string,
	coin Coin,
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
		"appID":     transmit.config.AppID,
		"accountID": accountID,
		"password":  password,
		"sid":       sid,
		"coin":      coin,
		"amount":    amount,
		"address":   address,
		"feeRate":   feeRate,
		"memo":      memo,
	}

	return transmit.node.Call(nodeID, "sendTransaction", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		failedRawTxs := make([]*FailedRawTransaction, 0)
		failedArray := data.Get("failure").Array()
		for _, failed := range failedArray {
			var rawTx RawTransaction
			json.Unmarshal([]byte(failed.Get("rawTx").Raw), &rawTx)
			failedRawTx := &FailedRawTransaction{
				Reason: failed.Get("error").String(),
				RawTx:  &rawTx,
			}
			failedRawTxs = append(failedRawTxs, failedRawTx)
		}

		var txs []*Transaction
		successArray := data.Get("success").Array()
		for _, a := range successArray {
			var tx Transaction
			err := json.Unmarshal([]byte(a.Raw), &tx)
			if err == nil {
				txs = append(txs, &tx)
			}
		}

		reqFunc(resp.Status, resp.Msg, txs, failedRawTxs)
	})
}

//SetSummaryInfo 指定节点，设置汇总信息
func (transmit *TransmitNode) SetSummaryInfo(
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

	return transmit.node.Call(nodeID, "setSummaryInfo", params, sync, func(resp owtp.Response) {
		reqFunc(resp.Status, resp.Msg)
	})
}

//FindSummaryInfoByWalletID 指定节点，获取汇总设置信息
func (transmit *TransmitNode) FindSummaryInfoByWalletID(
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

	return transmit.node.Call(nodeID, "findSummaryInfoByWalletID", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		var summaryInfoList []*SummarySetting
		summaryInfoArray := data.Get("summaryInfoList").Array()
		for _, a := range summaryInfoArray {
			var sumInfo SummarySetting
			err := json.Unmarshal([]byte(a.Raw), &sumInfo)
			if err == nil {
				summaryInfoList = append(summaryInfoList, &sumInfo)
			}
		}

		reqFunc(resp.Status, resp.Msg, summaryInfoList)
	})
}

//StartSummaryTask 指定节点，启动汇总任务
func (transmit *TransmitNode) StartSummaryTask(
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

	return transmit.node.Call(nodeID, "startSummaryTask", params, sync, func(resp owtp.Response) {
		reqFunc(resp.Status, resp.Msg)
	})
}

//StopSummaryTask 指定节点，停止汇总任务
func (transmit *TransmitNode) StopSummaryTask(
	nodeID string,
	sync bool, reqFunc func(status uint64, msg string)) error {
	if transmit == nil {
		return fmt.Errorf("TransmitNode is not inited")
	}
	params := map[string]interface{}{
		"appID": transmit.config.AppID,
	}

	return transmit.node.Call(nodeID, "stopSummaryTask", params, sync, func(resp owtp.Response) {
		reqFunc(resp.Status, resp.Msg)
	})
}

//UpdateInfo 指定节点，更新主链信息和合约资料
func (transmit *TransmitNode) UpdateInfo(
	nodeID string,
	sync bool, reqFunc func(status uint64, msg string)) error {
	if transmit == nil {
		return fmt.Errorf("TransmitNode is not inited")
	}
	params := map[string]interface{}{
		"appID": transmit.config.AppID,
	}

	return transmit.node.Call(nodeID, "updateInfo", params, sync, func(resp owtp.Response) {
		reqFunc(resp.Status, resp.Msg)
	})
}
