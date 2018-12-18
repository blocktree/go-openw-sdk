package model

import (
	"github.com/blocktree/OpenWallet/openwallet"
)

// 智能合约
type OwContract struct {
	Id         int64  `json:"id" bson:"_id" tb:"ow_contract" mg:"true"`
	ContractID string `json:"contractID" bson:"contractID"`
	Symbol     string `json:"symbol" bson:"symbol"` //主链标记
	Name       string `json:"name" bson:"name"`
	Decimals   int64  `json:"decimals" bson:"decimals"`
	Address    string `json:"address" bson:"address"`
	Token      string `json:"token" bson:"token"` //token标记
	Protocol   string `json:"protocol" bson:"protocol"`
	Ctime      int64  `json:"ctime" bson:"ctime"`
	State      int64  `json:"state" bson:"state"`
}

// 中心数据库contract转成openwallet.SmartContract
func (contract OwContract) ToSmartContract() openwallet.SmartContract {
	return openwallet.SmartContract{
		ContractID:contract.ContractID,
		Symbol:contract.Symbol,
		Address:contract.Address,
		Token:contract.Token,
		Protocol:contract.Protocol,
		Name:contract.Name,
		Decimals:uint64(contract.Decimals),
	}
}

