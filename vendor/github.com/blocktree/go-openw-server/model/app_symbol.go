package model

// 应用开通货币关系
type OwAppSymbol struct {
	Id        int64  `json:"id" bson:"_id" tb:"ow_app_symbol" mg:"true"`
	AppID     string `json:"appID" bson:"appID"`
	SymbolId  int64  `json:"symbolId" bson:"symbolId"`
	Usestate  int64  `json:"usestate" bson:"usestate"`
	Applytime int64  `json:"applytime" bson:"applytime"`
	Ctime     int64  `json:"ctime" bson:"ctime"`
	Utime     int64  `json:"utime" bson:"utime"`
	State     int64  `json:"state" bson:"state"`
}
