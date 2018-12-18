package model

// 交易订单
type OwTrade struct {
	Id          int64                  `json:"id" bson:"_id" tb:"ow_trade" mg:"true"`
	AppID       string                 `json:"appID" bson:"appID"`
	WalletID    string                 `json:"walletID" bson:"walletID"`
	AccountID   string                 `json:"accountID" bson:"accountID"`
	Sid         string                 `json:"sid" bson:"sid"`
	TxID        string                 `json:"txID" bson:"txID"`
	FromAddress []string               `json:"fromAddress" bson:"fromAddress"`
	ToAddress   []string               `json:"toAddress" bson:"toAddress"`
	Amount      string                 `json:"amount" bson:"amount"`
	Fees        string                 `json:"fees" bson:"fees"`
	Reqtype     int64                  `json:"reqtype" bson:"reqtype"`
	Request     map[string]interface{} `json:"request" bson:"request"`
	Response    map[string]interface{} `json:"response" bson:"response"`
	Applytime   int64                  `json:"applytime" bson:"applytime"`
	Succtime    int64                  `json:"succtime" bson:"succtime"`
	Dealstate   int64                  `json:"dealstate" bson:"dealstate"`
	Ctime       int64                  `json:"ctime" bson:"ctime"`
	Utime       int64                  `json:"utime" bson:"utime"`
	State       int64                  `json:"state" bson:"state"`
}
