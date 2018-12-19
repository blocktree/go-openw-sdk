package openwsdk

import (
	"fmt"
	"github.com/blocktree/OpenWallet/hdkeystore"
	"github.com/blocktree/go-openw-server/model"
)

type Wallet struct {
	model.OwWallet
}

type Symbol struct {
	model.OwSymbol
}

type Account struct {
	model.OwAccount
}

type Address struct {
	model.OwAddress
}

type TokenContract struct {
	model.OwContract
}

type Coin struct {
	Symbol     string `json:"symbol"`
	IsContract uint64 `json:"isContract"`
	ContractID string `json:"contractID"`
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
	model.OwTradeLog
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
	account.AccountID = account.GetAccountID()
	account.AddressIndex = -1
	account.WalletID = wallet.WalletID

	return account, nil

}
