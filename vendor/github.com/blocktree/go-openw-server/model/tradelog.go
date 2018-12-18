package model

// 交易订单日志
type OwTradeLog struct {
	Id           int64                  `json:"id" bson:"_id" tb:"ow_trade_log" mg:"true"`
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
	Ctime        int64                  `json:"ctime" bson:"ctime"`
	Utime        int64                  `json:"utime" bson:"utime"`
	State        int64                  `json:"state" bson:"state"`
}

// 交易订单日志-临时数据
type OwTradeLogTmp struct {
	Id           int64                  `json:"id" bson:"_id" tb:"ow_trade_log_tmp" mg:"true"`
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
	Ctime        int64                  `json:"ctime" bson:"ctime"`
	Utime        int64                  `json:"utime" bson:"utime"`
	State        int64                  `json:"state" bson:"state"`
}
