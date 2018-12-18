package model

// 货币信息
type OwSymbol struct {
	Id       int64  `json:"id" bson:"_id" tb:"ow_symbol" mg:"true"`
	Top      string `json:"top" bson:"top"`
	Parent   string `json:"parent" bson:"parent"`
	Ctype    int64  `json:"ctype" bson:"ctype"`
	Name     string `json:"name" bson:"name"`
	Coin     string `json:"coin" bson:"coin"`
	Resume   string `json:"resume" bson:"resume"`
	Icon     string `json:"icon" bson:"icon"`
	Orderno  int64  `json:"orderno" bson:"orderno"`
	Confirm  int64  `json:"confirm" bson:"confirm"`
	Decimals int64  `json:"decimals" bson:"decimals"`
	Ctime    int64  `json:"ctime" bson:"ctime"`
	Utime    int64  `json:"utime" bson:"utime"`
	State    int64  `json:"state" bson:"state"`
}
