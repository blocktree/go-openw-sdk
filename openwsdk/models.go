package openwsdk

import (
	"encoding/json"
	"fmt"
	"github.com/blocktree/openwallet/hdkeystore"
	"github.com/blocktree/openwallet/openwallet"
	"github.com/tidwall/gjson"
)

type BlockHeader struct {
	Hash              string `json:"hash"`
	Confirmations     uint64 `json:"confirmations"`
	Merkleroot        string `json:"merkleroot"`
	Previousblockhash string `json:"previousblockhash"`
	Height            uint64 `json:"height"`
	Version           uint64 `json:"version"`
	Time              uint64 `json:"time"`
	Fork              bool   `json:"fork"`
	Symbol            string `json:"symbol"`
}

type CallbackNode struct {
	NodeID             string `json:"nodeID"`             //@required 节点ID
	Address            string `json:"address"`            //@required 连接IP地址
	ConnectType        string `json:"connectType"`        //@required 连接方式
	EnableKeyAgreement bool   `json:"enableKeyAgreement"` //是否开启owtp协议协商密码
	EnableSSL          bool   `json:"enableSSL"`          //是否开启链接SSL，https，wss
}

func NewCallbackNode(result gjson.Result) *CallbackNode {
	obj := &CallbackNode{
		NodeID:             result.Get("nodeID").String(),
		Address:            result.Get("address").String(),
		ConnectType:        result.Get("connectType").String(),
		EnableKeyAgreement: result.Get("enableKeyAgreement").Bool(),
		EnableSSL:          result.Get("enableSSL").Bool(),
	}
	return obj
}

type TrustNodeInfo struct {
	NodeID      string `json:"nodeID"` //@required 节点ID
	NodeName    string `json:"nodeName"`
	ConnectType string `json:"connectType"`
}

//SummarySetting 汇总设置信息
type SummarySetting struct {
	WalletID        string `json:"walletID"`
	AccountID       string `json:"accountID" storm:"id"`
	SumAddress      string `json:"sumAddress"`
	Threshold       string `json:"threshold"`
	MinTransfer     string `json:"minTransfer"`
	RetainedBalance string `json:"retainedBalance"`
	Confirms        uint64 `json:"confirms"`
}

func NewSummarySetting(result gjson.Result) *SummarySetting {
	obj := &SummarySetting{
		WalletID:        result.Get("walletID").String(),
		AccountID:       result.Get("accountID").String(),
		SumAddress:      result.Get("sumAddress").String(),
		Threshold:       result.Get("threshold").String(),
		MinTransfer:     result.Get("minTransfer").String(),
		RetainedBalance: result.Get("retainedBalance").String(),
		Confirms:        result.Get("confirms").Uint(),
	}
	return obj
}

type Wallet struct {
	AppID        string `json:"appID" bson:"appID"`
	WalletID     string `json:"walletID" bson:"walletID"`
	Alias        string `json:"alias" bson:"alias"`
	IsTrust      int64  `json:"isTrust" bson:"isTrust"`
	PasswordType int64  `json:"passwordType" bson:"passwordType"`
	Password     string `json:"password" bson:"password"`
	AuthKey      string `json:"authKey" bson:"authKey"`
	RootPath     string `json:"rootPath" bson:"rootPath"`
	AccountIndex int64  `json:"accountIndex" bson:"accountIndex"`
	Keystore     string `json:"keystore" bson:"keystore"`
	Applytime    int64  `json:"applytime" bson:"applytime"`
	Dealstate    int64  `json:"dealstate" bson:"dealstate"`
}

type Symbol struct {
	Name     string `json:"name" bson:"name" storm:"id"`
	Coin     string `json:"coin" bson:"coin"`
	Curve    int64  `json:"curve" bson:"curve"`
	Orderno  int64  `json:"orderno" bson:"orderno"`
	Confirm  int64  `json:"confirm" bson:"confirm"`
	Decimals int64  `json:"decimals" bson:"decimals"`
}

type Account struct {
	AppID           string   `json:"appID" bson:"appID"`
	WalletID        string   `json:"walletID" bson:"walletID"`
	AccountID       string   `json:"accountID" bson:"accountID"`
	Alias           string   `json:"alias" bson:"alias"`
	Symbol          string   `json:"symbol" bson:"symbol"`
	OtherOwnerKeys  []string `json:"otherOwnerKeys" bson:"otherOwnerKeys"`
	ReqSigs         int64    `json:"reqSigs" bson:"reqSigs"`
	IsTrust         int64    `json:"isTrust" bson:"isTrust"`
	Password        string   `json:"password" bson:"password"`
	PublicKey       string   `json:"publicKey" bson:"publicKey"`
	HdPath          string   `json:"hdPath" bson:"hdPath"`
	ContractAddress string   `json:"contractAddress" bson:"contractAddress"`
	AccountIndex    int64    `json:"accountIndex" bson:"accountIndex"`
	Balance         string   `json:"balance" bson:"balance"`
	ExtInfo         string   `json:"extInfo" bson:"extInfo"`
	AddressIndex    int64    `json:"addressIndex" bson:"addressIndex"`
	Applytime       int64    `json:"applytime" bson:"applytime"`
	Dealstate       int64    `json:"dealstate" bson:"dealstate"`
}

type Address struct {
	AppID     string `json:"appID" bson:"appID"`
	WalletID  string `json:"walletID" bson:"walletID"`
	AccountID string `json:"accountID" bson:"accountID"`
	Alias     string `json:"alias" bson:"alias"`
	Symbol    string `json:"symbol" bson:"symbol"`
	AddrIndex int64  `json:"addrIndex" bson:"addrIndex"`
	Address   string `json:"address" bson:"address"`
	Balance   string `json:"balance" bson:"balance"`
	IsMemo    int64  `json:"isMemo" bson:"isMemo"`
	Memo      string `json:"memo" bson:"memo"`
	WatchOnly int64  `json:"watchOnly" bson:"watchOnly"`
	PublicKey string `json:"publicKey" bson:"publicKey"`
	CreatedAt int64  `json:"createdAt" bson:"createdAt"`
	Num       int64  `json:"num" bson:"num"`
	Tag       string `json:"tag" bson:"tag"`
	HdPath    string `json:"hdPath" bson:"hdPath"`
	IsChange  int64  `json:"isChange" bson:"isChange"`
	Applytime int64  `json:"applytime" bson:"applytime"`
	Dealstate int64  `json:"dealstate" bson:"dealstate"`
}

type TokenContract struct {
	ContractID string `json:"contractID" bson:"contractID" storm:"id"`
	Symbol     string `json:"symbol" bson:"symbol"` //主链标记
	Name       string `json:"name" bson:"name"`
	Decimals   int64  `json:"decimals" bson:"decimals"`
	Address    string `json:"address" bson:"address"`
	Token      string `json:"token" bson:"token"` //token标记
	Protocol   string `json:"protocol" bson:"protocol"`
}

type Coin struct {
	Symbol     string `json:"symbol"`
	IsContract bool   `json:"isContract"`
	ContractID string `json:"contractID"`
}

func NewCoin(result gjson.Result) *Coin {
	obj := &Coin{
		Symbol:     result.Get("symbol").String(),
		IsContract: result.Get("isContract").Bool(),
		ContractID: result.Get("contractID").String(),
	}
	return obj
}

type RawTransaction struct {
	Coin       Coin                       `json:"coin"`      //@required 区块链类型标识
	TxID       string                     `json:"txID"`      //交易单ID，广播后会生成
	Sid        string                     `json:"sid"`       //业务订单号，保证业务不重复交易而用
	RawHex     string                     `json:"rawHex"`    //区块链协议构造的交易原生数据
	FeeRate    string                     `json:"feeRate"`   //自定义费率
	To         map[string]string          `json:"to"`        //@required 目的地址:转账数量
	AccountID  string                     `json:"accountID"` //@required 创建交易单的账户
	Signatures map[string][]*KeySignature `json:"sigParts"`  //拥有者accountID: []未花签名
	Required   uint64                     `json:"reqSigs"`   //必要签名
	Fees       string                     `json:"fees"`      //手续费
}

type KeySignature struct {
	EccType     uint32 `json:"eccType"`     //曲线类型
	Nonce       string `json:"nonce"`       //nonce
	Address     string `json:"address"`     //提供签名的地址
	Signature   string `json:"signed"`      //未花签名
	Message     string `json:"msg"`         //被签消息
	DerivedPath string `json:"derivedPath"` //密钥路径
	WalletID    string `json:"walletID"`    //钱包ID
	InputIndex  uint32 `json:"inputIndex"`  //input索引位
}

type Transaction struct {
	AppID        string                 `json:"appID" bson:"appID"`
	WalletID     string                 `json:"walletID" bson:"walletID"`
	AccountID    string                 `json:"accountID" bson:"accountID"`
	Sid          string                 `json:"sid" bson:"sid"`
	Txid         string                 `json:"txid" bson:"txid"`
	Wxid         string                 `json:"wxid" bson:"wxid"`
	FromAddress  []string               `json:"fromAddress" bson:"fromAddress"`
	FromAddressV []string               `json:"fromAddressV" bson:"fromAddressV"`
	ToAddress    []string               `json:"toAddress" bson:"toAddress"`
	ToAddressV   []string               `json:"toAddressV" bson:"toAddressV"`
	Amount       string                 `json:"amount" bson:"amount"`
	Fees         string                 `json:"fees" bson:"fees"`
	Type         int64                  `json:"type" bson:"type"`
	Symbol       string                 `json:"symbol" bson:"symbol"`
	ContractID   string                 `json:"contractID" bson:"contractID"`
	IsContract   int64                  `json:"isContract" bson:"isContract"`
	Confirm      int64                  `json:"confirm" bson:"confirm"`
	BlockHash    string                 `json:"blockHash" bson:"blockHash"`
	BlockHeight  int64                  `json:"blockHeight" bson:"blockHeight"`
	IsMemo       int64                  `json:"isMemo" bson:"isMemo"`
	IsMain       int64                  `json:"isMain" bson:"isMain"`
	Memo         string                 `json:"memo" bson:"memo"`
	Applytime    int64                  `json:"applytime" bson:"applytime"`
	SubmitTime   int64                  `json:"submitTime" bson:"submitTime"`
	ConfirmTime  int64                  `json:"confirmTime" bson:"confirmTime"`
	Decimals     int64                  `json:"decimals" bson:"decimals"`
	Succtime     int64                  `json:"succtime" bson:"succtime"`
	Dealstate    int64                  `json:"dealstate" bson:"dealstate"`
	Notifystate  int64                  `json:"notifystate" bson:"notifystate"`
	ContractID2  string                 `json:"contractID2" bson:"contractID2"`
	ContractName string                 `json:"contractName" bson:"contractName"`
	ContractAddr string                 `json:"contractAddr" bson:"contractAddr"`
	Contract     map[string]interface{} `json:"contract" bson:"contract"`
}

type FailedRawTransaction struct {
	RawTx  *RawTransaction `json:"rawTx"`
	Reason string          `json:"error"`
}

type SummaryTask struct {
	Wallets []*SummaryWalletTask `json:"wallets"`
}

func NewSummaryTask(result gjson.Result) *SummaryTask {
	var obj SummaryTask
	json.Unmarshal([]byte(result.Raw), &obj)
	return &obj
}

type SummaryAccountTask struct {
	AccountID string   `json:"accountID"`
	Contracts []string `json:"contracts"`
	FeeRate   string   `json:"feeRate"`
}

type SummaryWalletTask struct {
	WalletID string                `json:"walletID"`
	Password string                `json:"password"`
	Accounts []*SummaryAccountTask `json:"accounts"`
	Wallet   *Wallet
}

func NewSummaryWalletTask(result gjson.Result) *SummaryWalletTask {
	var obj SummaryWalletTask
	json.Unmarshal([]byte(result.Raw), &obj)
	return &obj
}

/*
{
	"wallets": [
		{
			"walletID": "1234qwer",
			"password": "12345678",
			"accounts": [
				{
					"accountID": "123",
					"feeRate": "0.0001"
					"contracts":[
						"all", //全部合约
						"0x1234567890abcdef", //指定的合约地址
					]
				},
			],
		},
	]
}
*/

type SummaryTaskLog struct {
	Sid            string   `json:"sid" storm:"id"`           //汇总执行批次号
	WalletID       string   `json:"walletID"`                 //汇总钱包ID
	AccountID      string   `json:"accountID"`                //汇总资产账户ID
	StartAddrIndex int      `json:"startAddrIndex"`           //账户汇总起始的地址索引位
	EndAddrIndex   int      `json:"endAddrIndex"`             //账户汇总结束的地址索引位
	Coin           Coin     `json:"coin"`                     //汇总的币种信息
	SuccessCount   int      `json:"successCount"`             //汇总交易发送的成功个数
	FailCount      int      `json:"failCount"`                //汇总交易发送的失败个数
	TxIDs          []string `json:"txIDs"`                    //汇总交易成功的txid
	TotalSumAmount string   `json:"sumAmount"`                //这次汇总总数
	TotalCostFees  string   `json:"sumFees"`                  //这次汇总总消费手续费
	CreateTime     int64    `json:"createTime" storm:"index"` //汇总时间
}

func (wallet *Wallet) CreateAccount(alias string, symbol *Symbol, key *hdkeystore.HDKey) (*Account, error) {

	var (
		account = &Account{}
	)

	account.Alias = alias
	account.Symbol = symbol.Coin
	account.ReqSigs = 1

	newAccIndex := wallet.AccountIndex + 1

	// root/n' , 使用强化方案
	account.HdPath = fmt.Sprintf("%s/%d'", wallet.RootPath, newAccIndex)

	childKey, err := key.DerivedKeyWithPath(account.HdPath, uint32(symbol.Curve))
	if err != nil {
		return nil, err
	}

	account.PublicKey = childKey.GetPublicKey().OWEncode()
	account.AccountIndex = newAccIndex
	account.AccountID = openwallet.GenAccountID(account.PublicKey)
	account.AddressIndex = -1
	account.WalletID = wallet.WalletID

	return account, nil

}
