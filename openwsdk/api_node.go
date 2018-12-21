package openwsdk

import (
	"encoding/json"
	"fmt"
	"github.com/blocktree/OpenWallet/crypto"
	"github.com/blocktree/OpenWallet/hdkeystore"
	"github.com/blocktree/OpenWallet/owtp"
	"time"
)

const (
	HostNodeID = "openw-server"
)

func init() {
	owtp.Debug = false
	//initAssetAdapter()
}

type APINodeConfig struct {
	Host   string           `json:"host"`
	AppID  string           `json:"appid"`
	AppKey string           `json:"appkey"`
	Cert   owtp.Certificate `json:"cert"`
	//ConnectType     string           `json:"connectType"`
	//EnableSignature bool             `json:"enableSignature"`
	//HostNodeID string           `json:"hostNodeID"`
}

//APINode APINode通信节点
type APINode struct {
	node   *owtp.OWTPNode
	config *APINodeConfig
}

//NewAPINode 创建API节点
func NewAPINode(config *APINodeConfig) *APINode {
	connectCfg := make(map[string]string)
	connectCfg["address"] = config.Host
	connectCfg["connectType"] = owtp.HTTP
	connectCfg["enableSignature"] = "1"

	node := owtp.NewOWTPNode(config.Cert, 0, 0)
	node.Connect(HostNodeID, connectCfg)
	api := APINode{
		node:   node,
		config: config,
	}
	return &api
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
func (api APINode) BindAppDevice() error {

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

	params := map[string]interface{}{
		"appID":  api.config.AppID,
		"offset": offset,
		"limit":  limit,
	}

	return api.node.Call(HostNodeID, "getSymbolList", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		symbols := make([]*Symbol, 0)
		symbolArray := data.Get("symbols").Array()
		for _, s := range symbolArray {
			var sym Symbol
			err := json.Unmarshal([]byte(s.Raw), &sym)
			if err == nil {
				symbols = append(symbols, &sym)
			}
		}

		reqFunc(resp.Status, resp.Msg, symbols)
	})
}

//CreateWallet 创建钱包
func (api *APINode) CreateWallet(alias, walletID string, sync bool, reqFunc func(status uint64, msg string, wallet *Wallet)) error {

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

//FindAccountByAccountID 通过资产账户ID获取资产账户信息
func (api *APINode) FindAccountByAccountID(accountID string, sync bool, reqFunc func(status uint64, msg string, account *Account)) error {

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

	params := map[string]interface{}{
		"appID":    api.config.AppID,
		"walletID": walletID,
	}

	return api.node.Call(HostNodeID, "findAccountByWalletID", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()

		var accounts []*Account
		accountArray := data.Array()
		for _, a := range accountArray {
			var acc Account
			err := json.Unmarshal([]byte(a.Raw), &acc)
			if err == nil {
				accounts = append(accounts, &acc)
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

	params := map[string]interface{}{
		"appID":     api.config.AppID,
		"walletID":  walletID,
		"accountID": accountID,
		"count":     count,
	}

	return api.node.Call(HostNodeID, "createAddress", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()

		var addresses []*Address
		addressArray := data.Array()
		for _, a := range addressArray {
			var addr Address
			err := json.Unmarshal([]byte(a.Raw), &addr)
			if err == nil {
				addresses = append(addresses, &addr)
			}
		}

		reqFunc(resp.Status, resp.Msg, addresses)
	})
}

//FindAddressByAddress 通获取具体交易地址信息
func (api *APINode) FindAddressByAddress(address string, sync bool, reqFunc func(status uint64, msg string, address *Address)) error {

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
func (api *APINode) FindAddressByAccountID(accountID string, sync bool, reqFunc func(status uint64, msg string, addresses []*Address)) error {

	params := map[string]interface{}{
		"appID":     api.config.AppID,
		"accountID": accountID,
	}

	return api.node.Call(HostNodeID, "findAddressByAccountID", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()

		var addresses []*Address
		array := data.Array()
		for _, a := range array {
			var addr Address
			err := json.Unmarshal([]byte(a.Raw), &addr)
			if err == nil {
				addresses = append(addresses, &addr)
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
	rawTx *RawTransaction,
	sync bool,
	reqFunc func(status uint64, msg string, successTx []*Transaction, failedRawTxs []*FailedRawTransaction),
) error {

	params := map[string]interface{}{
		"appID": api.config.AppID,
		"rawTx": []*RawTransaction{
			rawTx,
		},
	}

	return api.node.Call(HostNodeID, "submitTrade", params, sync, func(resp owtp.Response) {
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

	params := map[string]interface{}{
		"appID": api.config.AppID,
	}

	return api.node.Call(HostNodeID, "findTradeLog", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()

		var txs []*Transaction
		array := data.Array()
		for _, a := range array {
			var tx Transaction
			err := json.Unmarshal([]byte(a.Raw), &tx)
			if err == nil {
				txs = append(txs, &tx)
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

	params := map[string]interface{}{
		"appID":  api.config.AppID,
		"symbol": symbol,
		"offset": offset,
		"limit":  limit,
	}

	return api.node.Call(HostNodeID, "getContracts", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		tokens := make([]*TokenContract, 0)
		array := data.Get("symbols").Array()
		for _, s := range array {
			var t TokenContract
			err := json.Unmarshal([]byte(s.Raw), &t)
			if err == nil {
				tokens = append(tokens, &t)
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
