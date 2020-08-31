package openwsdk

import (
	"encoding/json"
	"fmt"
	"github.com/blocktree/openwallet/v2/common"
	"github.com/blocktree/openwallet/v2/crypto"
	"github.com/blocktree/openwallet/v2/hdkeystore"
	"github.com/blocktree/openwallet/v2/log"
	"github.com/blocktree/openwallet/v2/openwallet"
	"github.com/blocktree/openwallet/v2/owtp"
	"strconv"
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
}

//APINode APINode通信节点
type APINode struct {
	mu            sync.RWMutex //读写锁
	node          *owtp.OWTPNode
	config        *APINodeConfig
	observers     map[OpenwNotificationObject]bool //观察者
	transmitNode  *TransmitNode                    //钱包转发节点
	proxyNode     *ProxyNode                       //代理服务节点，用于转发请求到openw-server接口
	subscribeInfo *CallbackNode                    `json:"subscribeInfo"`
}

//NewAPINodeWithError 创建API节点
func NewAPINodeWithError(config *APINodeConfig) (*APINode, error) {
	connectCfg := owtp.ConnectConfig{}
	connectCfg.Address = config.Host
	connectCfg.ConnectType = config.ConnectType
	connectCfg.EnableSSL = config.EnableSSL
	connectCfg.EnableSignature = config.EnableSignature
	connectCfg.EnableKeyAgreement = config.EnableKeyAgreement
	node := owtp.NewNode(owtp.NodeConfig{
		Cert:       config.Cert,
		TimeoutSEC: config.TimeoutSEC,
	})
	_, err := node.Connect(HostNodeID, connectCfg)
	if err != nil {
		return nil, err
	}
	api := APINode{
		node:   node,
		config: config,
	}

	api.observers = make(map[OpenwNotificationObject]bool)

	//开启协商密码
	//if config.EnableKeyAgreement {
	//	if err := node.KeyAgreement(HostNodeID, "aes"); err != nil {
	//		log.Error(err)
	//		return nil, err
	//	}
	//}

	api.node.HandleFunc("checkNodeIsOnline", api.checkNodeIsOnline)
	api.node.HandleFunc("subscribeToAccount", api.subscribeToAccount)
	api.node.HandleFunc("subscribeToTrade", api.subscribeToTrade)
	api.node.HandleFunc("subscribeToBlock", api.subscribeToBlock)
	api.node.HandleFunc("subscribeToSmartContractReceipt", api.subscribeToSmartContractReceipt)

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

//OWTPNode
func (api *APINode) OWTPNode() *owtp.OWTPNode {
	if api == nil {
		return nil
	}
	return api.node
}

//NodeID
func (api *APINode) NodeID() string {
	if api == nil {
		return ""
	}
	return api.node.NodeID()
}

//Subscribe 订阅
func (api *APINode) Subscribe(subscribeMethod []string, listenAddr string, callbackMode int, callbackNode CallbackNode, subscribeToken string) error {

	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}

	//获取通知节点的NodeID
	_, notifierNodeID, err := api.GetNotifierNodeInfo()
	if err != nil {
		return err
	}

	//http不能用当前连接模式
	if callbackMode == CallbackModeCurrentConnection {
		if callbackNode.ConnectType != owtp.Websocket {
			return fmt.Errorf("%s can not use [SubscribeModeCurrentConnection]", callbackNode.ConnectType)
		}
	} else {

		if api.node.Listening(callbackNode.ConnectType) {
			return fmt.Errorf("subscribe connenct type [%s] is listening", callbackNode.ConnectType)
		}

		//开启监听
		log.Infof("%s start to listen [%s] connection...", listenAddr, callbackNode.ConnectType)
		api.node.Listen(owtp.ConnectConfig{
			Address:         listenAddr,
			ConnectType:     callbackNode.ConnectType,
			EnableSignature: callbackNode.EnableSignature,
		})

	}

	params := map[string]interface{}{
		//"subscriptions": subscriptions,
		"appID":           api.config.AppID,
		"subscribeMethod": subscribeMethod,
		"callbackMode":    callbackMode,
		"callbackNode":    callbackNode,
		"subscribeToken":  subscribeToken,
	}

	response, err := api.node.CallSync(HostNodeID, "subscribe", params)
	if err != nil {
		return err
	}

	if response.Status == owtp.StatusSuccess {

		api.subscribeInfo = &callbackNode
		api.subscribeInfo.notifierNodeID = notifierNodeID

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
func (api *APINode) GetSymbolList(symbol string, offset, limit, hasRole int, sync bool, reqFunc func(status uint64, msg string, total int, symbols []*Symbol)) error {

	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}

	params := map[string]interface{}{
		"appID":   api.config.AppID,
		"symbol":  symbol,
		"offset":  offset,
		"limit":   limit,
		"hasRole": hasRole,
	}

	return api.node.Call(HostNodeID, "getSymbolList", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		symbols := make([]*Symbol, 0)
		symbolArray := data.Get("symbols")
		total := data.Get("total").Int()
		if symbolArray.IsArray() {
			for _, s := range symbolArray.Array() {
				var sym Symbol
				err := json.Unmarshal([]byte(s.Raw), &sym)
				if err == nil {
					symbols = append(symbols, &sym)
				}
			}
		}

		reqFunc(resp.Status, resp.Msg, int(total), symbols)
	})
}

//CreateWallet 创建钱包
func (api *APINode) CreateWallet(wallet *Wallet, sync bool, reqFunc func(status uint64, msg string, wallet *Wallet)) error {

	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}

	params := map[string]interface{}{
		"appID":    api.config.AppID,
		"alias":    wallet.Alias,
		"walletID": wallet.WalletID,
		"rootPath": hdkeystore.OpenwCoinTypePath,
		"isTrust":  0,
	}

	if len(wallet.RootPath) > 0 {
		params["rootPath"] = wallet.RootPath
	}
	if len(wallet.AuthKey) > 0 {
		params["authKey"] = wallet.AuthKey
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
func (api *APINode) FindAccountByAccountID(accountID string, refresh int, sync bool, reqFunc func(status uint64, msg string, account *Account)) error {
	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}
	params := map[string]interface{}{
		"appID":     api.config.AppID,
		"accountID": accountID,
		"refresh":   refresh,
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
	extParam string,
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
		"extParam":  extParam,
	}

	return api.node.Call(HostNodeID, "createTrade", params, sync, func(resp owtp.Response) {

		if resp.Status != owtp.StatusSuccess {
			reqFunc(resp.Status, resp.Msg, nil)
			return
		}

		data := resp.JsonData()
		jsonRawTx := data.Get("rawTx")

		var rawTx RawTransaction
		err := json.Unmarshal([]byte(jsonRawTx.Raw), &rawTx)
		if err != nil {
			reqFunc(openwallet.ErrUnknownException, err.Error(), nil)
			return
		}

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

// new
func (api *APINode) FindTradeLogByParams(
	params map[string]interface{},
	sync bool,
	reqFunc func(status uint64, msg string, tx []*Transaction),
) error {
	params["appID"] = api.config.AppID
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

//FindTradeLog 获取转账交易订单日志
func (api *APINode) FindTradeLog(
	walletID string,
	accountID string,
	symbol string, // 主链币
	txid string,
	address string,
	isTmp int,
	orderType int,
	start_height int64,
	end_height int64,
	height int64,
	isDesc bool, // 是否倒序, 默认是
	offset int,
	limit int,
	sync bool,
	reqFunc func(status uint64, msg string, tx []*Transaction),
) error {
	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}
	sortby := -1
	if isDesc {
		sortby = 1
	}
	params := map[string]interface{}{
		"appID":        api.config.AppID,
		"walletID":     walletID,
		"accountID":    accountID,
		"symbol":       symbol,
		"txid":         txid,
		"isTmp":        isTmp,
		"orderType":    orderType,
		"sortby":       sortby,
		"start_height": start_height,
		"end_height":   end_height,
		"blockHeight":  height,
		"offset":       offset,
		"limit":        limit,
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
	symbol, contractID string,
	offset, limit int,
	sync bool,
	reqFunc func(status uint64, msg string, tokenContract []*TokenContract)) error {
	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}
	params := map[string]interface{}{
		"appID":      api.config.AppID,
		"symbol":     symbol,
		"contractID": contractID,
		"offset":     offset,
		"limit":      limit,
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
	reqFunc func(status uint64, msg string, balance *TokenBalance)) error {
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
		balance := NewTokenBalance(data)
		reqFunc(resp.Status, resp.Msg, balance)
	})
}

//GetAllTokenBalanceByAccount 获取账户所有token余额接口
func (api *APINode) GetAllTokenBalanceByAccount(
	accountID string,
	symbol string,
	sync bool,
	reqFunc func(status uint64, msg string, balance []*TokenBalance)) error {
	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}
	params := map[string]interface{}{
		"appID":     api.config.AppID,
		"accountID": accountID,
		"symbol":    symbol,
	}

	return api.node.Call(HostNodeID, "getAllTokenBalanceByAccount", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		balance := make([]*TokenBalance, 0)
		if data.IsArray() {
			for _, s := range data.Array() {
				t := NewTokenBalance(s)
				balance = append(balance, t)
			}
		}

		reqFunc(resp.Status, resp.Msg, balance)
	})
}

//GetAllTokenBalanceByAddress 获取地址的token余额接口
func (api *APINode) GetAllTokenBalanceByAddress(
	accountID string,
	address string,
	symbol string,
	sync bool,
	reqFunc func(status uint64, msg string, balance []*TokenBalance)) error {
	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}
	params := map[string]interface{}{
		"appID":     api.config.AppID,
		"accountID": accountID,
		"address":   address,
		"symbol":    symbol,
	}

	return api.node.Call(HostNodeID, "getAllTokenBalanceByAddress", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		balance := make([]*TokenBalance, 0)
		if data.IsArray() {
			for _, s := range data.Array() {
				t := NewTokenBalance(s)
				balance = append(balance, t)
			}
		}

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

//
func (api *APINode) GetFeeRateList(
	sync bool,
	reqFunc func(status uint64, msg string, feeRates []SupportFeeRate),
) error {
	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}
	params := map[string]interface{}{
		"appID":  api.config.AppID,
		"symbol": "",
	}
	return api.node.Call(HostNodeID, "getFeeRateList", params, sync, func(resp owtp.Response) {
		out := make([]SupportFeeRate, 0)
		data := resp.JsonData()
		if data.IsArray() {
			for _, d := range data.Array() {
				out = append(out, SupportFeeRate{
					FeeRate: d.Get("feeRate").String(),
					Symbol:  d.Get("symbol").String(),
					Unit:    d.Get("unit").String(),
				})
			}
		}
		reqFunc(resp.Status, resp.Msg, out)
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
	feesSupportAccount *FeesSupportAccount,
	memo string,
	sync bool,
	reqFunc func(status uint64, msg string, rawTxs []*RawTransaction)) error {
	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}
	params := map[string]interface{}{
		"appID":              api.config.AppID,
		"accountID":          accountID,
		"address":            sumAddress,
		"coin":               coin,
		"minTransfer":        minTransfer,
		"retainedBalance":    retainedBalance,
		"feeRate":            feeRate,
		"addressStartIndex":  addressStartIndex,
		"addressLimit":       addressLimit,
		"confirms":           confirms,
		"sid":                sid,
		"feesSupportAccount": feesSupportAccount,
		"memo":               memo,
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
	transmitNode.parent = api
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

//GetSymbolBlockList 获取币种最大高度
func (api *APINode) GetSymbolBlockList(
	symbol string,
	sync bool,
	reqFunc func(status uint64, msg string, blockHeaders []*BlockHeader)) error {
	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}
	params := map[string]interface{}{
		"appID":  api.config.AppID,
		"symbol": symbol,
	}

	return api.node.Call(HostNodeID, "getSymbolBlockList", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		headers := make([]*BlockHeader, 0)
		if data.IsArray() {
			for _, d := range data.Array() {
				headers = append(headers, NewBlockHeader(d))
			}
		}

		reqFunc(resp.Status, resp.Msg, headers)
	})
}

// ImportAccount 导入第三方资产账户
func (api *APINode) ImportAccount(
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
	}

	return api.node.Call(HostNodeID, "importAccount", params, sync, func(resp owtp.Response) {
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

// BindDevice 绑定设备ID
func (api *APINode) BindDevice(
	deviceID string,
	sync bool,
	reqFunc func(status uint64, msg string)) error {
	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}
	appID := api.config.AppID
	appKey := api.config.AppKey
	accessTime := time.Now().UnixNano() / 1e6
	t := strconv.FormatInt(accessTime, 10)
	sigStr := appID + "." + deviceID + "." + t + "." + appKey
	params := map[string]interface{}{
		"appID":      appID,
		"deviceID":   deviceID,
		"accessTime": accessTime,
		"sign":       crypto.GetMD5(sigStr),
	}
	return api.node.Call(HostNodeID, "bindAppDevice", params, sync, func(resp owtp.Response) {
		reqFunc(resp.Status, resp.Msg)
	})
}

// ImportBatchAddress 批量导入地址
func (api *APINode) ImportBatchAddress(
	walletID, accountID, memo string,
	addressAndPubs map[string]string,
	updateBalance bool,
	sync bool,
	reqFunc func(status uint64, msg string, importAddresses []string)) error {
	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}

	addresses := make([]string, 0)
	publickeys := make([]string, 0)
	for addr, pub := range addressAndPubs {
		addresses = append(addresses, addr)
		publickeys = append(publickeys, pub)
	}

	appID := api.config.AppID
	params := map[string]interface{}{
		"appID":         appID,
		"walletID":      walletID,
		"accountID":     accountID,
		"memo":          memo,
		"addresses":     addresses,
		"publicKeys":    publickeys,
		"updateBalance": common.NewString(updateBalance).Int64(),
	}
	return api.node.Call(HostNodeID, "importBatchAddress", params, sync, func(resp owtp.Response) {

		data := resp.JsonData()

		var importAddresses []string
		array := data
		if array.IsArray() {
			for _, a := range array.Array() {
				importAddresses = append(importAddresses, a.String())
			}
		}

		reqFunc(resp.Status, resp.Msg, importAddresses)
	})
}

// GetNotifierNodeInfo 获取通知者节点信息
func (api *APINode) GetNotifierNodeInfo() (string, string, error) {
	if api == nil {
		return "", "", fmt.Errorf("APINode is not inited")
	}

	var (
		pubKey     string
		nodeId     string
		requestErr *openwallet.Error
	)
	appID := api.config.AppID
	time := time.Now().UnixNano()
	plainText := fmt.Sprintf("%s%d%s", appID, time, api.config.AppKey)
	sign := crypto.GetMD5(plainText)

	params := map[string]interface{}{
		"appID": appID,
		"time":  time,
		"sign":  sign,
	}

	err := api.node.Call(HostNodeID, "getNodeInfo", params, true, func(resp owtp.Response) {

		if resp.Status != owtp.StatusSuccess {
			requestErr = openwallet.Errorf(resp.Status, resp.Msg)
			return
		}

		data := resp.JsonData()
		pubKey = data.Get("pubKey").String()
		nodeId = data.Get("nodeId").String()
	})
	if err != nil {
		return "", "", err
	}

	if requestErr != nil {
		return "", "", requestErr
	}

	return pubKey, nodeId, nil
}

// FindWalletByParams 查询钱包列表
func (api *APINode) FindWalletByParams(
	params map[string]interface{},
	offset int,
	limit int,
	sync bool,
	reqFunc func(status uint64, msg string, wallets []*Wallet),
) error {
	if params == nil {
		params = make(map[string]interface{})
	}

	params["appID"] = api.config.AppID
	params["offset"] = offset
	params["limit"] = limit
	return api.node.Call(HostNodeID, "findWalletByParams", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		var wallets []*Wallet
		array := data
		if array.IsArray() {
			for _, a := range array.Array() {
				var wallet Wallet
				err := json.Unmarshal([]byte(a.Raw), &wallet)
				if err == nil {
					wallets = append(wallets, &wallet)
				}
			}
		}
		reqFunc(resp.Status, resp.Msg, wallets)
	})
}

// FindAccountByParams 根据条件查询账户列表
func (api *APINode) FindAccountByParams(
	params map[string]interface{},
	offset int,
	limit int,
	sync bool,
	reqFunc func(status uint64, msg string, accounts []*Account),
) error {

	if params == nil {
		params = make(map[string]interface{})
	}

	params["appID"] = api.config.AppID
	params["offset"] = offset
	params["limit"] = limit
	return api.node.Call(HostNodeID, "findAccountByParams", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		var accounts []*Account
		array := data
		if array.IsArray() {
			for _, a := range array.Array() {
				var account Account
				err := json.Unmarshal([]byte(a.Raw), &account)
				if err == nil {
					accounts = append(accounts, &account)
				}
			}
		}
		reqFunc(resp.Status, resp.Msg, accounts)
	})
}

// FindAddressByParams 过条件查询地址列表
func (api *APINode) FindAddressByParams(
	params map[string]interface{},
	offset int,
	limit int,
	sync bool,
	reqFunc func(status uint64, msg string, addresses []*Address),
) error {
	if params == nil {
		params = make(map[string]interface{})
	}

	params["appID"] = api.config.AppID
	params["offset"] = offset
	params["limit"] = limit
	return api.node.Call(HostNodeID, "findAddressByParams", params, sync, func(resp owtp.Response) {
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

// VerifyAddress 地址校验
func (api *APINode) VerifyAddress(symbol, address string, sync bool,
	reqFunc func(status uint64, msg string, flag bool),
) error {
	params := make(map[string]interface{})

	params["appID"] = api.config.AppID
	params["symbol"] = symbol
	params["address"] = address
	return api.node.Call(HostNodeID, "verifyAddress", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		reqFunc(resp.Status, resp.Msg, data.Bool())
	})
}

// CallSmartContractABI 调用智能合约ABI方法
func (api *APINode) CallSmartContractABI(
	accountID string,
	coin Coin,
	abiParam []string,
	raw string,
	rawType uint64,
	sync bool, reqFunc func(status uint64, msg string, callResult *SmartContractCallResult),
) error {
	params := make(map[string]interface{})

	params["appID"] = api.config.AppID
	params["accountID"] = accountID
	params["coin"] = coin
	params["abiParam"] = abiParam
	params["raw"] = raw
	params["rawType"] = rawType
	return api.node.Call(HostNodeID, "callSmartContractABI", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		var callResult SmartContractCallResult
		err := json.Unmarshal([]byte(data.Raw), &callResult)
		if err != nil {
			reqFunc(openwallet.ErrUnknownException, err.Error(), nil)
			return
		}
		reqFunc(resp.Status, resp.Msg, &callResult)
	})
}

// CreateSmartContractTrade 创建智能合约交易单
// @param sid 必填 业务编号
// @param accountID 必填 账户ID
// @param coin 必填 币种信息
// @param abiParam 可选 ABI参数组
// @param raw 可选 原始交易单
// @param rawType 可选 原始交易单编码类型，0：hex字符串，1：json字符串，2：base64字符串
// @param feeRate 可选 自定义手续费率
// @param value 可选 主币数量
// @param sync 必填 是否同步线程
// @param reqFunc 必填 回调函数处理
func (api *APINode) CreateSmartContractTrade(
	sid string,
	accountID string,
	coin Coin,
	abiParam []string,
	raw string,
	rawType uint64,
	feeRate string,
	value string,
	sync bool, reqFunc func(status uint64, msg string, rawTx *SmartContractRawTransaction),
) error {
	params := make(map[string]interface{})

	params["appID"] = api.config.AppID
	params["sid"] = sid
	params["accountID"] = accountID
	params["coin"] = coin
	params["abiParam"] = abiParam
	params["raw"] = raw
	params["rawType"] = rawType
	params["feeRate"] = feeRate
	params["value"] = value
	return api.node.Call(HostNodeID, "createSmartContractTrade", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		var rawTx SmartContractRawTransaction
		err := json.Unmarshal([]byte(data.Raw), &rawTx)
		if err != nil {
			reqFunc(openwallet.ErrUnknownException, err.Error(), nil)
			return
		}
		reqFunc(resp.Status, resp.Msg, &rawTx)
	})
}

//SubmitSmartContractTrade 广播转账交易订单
func (api *APINode) SubmitSmartContractTrade(
	rawTx []*SmartContractRawTransaction,
	sync bool,
	reqFunc func(status uint64, msg string, successTx []*SmartContractReceipt, failedRawTxs []*FailureSmartContractLog),
) error {
	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}
	params := map[string]interface{}{
		"appID": api.config.AppID,
		"rawTx": rawTx,
	}

	return api.node.Call(HostNodeID, "submitSmartContractTrade", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		failedRawTxs := make([]*FailureSmartContractLog, 0)
		failedArray := data.Get("failure")
		if failedArray.IsArray() {
			for _, failed := range failedArray.Array() {
				var rawTx SmartContractRawTransaction
				err := json.Unmarshal([]byte(failed.Get("rawTx").Raw), &rawTx)
				if err == nil {
					failedRawTx := &FailureSmartContractLog{
						Reason: failed.Get("error").String(),
						RawTx:  &rawTx,
					}

					failedRawTxs = append(failedRawTxs, failedRawTx)
				}

			}
		}

		var txs []*SmartContractReceipt
		successArray := data.Get("success")
		if successArray.IsArray() {
			for _, a := range successArray.Array() {
				var tx SmartContractReceipt
				err := json.Unmarshal([]byte(a.Raw), &tx)
				if err == nil {
					txs = append(txs, &tx)
				}
			}
		}

		reqFunc(resp.Status, resp.Msg, txs, failedRawTxs)
	})
}

// FindSmartContractReceiptByParams 获取智能合约交易回执
func (api *APINode) FindSmartContractReceiptByParams(
	params map[string]interface{},
	sync bool,
	reqFunc func(status uint64, msg string, receipts []*SmartContractReceipt),
) error {
	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}
	params["appID"] = api.config.AppID
	return api.node.Call(HostNodeID, "findSmartContractReceipt", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		var receipts []*SmartContractReceipt
		array := data
		if array.IsArray() {
			for _, a := range array.Array() {
				var receipt SmartContractReceipt
				err := json.Unmarshal([]byte(a.Raw), &receipt)
				if err == nil {
					receipts = append(receipts, &receipt)
				}
			}
		}
		reqFunc(resp.Status, resp.Msg, receipts)
	})

}

// FollowSmartContractReceipt 订阅要关注智能合约回执通知
func (api *APINode) FollowSmartContractReceipt(
	followContracts []string,
	sync bool,
	reqFunc func(status uint64, msg string),
) error {
	if api == nil {
		return fmt.Errorf("APINode is not inited")
	}
	params := map[string]interface{}{
		"appID":           api.config.AppID,
		"followContracts": followContracts,
	}
	return api.node.Call(HostNodeID, "followSmartContractReceipt", params, sync, func(resp owtp.Response) {
		reqFunc(resp.Status, resp.Msg)
	})

}
