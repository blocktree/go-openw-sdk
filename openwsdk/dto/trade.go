package dto

import (
	"github.com/godaddy-x/freego/node/common"
)

//easyjson:json
type AssetsAccount struct {
	WalletID        string   `json:"walletID"`
	AccountID       string   `json:"accountID"`
	Index           uint64   `json:"index"`
	HdPath          string   `json:"hdPath"`
	PublicKey       string   `json:"publicKey"`
	OwnerKeys       []string `json:"ownerKeys"`
	ContractAddress string   `json:"contractAddress"`
	Symbol          string   `json:"symbol"`
	AddressIndex    int64    `json:"addressIndex"`
	MainSymbol      string   `json:"mainSymbol"`
	Alias           string   `json:"alias"`
	ReqSigs         int64    `json:"reqSigs"`
	IsTrust         bool     `json:"isTrust"`
}

//easyjson:json
type CoinInfo struct {
	Symbol          string        `json:"symbol"`
	IsContract      bool          `json:"isContract"`
	ContractID      string        `json:"contractID"`
	ContractAddress string        `json:"contractAddress"`
	ContractABI     string        `json:"contractABI"`
	Contract        SmartCoinInfo `json:"contract"`
}

//easyjson:json
type FeesSupportAccount struct {
	AccountID        string `json:"accountID"`
	FixSupportAmount string `json:"fixSupportAmount"`
	FeesSupportScale string `json:"feesSupportScale"`
}

//easyjson:json
type SmartCoinInfo struct {
	ContractID string `json:"contractID"`
	Symbol     string `json:"symbol"`
	Address    string `json:"address"`
	Token      string `json:"token"`
	Protocol   string `json:"protocol"`
	Name       string `json:"name"`
	Decimals   uint64 `json:"decimals"`
}

//easyjson:json
type KeySig struct {
	Address     string `json:"address"`
	DerivedPath string `json:"derivedPath"`
	EccType     uint32 `json:"eccType"`
	InputIndex  int64  `json:"inputIndex"`
	Msg         string `json:"msg"`
	Nonce       string `json:"nonce"`
	Signed      string `json:"signed"`
	WalletID    string `json:"walletID"`
	IsImport    int64  `json:"isImport"`
	PublicKey   string `json:"publicKey"`
	Rsv         bool   `json:"rsv"`
}

//easyjson:json
type SigParts struct {
	Key string   `json:"key"`
	Sig []KeySig `json:"sig"`
}

//easyjson:json
type SubmitRawTransaction struct {
	AppID     string            `json:"appID"`
	WalletID  string            `json:"walletID"`
	AccountID string            `json:"accountID"`
	Coin      CoinInfo          `json:"coin"`
	RawHex    string            `json:"rawHex"`
	RawHexSig string            `json:"rawHexSig"`
	ReqSigs   uint64            `json:"reqSigs"`
	Sid       string            `json:"sid"`
	SigCount  int64             `json:"sigCount"`
	ExtParam  string            `json:"extParam"`
	Fees      string            `json:"fees"`
	To        map[string]string `json:"to"`
	FeeRate   string            `json:"feeRate"`
	SigParts  []SigParts        `json:"sigParts"`
	ErrorMsg  map[string]string `json:"errorMsg"`
	UserID    int64             `json:"userID"`
	Account   AssetsAccount     `json:"account"`
	DataType  int64             `json:"dataType"` // 1.普通交易 2.汇总交易
}

//easyjson:json
type SubmitRawTransactionReq struct {
	common.BaseReq
	TxData *TxData `json:"txData"`
}

//easyjson:json
type SubmitRawTransactionRes struct {
	Txid        string   `json:"txid"`
	Wxid        string   `json:"wxid"`
	AccountID   string   `json:"accountID"`
	Coin        CoinInfo `json:"coin"`
	From        []string `json:"from"`
	To          []string `json:"to"`
	Amount      string   `json:"amount"`
	Decimal     int32    `json:"decimal"`
	TxType      uint64   `json:"txType"`
	TxAction    string   `json:"txAction"`
	Confirm     int64    `json:"confirm"`
	BlockHash   string   `json:"blockHash"`
	BlockHeight uint64   `json:"blockHeight"`
	IsMemo      bool     `json:"isMemo"`
	Memo        string   `json:"memo"`
	Fees        string   `json:"fees"`
	Received    bool     `json:"received"`
	SubmitTime  int64    `json:"submitTime"`
	ConfirmTime int64    `json:"confirmTime"`
	Status      string   `json:"status"`
	Reason      string   `json:"reason"`
	ExtParam    string   `json:"extParam"`
}

//easyjson:json
type GetTransactionFeeEstimatedReq struct {
	common.BaseReq
	Symbol        string `json:"symbol"`
	From          string `json:"from"`
	To            string `json:"to"`
	Amount        string `json:"amount"`
	Data          string `json:"data"`
	Decimals      int32  `json:"decimals"`
	TokenAddress  string `json:"tokenAddress"`
	TokenDecimals int32  `json:"tokenDecimals"`
}

//easyjson:json
type GetTransactionFeeEstimatedRes struct {
	GasLimit    string `json:"gasLimit"`
	GasPrice    string `json:"gasPrice"`
	Fee         string `json:"fee"`
	MainBalance string `json:"mainBalance"`
}

//easyjson:json
type GetTransactionCountOnChainReq struct {
	common.BaseReq
	Symbol  string `json:"symbol"`
	Address string `json:"address"`
}

//easyjson:json
type GetTransactionCountOnChainRes struct {
	Nonce uint64 `json:"nonce"`
}

//easyjson:json
type CreateTradeReq struct {
	common.BaseReq
	AccountID string            `json:"accountID"`
	Sid       string            `json:"sid"`
	Coin      CoinInfo          `json:"coin"`
	FeeRate   string            `json:"feeRate"`
	ExtParam  string            `json:"extParam"`
	Memo      string            `json:"memo"`
	To        map[string]string `json:"to"`
}

//easyjson:json
type TxData struct {
	Sid        string            `json:"sid"`
	Data       string            `json:"data"`
	DataSign   string            `json:"dataSign"`  // 业务系统进行校验签名
	TradeSign  string            `json:"tradeSign"` // CLI系统进行校验签名
	Code       string            `json:"code"`
	Message    string            `json:"message"`
	SignerList map[string]string `json:"signerList"`
}

//easyjson:json
type CreateTradeRes struct {
	TxData []*TxData `json:"txData"`
}

// CreateSummaryTxReq 用于创建汇总交易（归集交易）的请求结构体。
//
//easyjson:json
type CreateSummaryTxReq struct {
	common.BaseReq // 嵌入基础请求字段，通常包含通用参数如签名、时间戳等。
	// AccountID 是发起归集操作的账户唯一标识符。
	AccountID string `json:"accountID"`
	// MinTransfer 表示只有当地址余额大于等于此值时，才会被纳入归集范围。
	// 通常以最小单位（如 satoshi、wei）表示，字符串类型避免精度丢失。
	MinTransfer string `json:"minTransfer"`
	// RetainedBalance 表示在每个被归集的地址上保留的最小余额（不归集的部分）。
	// 同样以最小单位表示，字符串类型。
	RetainedBalance string `json:"retainedBalance"`
	// Address 是目标归集地址，即所有满足条件的资金将被发送到该地址。
	Address string `json:"address"`
	// Coin 指定要归集的币种信息，包括币种名称、链类型等。
	Coin CoinInfo `json:"coin"`
	// FeeRate 是交易手续费率，通常以每字节/每虚拟字节（vByte）或 Gas Price 形式表示。
	// 具体格式取决于底层链（如 BTC 用 sat/vB，ETH 用 Gwei）。
	FeeRate string `json:"feeRate"`
	// AddressStartIndex 用于分页扫描地址时的起始索引（通常用于 HD 钱包派生地址）。
	AddressStartIndex int64 `json:"addressStartIndex"`
	// AddressLimit 表示本次归集操作最多扫描的地址数量，用于控制批量处理规模。
	AddressLimit int64 `json:"addressLimit"`
	// Confirms 表示只归集已确认数大于等于该值的 UTXO 或交易输出（主要用于 UTXO 链如 BTC）。
	// 例如，Confirms=6 表示只归集经过 6 个区块确认的资金。
	Confirms int64 `json:"confirms"`
	// FeesSupportAccount 指定用于支付交易手续费的账户信息（可选）。
	// 在某些场景下，归集资金账户本身可能不支付手续费，而是由另一个账户代付。
	FeesSupportAccount FeesSupportAccount `json:"feesSupportAccount"`
	// Memo 是附加的备注信息，某些链（如 XRP、Stellar、Cosmos）支持在交易中携带 memo。
	// 用于标识交易目的或关联业务信息。
	Memo string `json:"memo"`
	// Sid 是客户端生成的唯一请求 ID，用于幂等性控制，防止重复提交相同请求。
	Sid string `json:"sid"`
}
