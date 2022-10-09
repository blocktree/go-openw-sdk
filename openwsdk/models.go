package openwsdk

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/blocktree/openwallet/v2/crypto"
	"github.com/blocktree/openwallet/v2/hdkeystore"
	"github.com/blocktree/openwallet/v2/openwallet"
	"github.com/tidwall/gjson"
	"time"
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

func NewBlockHeader(result gjson.Result) *BlockHeader {
	obj := &BlockHeader{
		Hash:              result.Get("hash").String(),
		Confirmations:     result.Get("confirmations").Uint(),
		Merkleroot:        result.Get("merkleroot").String(),
		Previousblockhash: result.Get("previousblockhash").String(),
		Height:            result.Get("height").Uint(),
		Version:           result.Get("version").Uint(),
		Time:              result.Get("time").Uint(),
		Fork:              result.Get("fork").Bool(),
		Symbol:            result.Get("symbol").String(),
	}
	return obj
}

type CallbackNode struct {
	NodeID             string `json:"nodeID"`             //@required 节点ID
	Address            string `json:"address"`            //@required 连接IP地址
	ConnectType        string `json:"connectType"`        //@required 连接方式
	EnableKeyAgreement bool   `json:"enableKeyAgreement"` //是否开启owtp协议协商密码
	EnableSSL          bool   `json:"enableSSL"`          //是否开启链接SSL，https，wss
	EnableSignature    bool   `json:"enableSignature"`    //是否开启owtp协议内签名，防重放
	notifierNodeID     string `json:"notifierNodeID"`     //通知者节点ID
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
	Version     string `json:"version"`
	GitRev      string `json:"gitRev"`
	BuildTime   string `json:"buildTime"`
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
	AddressLimit    uint64 `json:"addressLimit"`
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
	Name         string `json:"name" bson:"name" storm:"id"`
	MainSymbol   string `json:"mainSymbol" bson:"mainSymbol"`
	Symbol       string `json:"symbol" bson:"symbol"`
	Curve        int64  `json:"curve" bson:"curve"`
	OrderNo      int64  `json:"orderNo" bson:"orderNo"`
	Confirm      int64  `json:"confirm" bson:"confirm"`
	Decimals     int64  `json:"decimals" bson:"decimals"`
	BalanceMode  uint64 `json:"balanceMode" bson:"balanceMode"`
	Icon         string `json:"icon"`
	SupportMemo  uint64 `json:"supportMemo"`  //交易是否支持memo, 0: false, 1: true
	OnlyContract uint64 `json:"onlyContract"` //支持合约代币, 0: false, 1: true
	WithdrawStop int64  `json:"withdrawStop"`
	BlockStop    int64  `json:"blockStop"`
	MaxHeight    int64  `json:"maxHeight"`
	FeeRate      string `json:"feeRate"`
	Unit         string `json:"unit"`
}

type Account struct {
	AppID            string   `json:"appID" bson:"appID"`
	WalletID         string   `json:"walletID" bson:"walletID"`
	AccountID        string   `json:"accountID" bson:"accountID"`
	Alias            string   `json:"alias" bson:"alias"`
	Symbol           string   `json:"symbol" bson:"symbol"`
	OtherOwnerKeys   []string `json:"otherOwnerKeys" bson:"otherOwnerKeys"`
	ReqSigs          int64    `json:"reqSigs" bson:"reqSigs"`
	IsTrust          int64    `json:"isTrust" bson:"isTrust"`
	Password         string   `json:"password" bson:"password"`
	PublicKey        string   `json:"publicKey" bson:"publicKey"`
	HdPath           string   `json:"hdPath" bson:"hdPath"`
	ContractAddress  string   `json:"contractAddress" bson:"contractAddress"`
	AccountIndex     int64    `json:"accountIndex" bson:"accountIndex"`
	Balance          string   `json:"balance" bson:"balance"`
	ConfirmBalance   string   `json:"confirmBalance" bson:"confirmBalance"`
	UnconfirmBalance string   `json:"unconfirmBalance" bson:"unconfirmBalance"`
	ExtInfo          string   `json:"extInfo" bson:"extInfo"`
	AddressIndex     int64    `json:"addressIndex" bson:"addressIndex"`
	Applytime        int64    `json:"applytime" bson:"applytime"`
	Dealstate        int64    `json:"dealstate" bson:"dealstate"`
}

type Address struct {
	AppID            string `json:"appID" bson:"appID"`
	WalletID         string `json:"walletID" bson:"walletID"`
	AccountID        string `json:"accountID" bson:"accountID"`
	Alias            string `json:"alias" bson:"alias"`
	Symbol           string `json:"symbol" bson:"symbol"`
	AddrIndex        int64  `json:"addrIndex" bson:"addrIndex"`
	Address          string `json:"address" bson:"address"`
	Balance          string `json:"balance" bson:"balance"`
	ConfirmBalance   string `json:"confirmBalance" bson:"confirmBalance"`
	UnconfirmBalance string `json:"unconfirmBalance" bson:"unconfirmBalance"`
	IsMemo           int64  `json:"isMemo" bson:"isMemo"`
	Memo             string `json:"memo" bson:"memo"`
	WatchOnly        int64  `json:"watchOnly" bson:"watchOnly"`
	PublicKey        string `json:"publicKey" bson:"publicKey"`
	CreatedAt        int64  `json:"createdAt" bson:"createdAt"`
	Num              int64  `json:"num" bson:"num"`
	Tag              string `json:"tag" bson:"tag"`
	HdPath           string `json:"hdPath" bson:"hdPath"`
	IsChange         int64  `json:"isChange" bson:"isChange"`
	Applytime        int64  `json:"applytime" bson:"applytime"`
	Dealstate        int64  `json:"dealstate" bson:"dealstate"`
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
	Symbol          string `json:"symbol"`
	IsContract      bool   `json:"isContract"`
	ContractID      string `json:"contractID"`
	ContractAddress string `json:"contractAddress"`
	ContractABI     string `json:"contractABI"`
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
	ErrorMsg   *ErrorMsg                  `json:"errorMsg"`
}

type ErrorMsg struct {
	Code uint64 `json:"code"`
	Err  string `json:"err"`
}

type KeySignature struct {
	EccType     uint32 `json:"eccType"`     //曲线类型
	RSV         bool   `json:"rsv"`         //签名是否需要拼接V
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
	Success      string                 `json:"success"`                        //用于判断交易单链上的真实状态，0：失败，1：成功
	TxType       int64                  `json:"txType"`                         //0:转账, 1:合约调用(发生于主链), >100: 自定义
	TxAction     string                 `json:"txAction"`                       //执行事件, 例如：合约的Transfer事
	BalanceMode  uint64                 `json:"balanceMode" bson:"balanceMode"` //余额模型 0.地址 1.账户
}

func (tx *Transaction) FromSID(n int) string {
	return openwallet.GenTxInputSID(tx.Txid, tx.Symbol, tx.ContractID, uint64(n))
}

func (tx *Transaction) ToSID(n int) string {
	return openwallet.GenTxOutPutSID(tx.Txid, tx.Symbol, tx.ContractID, uint64(n))
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
	AccountID          string                          `json:"accountID"`
	Contracts          map[string]*SummaryContractTask `json:"contracts"`
	FeeRate            string                          `json:"feeRate"`
	OnlyContracts      bool                            `json:"onlyContracts"`
	FeesSupportAccount *FeesSupportAccount             `json:"feesSupportAccount"`
	SwitchSymbol       string                          `json:"switchSymbol"`
	Memo               string                          `json:"memo"`
	*SummarySetting
}

type SummaryContractTask struct {
	*SummarySetting
}

type SummaryWalletTask struct {
	WalletID string                `json:"walletID"`
	Password string                `json:"password,omitempty"`
	Accounts []*SummaryAccountTask `json:"accounts"`
	Wallet   *Wallet               `json:"wallet,omitempty"`
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
	Sids           []string `json:"sids"`                     //汇总交易成功批次号
	TotalSumAmount string   `json:"sumAmount"`                //这次汇总总数
	TotalCostFees  string   `json:"sumFees"`                  //这次汇总总消费手续费
	CreateTime     int64    `json:"createTime" storm:"index"` //汇总时间
}

func (wallet *Wallet) CreateAccount(alias string, symbol *Symbol, key *hdkeystore.HDKey) (*Account, error) {

	var (
		account = &Account{}
	)

	account.Alias = alias
	account.Symbol = symbol.Symbol
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

type Balance struct {
	Symbol           string `json:"symbol" bson:"symbol"`
	AccountID        string `json:"accountID" bson:"accountID"`
	Address          string `json:"address" bson:"address"`
	BalanceType      int64  `json:"type" bson:"type"`
	Balance          string `json:"balance" bson:"balance"`
	ConfirmBalance   string `json:"confirmBalance" bson:"confirmBalance"`
	UnconfirmBalance string `json:"unconfirmBalance" bson:"unconfirmBalance"`
}

func NewBalance(result gjson.Result) *Balance {
	b := Balance{
		Symbol:           result.Get("symbol").String(),
		AccountID:        result.Get("accountID").String(),
		Address:          result.Get("address").String(),
		BalanceType:      result.Get("type").Int(),
		Balance:          result.Get("balance").String(),
		ConfirmBalance:   result.Get("confirmBalance").String(),
		UnconfirmBalance: result.Get("unconfirmBalance").String(),
	}
	return &b
}

type TokenBalance struct {
	ContractID string
	Token      string
	Address    string
	Balance    Balance
	IsContract int64
}

func NewTokenBalance(result gjson.Result) *TokenBalance {
	b := TokenBalance{
		IsContract: result.Get("isContract").Int(),
		ContractID: result.Get("contractID").String(),
		Token:      result.Get("token").String(),
		Address:    result.Get("contractAddress").String(),
		Balance: Balance{
			//Symbol:    symbol,
			AccountID: result.Get("accountID").String(),
			Balance:   result.Get("balance").String(),
		},
	}
	return &b
}

// FeesSupportAccount 主币余额不足时，可选择一个账户提供手续费
type FeesSupportAccount struct {
	AccountID         string `json:"accountID"`         //手续费账户ID
	LowBalanceWarning string `json:"lowBalanceWarning"` //余额过低报警值
	LowBalanceStop    string `json:"lowBalanceStop"`    //余额过低停止手续费支持
	FixSupportAmount  string `json:"fixSupportAmount"`
	FeesScale         string `json:"feesScale"`
	IsTokenContract   bool   `json:"isTokenContract"` //手续费是否合约代币
	ContractAddress   string `json:"contractAddress"` //合约地址
}

// support feeRate
type SupportFeeRate struct {
	FeeRate string
	Symbol  string
	Unit    string
}

// 白名单地址
type TrustAddress struct {
	ID         string `json:"id" storm:"id"`
	Address    string `json:"address"`
	Symbol     string `json:"symbol"`
	Memo       string `json:"memo"`
	CreateTime int64  `json:"createTime"`
}

func NewTrustAddress(address, symbol, memo string) *TrustAddress {
	addr := &TrustAddress{
		Address:    address,
		Symbol:     symbol,
		Memo:       memo,
		CreateTime: time.Now().Unix(),
	}
	addr.ID = GenTrustAddressID(address, symbol)
	return addr
}

func GenTrustAddressID(address, symbol string) string {
	plain := fmt.Sprintf("trustaddress_%s_%s", symbol, address)
	id := base64.StdEncoding.EncodeToString(crypto.SHA256([]byte(plain)))
	return id
}

// SmartContractCallResult 调用结果，不产生交易
type SmartContractCallResult struct {
	Method    string `json:"method"`    //调用方法
	Value     string `json:"value"`     //json结果
	RawHex    string `json:"rawHex"`    //16进制字符串结果
	Status    uint64 `json:"status"`    //0：成功，1：失败
	Exception string `json:"exception"` //异常错误
}

// SmartContractRawTransaction 智能合约原始交易单
type SmartContractRawTransaction struct {
	Coin         Coin                       `json:"coin"`         //@required 区块链类型标识
	TxID         string                     `json:"txID"`         //交易单ID，广播后会生成
	Sid          string                     `json:"sid"`          //@required 业务订单号，保证业务不重复交易而用
	AccountID    string                     `json:"accountID"`    //@required 创建交易单的账户
	Signatures   map[string][]*KeySignature `json:"sigParts"`     //拥有者accountID: []未花签名
	Raw          string                     `json:"raw"`          //交易单调用参数，根据RawType填充数据
	RawType      uint64                     `json:"rawType"`      // 0：hex字符串，1：json字符串，2：base64字符串
	ABIParam     []string                   `json:"abiParam"`     //abi调用参数，[method, arg1, arg2, args...]
	Value        string                     `json:"value"`        //主币数量
	FeeRate      string                     `json:"feeRate"`      //自定义费率
	Fees         string                     `json:"fees"`         //手续费
	AwaitResult  bool                       `json:"awaitResult"`  //是否广播后同时等待结果
	AwaitTimeout uint64                     `json:"awaitTimeout"` //广播后等待超时秒，0 = 默认超时90秒
}

type SmartContractReceipt struct {
	WxID         string                `json:"wxid" storm:"id"` //@required 通过GenTransactionWxID计算
	TxID         string                `json:"txid"`            //@required
	FromAddress  string                `json:"fromAddress"`     //@required 调用者
	ToAddress    string                `json:"toAddress"`       //@required 调用地址，与合约地址一致
	Value        string                `json:"value"`           //主币数量
	Fees         string                `json:"fees"`            //手续费
	Symbol       string                `json:"symbol"`          //主链标识
	ContractID   string                `json:"contractID"`      //合约ID
	ContractName string                `json:"contractName"`    //合约名字
	ContractAddr string                `json:"contractAddr"`    //合约地址
	BlockHash    string                `json:"blockHash"`       //@required
	BlockHeight  uint64                `json:"blockHeight"`     //@required
	IsMain       int64                 `json:"isMain"`          //1.区块数据正常 2.重扫或分叉状态
	Applytime    int64                 `json:"applytime"`       //订单申请时间
	SubmitTime   int64                 `json:"submitTime"`      //订单提交时间
	Succtime     int64                 `json:"succtime"`        //订单处理成功时间
	Dealstate    int64                 `json:"dealstate"`       //处理状态 1.未成功 2.已成功 3.已确认
	Notifystate  int64                 `json:"notifystate"`     //通知状态 1.未通知 2.已通知
	ConfirmTime  int64                 `json:"confirmTime"`     //订单确认时间
	Status       string                `json:"status"`          //@required 链上状态，0：失败，1：成功
	Success      string                `json:"success"`         //用于判断交易单链上的真实状态，0：失败，1：成功
	RawReceipt   string                `json:"rawReceipt"`      //@required 原始交易回执，一般为json
	Events       []*SmartContractEvent `json:"events"`          //@required 执行事件, 例如：event Transfer
}

// SmartContractEvent 事件记录
type SmartContractEvent struct {
	Symbol       string `json:"symbol"`       //主币类型
	ContractID   string `json:"contractID"`   //合约ID
	ContractName string `json:"contractName"` //合约名称
	ContractAddr string `json:"contractAddr"` //合约地址
	Event        string `json:"event"`        //记录事件
	Value        string `json:"value"`        //结果参数，json字符串
}

// FailureSmartContractLog 广播失败的交易单
type FailureSmartContractLog struct {
	RawTx  *SmartContractRawTransaction `json:"rawTx"`
	Reason string                       `json:"error"`
}
