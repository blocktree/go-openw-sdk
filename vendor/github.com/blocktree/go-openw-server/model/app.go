package model

// APP客户端
type OwApp struct {
	Id            int64  `json:"id" bson:"_id" tb:"ow_app" mg:"true"`
	UserId        string `json:"userId" bson:"userId"`
	Name          string `json:"name" bson:"name"`
	Resume        string `json:"resume" bson:"resume"`
	Appid         string `json:"appid" bson:"appid"`
	Pubkey        string `json:"pubkey" bson:"pubkey"`
	Notify_pubkey string `json:"notify_pubkey" bson:"notify_pubkey"`
	Notify_prikey string `json:"notify_prikey" bson:"notify_prikey"`
	Applytime     int64  `json:"applytime" bson:"applytime"`
	Succtime      int64  `json:"succtime" bson:"succtime"`
	Usestate      int64  `json:"usestate" bson:"usestate"`
	IpAddress     string `json:"ipAddress" bson:"ipAddress"`
	Notifyurl     string `json:"notifyurl" bson:"notifyurl"`
	Ctime         int64  `json:"ctime" bson:"ctime"`
	Utime         int64  `json:"utime" bson:"utime"`
	State         int64  `json:"state" bson:"state"`
}
