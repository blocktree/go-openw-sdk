package dto

import "github.com/godaddy-x/freego/node/common"

//easyjson:json
type CliWalletResult struct {
	Alias    string `json:"alias"`
	WalletID string `json:"walletID"`
	RootPath string `json:"rootPath"`
	Version  int    `json:"version"`
}

//easyjson:json
type CliFindWalletListReq struct {
	common.BaseReq
}

//easyjson:json
type CliFindWalletListRes struct {
	Result []WalletResult `json:"result"`
}

//easyjson:json
type CliUnlockWalletReq struct {
	common.BaseReq
	Filename string `json:"filename"`
}

//easyjson:json
type CliUnlockWalletRes struct {
	WalletID string `json:"walletID"`
}

//easyjson:json
type CliCreateWalletReq struct {
	common.BaseReq
	Alias string `json:"alias"`
}

//easyjson:json
type CliCreateWalletRes struct {
	WalletID string `json:"walletID"`
}

//easyjson:json
type CliCreateAccountReq struct {
	common.BaseReq
	WalletID  string `json:"walletID"`
	LastIndex int64  `json:"lastIndex"` // 錢包所屬帳戶ID最後索引值
	Curve     int64  `json:"curve"`
}

//easyjson:json
type CliCreateAccountRes struct {
	WalletID       string   `json:"walletID"`
	AccountID      string   `json:"accountID"`
	OtherOwnerKeys []string `json:"otherOwnerKeys"`
	ReqSigs        int64    `json:"reqSigs"`
	PublicKey      string   `json:"publicKey"`
	HdPath         string   `json:"hdPath"`
	AccountIndex   int64    `json:"accountIndex"`
	AddressIndex   int64    `json:"addressIndex"`
}

//easyjson:json
type CliSignTransactionReq struct {
	common.BaseReq
	Type      int64  `json:"type"` // 0.普通交易 1.汇总交易
	Data      string `json:"data"`
	TradeSign string `json:"tradeSign"` // CLI系统进行校验签名
}

//easyjson:json
type CliSignTransactionRes struct {
	SignerList map[string]string `json:"signerList"`
}

//easyjson:json
type CliSignTradeKeyReq struct {
	common.BaseReq
	Type int64  `json:"type"` // 0.普通交易 1.汇总交易
	Data string `json:"data"`
}

//easyjson:json
type CliSignTradeKeyRes struct {
	Sign string `json:"sign"`
}

//easyjson:json
type CliShardingTaskReq struct {
	common.BaseReq
	TaskID    string `json:"taskID"`
	PublicKey string `json:"publicKey"`
}

//easyjson:json
type CliShardingTaskRes struct {
	TaskID        string `json:"taskID"`        // 任务ID
	KeyID         string `json:"keyID"`         // 钱包ID
	PublicKey     string `json:"publicKey"`     // 节点临时公钥
	ShardKey      string `json:"shardKey"`      // 加密分片数据
	ShardKeySize  int    `json:"shardKeySize"`  // 分片数据原始长度
	ShardKeyIndex int    `json:"shardKeyIndex"` // 分片数组的索引值
	ExpiredTime   int64  `json:"expiredTime"`   // 任务过期时间, 秒
	Status        int64  `json:"status"`        // 0.任务已创建 10.服务端已下发上传公钥通知 20.节点已上传公钥 30.服务端已下发拉取分片数据通知
}

//easyjson:json
type CliMPCPublicKeyPair struct {
	Subject   string `json:"subject"`
	PublicKey string `json:"publicKey"`
}

// CliMPCKeygenStartRes 服务端下发给节点的「开始 MPC keygen」消息（push: mpcKeygenStart）
//
//easyjson:json
type CliMPCKeygenStartRes struct {
	TaskID        string                `json:"taskID"`
	NodeIDs       []string              `json:"nodeIDs"`
	Threshold     int                   `json:"threshold"`
	ExpiredTime   int64                 `json:"expiredTime"`
	PublicKeyPair []CliMPCPublicKeyPair `json:"publicKeyPair"`
}

// CliMPCKeygenResultReq 节点上报 keygen 结果（POST /ws/mpcKeygenResult）
//
//easyjson:json
type CliMPCKeygenResultReq struct {
	common.BaseReq
	TaskID         string `json:"taskID"`
	NodeID         string `json:"nodeID"`
	KeyID          string `json:"keyID"`
	SaveDataBase64 string `json:"saveDataBase64"`
	Err            string `json:"err"`
}

// CliMPCKeygenResultRes 服务端对 keygen 结果的上报响应
//
//easyjson:json
type CliMPCKeygenResultRes struct {
	OK  bool   `json:"ok"`
	Err string `json:"err,omitempty"`
}

// CliMPCKeygenMsgReq 节点发出的 TSS 协议消息（服务端转发给其他节点）
//
//easyjson:json
type CliMPCKeygenMsgReq struct {
	common.BaseReq
	TaskID          string   `json:"taskID"`
	WireBytesBase64 string   `json:"wireBytesBase64"`
	FromIndex       int      `json:"fromIndex"`
	IsBroadcast     bool     `json:"isBroadcast"`
	ToNodeIDs       []string `json:"toNodeIDs,omitempty"`
}

// CliMPCKeygenMsgRes 服务端推送给节点的 TSS 协议消息（push: mpcKeygenMsg）
//
//easyjson:json
type CliMPCKeygenMsgRes struct {
	TaskID          string   `json:"taskID"`
	WireBytesBase64 string   `json:"wireBytesBase64"`
	FromIndex       int      `json:"fromIndex"`
	IsBroadcast     bool     `json:"isBroadcast"`
	ToNodeIDs       []string `json:"toNodeIDs"`
}

// CliMPCTempPublicKeyReq 节点推送给服务端临时ECDH公钥
//
//easyjson:json
type CliMPCTempPublicKeyReq struct {
	common.BaseReq
	TaskID    string `json:"taskID"`
	Module    string `json:"module"`
	PublicKey string `json:"publicKey"`
}

// CliMPCTempPublicKeyRes 节点推送给服务端临时ECDH公钥结果
//
//easyjson:json
type CliMPCTempPublicKeyRes struct {
	Success bool `json:"success"`
}

// CliMPCEncryptData 加密传输对象
//
//easyjson:json
type CliMPCEncryptData struct {
	TaskID  string `json:"taskID"`
	Subject string `json:"subject"`
	Data    string `json:"data"`
}

// 分布式签名相关 DTO

// CliMPCSignStartRes 服务端下发给节点的「开始 MPC sign」消息（push: mpcSignStart）
//
//easyjson:json
type CliMPCSignStartRes struct {
	TaskID        string                `json:"taskID"`
	KeyID         string                `json:"keyID"`       // 要使用的根密钥 KeyID
	NodeIDs       []string              `json:"nodeIDs"`     // 全量节点列表
	SignNodeIDs   []string              `json:"signNodeIDs"` // 参与签名的节点（TSS 顺序）
	Threshold     int                   `json:"threshold"`   // 门限（通常与 keygen 一致）
	MsgHashHex    string                `json:"msgHashHex"`  // 待签名消息哈希（32字节 hex）
	ExpiredTime   int64                 `json:"expiredTime"` // 任务过期时间，秒级时间戳
	PublicKeyPair []CliMPCPublicKeyPair `json:"publicKeyPair"`
}

// CliMPCSignResultReq 节点上报签名结果（POST /ws/mpcSignResult）
//
//easyjson:json
type CliMPCSignResultReq struct {
	common.BaseReq
	TaskID       string `json:"taskID"`
	NodeID       string `json:"nodeID"`
	KeyID        string `json:"keyID"`
	SignatureHex string `json:"signatureHex"` // R||S 的 64字节 hex（或空，当 Err 不为空时）
	Err          string `json:"err"`
}

// CliMPCSignResultRes 服务端对签名结果上报的响应
//
//easyjson:json
type CliMPCSignResultRes struct {
	OK  bool   `json:"ok"`
	Err string `json:"err,omitempty"`
}

// CliMPCSignMsgReq 节点发出的 TSS 签名协议消息（服务端转发给其他节点）
//
//easyjson:json
type CliMPCSignMsgReq struct {
	common.BaseReq
	TaskID          string   `json:"taskID"`
	WireBytesBase64 string   `json:"wireBytesBase64"`
	FromIndex       int      `json:"fromIndex"`
	IsBroadcast     bool     `json:"isBroadcast"`
	ToNodeIDs       []string `json:"toNodeIDs,omitempty"`
}

// CliMPCSignMsgRes 服务端推送给节点的 TSS 签名协议消息（push: mpcSignMsg）
//
//easyjson:json
type CliMPCSignMsgRes struct {
	TaskID          string   `json:"taskID"`
	WireBytesBase64 string   `json:"wireBytesBase64"`
	FromIndex       int      `json:"fromIndex"`
	IsBroadcast     bool     `json:"isBroadcast"`
	ToNodeIDs       []string `json:"toNodeIDs"`
}

//easyjson:json
type CliMPCResultRes struct {
	OK bool `json:"ok"`
}
