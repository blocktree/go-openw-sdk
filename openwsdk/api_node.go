package openwsdk

import (
	"encoding/json"
	"fmt"
	"github.com/blocktree/openwallet/crypto"
	"github.com/blocktree/openwallet/hdkeystore"
	"github.com/blocktree/openwallet/log"
	"github.com/blocktree/openwallet/owtp"
	"sync"
	"time"
)

const (
	HostNodeID = "openw-server"
)

const (
	/* 回调模式 */
	CallbackModeCurrentConnection = 1 //当前连接模式，长连接可以用当前连接，接收推送
	CallbackModeNewConnection     = 2 //新建连接模式，短连接可以采用建立回调服务接口接收推送
)

type APINodeConfig struct {
	Host               string           `json:"host"`
	AppID              string           `json:"appid"`
	AppKey             string           `json:"appkey"`
	ConnectType        string           `json:"connectType"`
	EnableKeyAgreement bool             `json:"enableKeyAgreement"`
	EnableSSL          bool             `json:"enableSSL"`
	EnableSignature    bool             `json:"enableSignature"`
	Cert               owtp.Certificate `json:"cert"`
	TimeoutSEC         int              `json:"timeoutSEC"`
	//HostNodeID string           `json:"hostNodeID"`
}

//APINode APINode通信节点
type APINode struct {
	mu           sync.RWMutex //读写锁
	node         *owtp.OWTPNode
	config       *APINodeConfig
	observers    map[OpenwNotificationObject]bool //观察者
	transmitNode *TransmitNode                    //钱包转发节点
}

//NewAPINodeWithError 创建API节点
func NewAPINodeWithError(config *APINodeConfig) (*APINode, error) {
	connectCfg := owtp.ConnectConfig{}
	connectCfg.Address = config.Host
	connectCfg.ConnectType = config.ConnectType
	connectCfg.EnableSSL = config.EnableSSL
	connectCfg.EnableSignature = config.EnableSignature
	node := owtp.NewNode(owtp.NodeConfig{
		Cert:       config.Cert,
		TimeoutSEC: config.TimeoutSEC,
	})
	err := node.Connect(HostNodeID, connectCfg)
	if err != nil {
		return nil, err
	}
	api := APINode{
		node:   node,
		config: config,
	}

	api.observers = make(map[OpenwNotificationObject]bool)

	//开启协商密码
	if config.EnableKeyAgreement {
		if err := node.KeyAgreement(HostNodeID, "aes"); err != nil {
			log.Error(err)
			return nil, err
		}
	}

	api.node.HandleFunc("subscribeToAccount", api.subscribeToAccount)
	api.node.HandleFunc("subscribeToTrade", api.subscribeToTrade)
	api.node.HandleFunc("subscribeToBlock", api.subscribeToBlock)

	return &api, nil
}

//NewAPINode 创建API节点
func NewAPINode(config *APINodeConfig) *APINode {
	api, err := NewAPINodeWithError(config)
	if err != nil {
		return nil
	}
	return api
}

//NodeID
func (api *APINode) NodeID() string {
	if api == nil {
		return ""
	}
	return api.node.NodeID()
}

//Subscribe 订阅
func (api *APINode) Subscribe(subscribeMethod []string, listenAddr string, callbackMode int, callbackNode CallbackNode) error {

	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}

	//http不能用当前连接模式
	if callbackMode == CallbackModeCurrentConnection {
		if api.config.ConnectType != owtp.Websocket {
			return fmt.Errorf("%s can not use [SubscribeModeCurrentConnection]", api.config.ConnectType)
		}
	} else {
		//开启监听
		log.Infof("%s start to listen [%s] connection...", listenAddr, callbackNode.ConnectType)
		api.node.Listen(owtp.ConnectConfig{
			Address:     listenAddr,
			ConnectType: callbackNode.ConnectType,
		})

	}

	params := map[string]interface{}{
		//"subscriptions": subscriptions,
		"appID":           api.config.AppID,
		"subscribeMethod": subscribeMethod,
		"callbackMode":    callbackMode,
		"callbackNode":    callbackNode,
	}

	response, err := api.node.CallSync(HostNodeID, "subscribe", params)
	if err != nil {
		return err
	}

	if response.Status == owtp.StatusSuccess {
		return nil
	} else {
		//关闭临时开启的端口
		log.Infof("%s close listener [%s] connection...", listenAddr, callbackNode.ConnectType)
		api.node.CloseListener(callbackNode.ConnectType)
		return fmt.Errorf("[%d]%s", response.Status, response.Msg)
	}

	return nil
}

//signAppDevice 生成登记节点的签名
func (api *APINode) signAppDevice(appID, nodID, appkey string, accessTime int64) string {
	// 校验签名
	plainText := fmt.Sprintf("%s.%s.%d.%s", appID, nodID, accessTime, appkey)
	signature := crypto.GetMD5(plainText)
	return signature
}

//BindAppDevice 绑定通信节点
//绑定节点ID成功，才能获得授权通信
func (api *APINode) BindAppDevice() error {

	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}

	nodeID := api.config.Cert.ID()
	accessTime := time.Now().UnixNano()
	sig := api.signAppDevice(api.config.AppID, nodeID, api.config.AppKey, accessTime)

	params := map[string]interface{}{
		"appID":      api.config.AppID,
		"deviceID":   nodeID,
		"accessTime": accessTime,
		"sign":       sig,
	}

	response, err := api.node.CallSync(HostNodeID, "bindAppDevice", params)
	if err != nil {
		return err
	}

	if response.Status == owtp.StatusSuccess {
		return nil
	} else {
		return fmt.Errorf("[%d]%s", response.Status, response.Msg)
	}

	return nil
}

//GetSymbolList 获取主链列表
func (api *APINode) GetSymbolList(offset, limit int, sync bool, reqFunc func(status uint64, msg string, symbols []*Symbol)) error {

	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}

	params := map[string]interface{}{
		"appID":  api.config.AppID,
		"offset": offset,
		"limit":  limit,
	}

	return api.node.Call(HostNodeID, "getSymbolList", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		symbols := make([]*Symbol, 0)
		symbolArray := data.Get("symbols")
		if symbolArray.IsArray() {
			for _, s := range symbolArray.Array() {
				var sym Symbol
				err := json.Unmarshal([]byte(s.Raw), &sym)
				if err == nil {
					symbols = append(symbols, &sym)
				}
			}
		}

		reqFunc(resp.Status, resp.Msg, symbols)
	})
}

//CreateWallet 创建钱包
func (api *APINode) CreateWallet(alias, walletID string, sync bool, reqFunc func(status uint64, msg string, wallet *Wallet)) error {

	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}

	params := map[string]interface{}{
		"appID":    api.config.AppID,
		"alias":    alias,
		"walletID": walletID,
		"rootPath": hdkeystore.OpenwCoinTypePath,
		"isTrust":  0,
	}

	return api.node.Call(HostNodeID, "createWallet", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		var wallet Wallet
		json.Unmarshal([]byte(data.Raw), &wallet)
		reqFunc(resp.Status, resp.Msg, &wallet)
	})
}

//FindWalletByWalletID 通过钱包ID获取钱包信息
func (api *APINode) FindWalletByWalletID(walletID string, sync bool, reqFunc func(status uint64, msg string, wallet *Wallet)) error {
	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}
	params := map[string]interface{}{
		"appID":    api.config.AppID,
		"walletID": walletID,
	}

	return api.node.Call(HostNodeID, "findWalletByWalletID", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		var wallet Wallet
		json.Unmarshal([]byte(data.Raw), &wallet)
		reqFunc(resp.Status, resp.Msg, &wallet)
	})
}

//CreateAccount 创建资产账户
func (api *APINode) CreateNormalAccount(
	accountParam *Account,
	sync bool,
	reqFunc func(status uint64, msg string, account *Account, addresses []*Address)) error {
	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}
	params := map[string]interface{}{
		"appID":        api.config.AppID,
		"alias":        accountParam.Alias,
		"walletID":     accountParam.WalletID,
		"accountID":    accountParam.AccountID,
		"symbol":       accountParam.Symbol,
		"publicKey":    accountParam.PublicKey,
		"accountIndex": accountParam.AccountIndex,
		"hdPath":       accountParam.HdPath,
		"reqSigs":      accountParam.ReqSigs,
		"isTrust":      0,
	}

	return api.node.Call(HostNodeID, "createAccount", params, sync, func(resp owtp.Response) {
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

//FindAccountByAccountID 通过资产账户ID获取资产账户信息
func (api *APINode) FindAccountByAccountID(accountID string, sync bool, reqFunc func(status uint64, msg string, account *Account)) error {
	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}
	params := map[string]interface{}{
		"appID":     api.config.AppID,
		"accountID": accountID,
	}

	return api.node.Call(HostNodeID, "findAccountByAccountID", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		var account Account
		json.Unmarshal([]byte(data.Raw), &account)
		reqFunc(resp.Status, resp.Msg, &account)
	})
}

//FindAccountByWalletID 通过钱包ID获取资产账户列表信息
func (api *APINode) FindAccountByWalletID(walletID string, sync bool, reqFunc func(status uint64, msg string, accounts []*Account)) error {
	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}
	params := map[string]interface{}{
		"appID":    api.config.AppID,
		"walletID": walletID,
	}

	return api.node.Call(HostNodeID, "findAccountByWalletID", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()

		var accounts []*Account
		accountArray := data
		if accountArray.IsArray() {
			for _, a := range accountArray.Array() {
				var acc Account
				err := json.Unmarshal([]byte(a.Raw), &acc)
				if err == nil {
					accounts = append(accounts, &acc)
				}
			}
		}
		reqFunc(resp.Status, resp.Msg, accounts)
	})
}

//CreateAddress 创建资产账户的地址
func (api *APINode) CreateAddress(
	walletID string,
	accountID string,
	count uint64,
	sync bool,
	reqFunc func(status uint64, msg string, addresses []*Address)) error {
	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}
	params := map[string]interface{}{
		"appID":     api.config.AppID,
		"walletID":  walletID,
		"accountID": accountID,
		"count":     count,
	}

	return api.node.Call(HostNodeID, "createAddress", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()

		var addresses []*Address
		addressArray := data
		if addressArray.IsArray() {
			for _, a := range addressArray.Array() {
				var addr Address
				err := json.Unmarshal([]byte(a.Raw), &addr)
				if err == nil {
					addresses = append(addresses, &addr)
				}
			}

		}
		reqFunc(resp.Status, resp.Msg, addresses)
	})
}


//CreateBatchAddress 批量创建资产账户的地址
func (api *APINode) CreateBatchAddress(
	walletID string,
	accountID string,
	count uint64,
	sync bool,
	reqFunc func(status uint64, msg string, addresses []string)) error {


	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}
	params := map[string]interface{}{
		"appID":     api.config.AppID,
		"walletID":  walletID,
		"accountID": accountID,
		"count":     count,
	}

	return api.node.Call(HostNodeID, "createBatchAddress", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		var addresses []string
		addressArray := data
		if addressArray.IsArray() {
			for _, a := range addressArray.Array() {
				addresses = append(addresses, a.String())
			}

		}
		reqFunc(resp.Status, resp.Msg, addresses)
	})
}

//FindAddressByAddress 通获取具体交易地址信息
func (api *APINode) FindAddressByAddress(address string, sync bool, reqFunc func(status uint64, msg string, address *Address)) error {
	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}
	params := map[string]interface{}{
		"appID":   api.config.AppID,
		"address": address,
	}

	return api.node.Call(HostNodeID, "findAddressByAddress", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		var address Address
		json.Unmarshal([]byte(data.Raw), &address)
		reqFunc(resp.Status, resp.Msg, &address)
	})
}

//FindAccountByWalletID 通过资产账户ID获取交易地址列表
func (api *APINode) FindAddressByAccountID(accountID string, offset int, limit int, sync bool, reqFunc func(status uint64, msg string, addresses []*Address)) error {
	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}
	params := map[string]interface{}{
		"appID":     api.config.AppID,
		"accountID": accountID,
		"offset":    offset,
		"limit":     limit,
	}

	return api.node.Call(HostNodeID, "findAddressByAccountID", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()

		var addresses []*Address
		array := data
		if array.IsArray() {
			for _, a := range array.Array() {
				var addr Address
				err := json.Unmarshal([]byte(a.Raw), &addr)
				if err == nil {
					addresses = append(addresses, &addr)
				}
			}
		}

		reqFunc(resp.Status, resp.Msg, addresses)
	})
}

//CreateTrade 创建转账交易订单
func (api *APINode) CreateTrade(
	accountID string,
	sid string,
	coin Coin,
	amount string,
	address string,
	feeRate string,
	memo string,
	sync bool,
	reqFunc func(status uint64, msg string, rawTx *RawTransaction),
) error {
	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}
	params := map[string]interface{}{
		"appID":     api.config.AppID,
		"accountID": accountID,
		"sid":       sid,
		"coin":      coin,
		"amount":    amount,
		"address":   address,
		"feeRate":   feeRate,
		"memo":      memo,
	}

	return api.node.Call(HostNodeID, "createTrade", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		jsonRawTx := data.Get("rawTx")

		var rawTx RawTransaction
		json.Unmarshal([]byte(jsonRawTx.Raw), &rawTx)

		reqFunc(resp.Status, resp.Msg, &rawTx)
	})
}

//SubmitTrade 广播转账交易订单
func (api *APINode) SubmitTrade(
	rawTx []*RawTransaction,
	sync bool,
	reqFunc func(status uint64, msg string, successTx []*Transaction, failedRawTxs []*FailedRawTransaction),
) error {
	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}
	params := map[string]interface{}{
		"appID": api.config.AppID,
		"rawTx": rawTx,
	}

	return api.node.Call(HostNodeID, "submitTrade", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		failedRawTxs := make([]*FailedRawTransaction, 0)
		failedArray := data.Get("failure")
		if failedArray.IsArray() {
			for _, failed := range failedArray.Array() {
				var rawTx RawTransaction
				err := json.Unmarshal([]byte(failed.Get("rawTx").Raw), &rawTx)
				if err == nil {
					failedRawTx := &FailedRawTransaction{
						Reason: failed.Get("error").String(),
						RawTx:  &rawTx,
					}

					failedRawTxs = append(failedRawTxs, failedRawTx)
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

//FindTradeLog 获取转账交易订单日志
func (api *APINode) FindTradeLog(
	walletID string,
	accountID string,
	txid string,
	address string,
	isTmp int,
	orderType int,
	offset int,
	limit int,
	sync bool,
	reqFunc func(status uint64, msg string, tx []*Transaction),
) error {
	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}
	params := map[string]interface{}{
		"appID":     api.config.AppID,
		"walletID":  walletID,
		"accountID": accountID,
		"txid":      txid,
		"isTmp":     isTmp,
		"orderType": orderType,
		"offset":    offset,
		"limit":     limit,
	}

	return api.node.Call(HostNodeID, "findTradeLog", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()

		var txs []*Transaction
		array := data
		if array.IsArray() {
			for _, a := range array.Array() {
				var tx Transaction
				err := json.Unmarshal([]byte(a.Raw), &tx)
				if err == nil {
					txs = append(txs, &tx)
				}
			}
		}

		reqFunc(resp.Status, resp.Msg, txs)
	})
}

//GetContracts 获取智能合约
func (api *APINode) GetContracts(
	symbol string,
	offset, limit int,
	sync bool,
	reqFunc func(status uint64, msg string, tokenContract []*TokenContract)) error {
	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}
	params := map[string]interface{}{
		"appID":  api.config.AppID,
		"symbol": symbol,
		"offset": offset,
		"limit":  limit,
	}

	return api.node.Call(HostNodeID, "getContracts", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		tokens := make([]*TokenContract, 0)
		array := data.Get("contracts")
		if array.IsArray() {
			for _, s := range array.Array() {
				var t TokenContract
				err := json.Unmarshal([]byte(s.Raw), &t)
				if err == nil {
					tokens = append(tokens, &t)
				}
			}
		}

		reqFunc(resp.Status, resp.Msg, tokens)
	})
}

//GetTokenBalanceByAccount 获取token余额接口
func (api *APINode) GetTokenBalanceByAccount(
	accountID string,
	contractID string,
	sync bool,
	reqFunc func(status uint64, msg string, balance string)) error {
	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}
	params := map[string]interface{}{
		"appID":      api.config.AppID,
		"accountID":  accountID,
		"contractID": contractID,
	}

	return api.node.Call(HostNodeID, "getTokenBalanceByAccount", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		balance := data.Get("balance").String()
		reqFunc(resp.Status, resp.Msg, balance)
	})
}

//GetFeeRate 获取推荐手续费率接口
func (api *APINode) GetFeeRate(
	symbol string,
	sync bool,
	reqFunc func(status uint64, msg string, symbol, feeRate, unit string)) error {
	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}
	params := map[string]interface{}{
		"appID":  api.config.AppID,
		"symbol": symbol,
	}

	return api.node.Call(HostNodeID, "getFeeRate", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		symbol := data.Get("symbol").String()
		feeRate := data.Get("feeRate").String()
		unit := data.Get("unit").String()
		reqFunc(resp.Status, resp.Msg, symbol, feeRate, unit)
	})
}

//CreateSummaryTx 创建汇总交易单
func (api *APINode) CreateSummaryTx(
	accountID string,
	sumAddress string,
	coin Coin,
	feeRate string,
	minTransfer string,
	retainedBalance string,
	addressStartIndex int,
	addressLimit int,
	confirms uint64,
	sid string,
	sync bool,
	reqFunc func(status uint64, msg string, rawTxs []*RawTransaction)) error {
	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}
	params := map[string]interface{}{
		"appID":             api.config.AppID,
		"accountID":         accountID,
		"address":           sumAddress,
		"coin":              coin,
		"minTransfer":       minTransfer,
		"retainedBalance":   retainedBalance,
		"feeRate":           feeRate,
		"addressStartIndex": addressStartIndex,
		"addressLimit":      addressLimit,
		"confirms":          confirms,
		"sid":               sid,
	}

	return api.node.Call(HostNodeID, "createSummaryTx", params, sync, func(resp owtp.Response) {

		data := resp.JsonData()
		rawTxs := make([]*RawTransaction, 0)
		if data.IsArray() {
			for _, jsonRawTx := range data.Array() {
				var rawTx RawTransaction
				json.Unmarshal([]byte(jsonRawTx.Raw), &rawTx)
				rawTxs = append(rawTxs, &rawTx)
			}
		}

		reqFunc(resp.Status, resp.Msg, rawTxs)
	})
}

//ServeTransmitNode 启动转发服务节点
func (api *APINode) ServeTransmitNode(address string) error {

	if api.transmitNode != nil {
		return fmt.Errorf("transmit node is inited")
	}

	transmitNode, err := NewTransmitNode(&APINodeConfig{
		Host:               address,
		ConnectType:        owtp.Websocket,
		AppID:              api.config.AppID,
		AppKey:             api.config.AppKey,
		Cert:               api.config.Cert,
		EnableSignature:    api.config.EnableSignature,
		EnableKeyAgreement: api.config.EnableKeyAgreement,
	})
	if err != nil {
		return nil
	}
	api.transmitNode = transmitNode
	api.transmitNode.Listen()

	return nil
}

//StopTransmitNode 停止转发服务节点
func (api *APINode) StopTransmitNode(port int) error {

	if api.transmitNode == nil {
		return fmt.Errorf("transmit node is not inited")
	}

	api.transmitNode.Close()
	api.transmitNode = nil

	return nil
}

//TransmitNode 转发节点
func (api *APINode) TransmitNode() (*TransmitNode, error) {
	if api.transmitNode == nil {
		return nil, fmt.Errorf("transmit node is not inited")
	}
	return api.transmitNode, nil
}
