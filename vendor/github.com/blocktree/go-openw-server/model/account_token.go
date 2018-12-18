package model

// 资产账户-合约
type OwAccountToken struct {
	Id         int64    `json:"id" bson:"_id" tb:"ow_account_token" mg:"true"`
	AppID      string   `json:"appID" bson:"appID"`
	WalletID   string   `json:"walletID" bson:"walletID"`
	AccountID  string   `json:"accountID" bson:"accountID"`
	Symbol     string   `json:"symbol" bson:"symbol"`
	Balance    string   `json:"balance" bson:"balance"`
	Token      string `json:"token" bson:"token"`
	ContractID string   `json:"contractID" bson:"contractID"`
	Ctime      int64    `json:"ctime" bson:"ctime"`
	Utime      int64    `json:"utime" bson:"utime"`
	State      int64    `json:"state" bson:"state"`
}
