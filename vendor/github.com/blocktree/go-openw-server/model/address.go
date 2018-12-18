package model

import (
	"github.com/blocktree/OpenWallet/common"
	"github.com/blocktree/OpenWallet/openwallet"
)

// 交易地址
type OwAddress struct {
	Id        int64                  `json:"id" bson:"_id" tb:"ow_address" mg:"true"`
	AppID     string                 `json:"appID" bson:"appID"`
	WalletID  string                 `json:"walletID" bson:"walletID"`
	AccountID string                 `json:"accountID" bson:"accountID"`
	Alias     string                 `json:"alias" bson:"alias"`
	Symbol    string                 `json:"symbol" bson:"symbol"`
	AddrIndex int64                  `json:"addrIndex" bson:"addrIndex"`
	Address   string                 `json:"address" bson:"address"`
	Balance   string                 `json:"balance" bson:"balance"`
	IsMemo    int64                  `json:"isMemo" bson:"isMemo"`
	Memo      string                 `json:"memo" bson:"memo"`
	WatchOnly int64                  `json:"watchOnly" bson:"watchOnly"`
	PublicKey string                 `json:"publicKey" bson:"publicKey"`
	ExtParam  map[string]interface{} `json:"extParam" bson:"extParam"`
	CreatedAt int64                  `json:"createdAt" bson:"createdAt"`
	Num       int64                  `json:"num" bson:"num"`
	Tag       string                 `json:"tag" bson:"tag"`
	HdPath    string                 `json:"hdPath" bson:"hdPath"`
	Batchno   string                 `json:"batchno" bson:"batchno"`
	IsChange  int64                  `json:"isChange" bson:"isChange"`
	Applytime int64                  `json:"applytime" bson:"applytime"`
	Succtime  int64                  `json:"succtime" bson:"succtime"`
	Dealstate int64                  `json:"dealstate" bson:"dealstate"`
	Ctime     int64                  `json:"ctime" bson:"ctime"`
	Utime     int64                  `json:"utime" bson:"utime"`
	State     int64                  `json:"state" bson:"state"`
}

func (self OwAddress) ToAddress() *openwallet.Address {
	return &openwallet.Address{
		AccountID: self.AccountID,
		Address:   self.Address,
		PublicKey: self.PublicKey,
		Alias:     self.Alias,
		Tag:       self.Tag,
		Index:     uint64(self.AddrIndex),
		HDPath:    self.HdPath,
		WatchOnly: common.UIntToBool(uint64(self.WatchOnly)),
		Symbol:    self.Symbol,
		Balance:   self.Balance,
		IsMemo:    common.UIntToBool(uint64(self.IsMemo)),
		IsChange:  common.UIntToBool(uint64(self.IsChange)),
	}
}
