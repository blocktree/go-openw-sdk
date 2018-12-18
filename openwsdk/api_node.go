package openwsdk

import (
	"encoding/json"
	"fmt"
	"github.com/blocktree/OpenWallet/crypto"
	"github.com/blocktree/OpenWallet/owtp"
	"time"
)

func init() {
	owtp.Debug = false
}

type APINodeConfig struct {
	Address         string           `json:"address"`
	ConnectType     string           `json:"connectType"`
	EnableSignature bool             `json:"enableSignature"`
	AppID           string           `json:"appid"`
	AppKey          string           `json:"appkey"`
	Cert            owtp.Certificate `json:"cert"`
	HostNodeID      string           `json:"hostNodeID"`
}

//APINode APINode通信节点
type APINode struct {
	node   *owtp.OWTPNode
	config *APINodeConfig
}

//NewAPINode 创建API节点
func NewAPINode(config *APINodeConfig) *APINode {
	connectCfg := make(map[string]string)
	connectCfg["address"] = config.Address
	connectCfg["connectType"] = config.ConnectType
	if config.EnableSignature {
		connectCfg["enableSignature"] = "1"
	}

	node := owtp.NewOWTPNode(config.Cert, 0, 0)
	node.Connect(config.HostNodeID, connectCfg)
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

	response, err := api.node.CallSync(api.config.HostNodeID, "bindAppDevice", params)
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
func (api *APINode) GetSymbolList(offset, limit uint64, sync bool, reqFunc func(status uint64, msg string, symbols []*Symbol)) error {

	params := map[string]interface{}{
		"appID":  api.config.AppID,
		"offset": offset,
		"limit":  limit,
	}

	return api.node.Call(api.config.HostNodeID, "getSymbolList", params, sync, func(resp owtp.Response) {
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
func (api *APINode) CreateWallet(alias, walletID, rootPath string, sync bool, reqFunc func(status uint64, msg string, wallet *Wallet)) error {

	params := map[string]interface{}{
		"appID":    api.config.AppID,
		"alias":    alias,
		"walletID": walletID,
		"rootPath": rootPath,
		"isTrust":  0,
	}

	return api.node.Call(api.config.HostNodeID, "createWallet", params, sync, func(resp owtp.Response) {
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

	return api.node.Call(api.config.HostNodeID, "findWalletByWalletID", params, sync, func(resp owtp.Response) {
		data := resp.JsonData()
		var wallet Wallet
		json.Unmarshal([]byte(data.Raw), &wallet)
		reqFunc(resp.Status, resp.Msg, &wallet)
	})
}

//CreateAccount 创建资产账户
func (api *APINode) CreateAccount(
	alias string,
	walletID string,
	accountID string,
	symbol string,
	publicKey string,
	accountIndex string,
	hdPath string,
	otherOwnerKeys []string,
	reqSigs uint64,
	sync bool,
	reqFunc func(status uint64, msg string, account *Account, addresses []*Address)) error {

	params := map[string]interface{}{
		"appID":          api.config.AppID,
		"alias":          alias,
		"walletID":       walletID,
		"accountID":      accountID,
		"symbol":         symbol,
		"publicKey":      publicKey,
		"accountIndex":   accountIndex,
		"hdPath":         hdPath,
		"otherOwnerKeys": otherOwnerKeys,
		"reqSigs":        reqSigs,
		"isTrust":        0,
	}

	return api.node.Call(api.config.HostNodeID, "createAccount", params, sync, func(resp owtp.Response) {
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
		"appID":    api.config.AppID,
		"accountID": accountID,
	}

	return api.node.Call(api.config.HostNodeID, "findAccountByAccountID", params, sync, func(resp owtp.Response) {
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

	return api.node.Call(api.config.HostNodeID, "findAccountByWalletID", params, sync, func(resp owtp.Response) {
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
