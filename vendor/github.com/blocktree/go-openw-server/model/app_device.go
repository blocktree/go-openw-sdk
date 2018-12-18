package model

// APP开通设备关系
type OwAppDevice struct {
	Id        int64  `json:"id" bson:"_id" tb:"ow_app_device" mg:"true"`
	AppID     string `json:"appID" bson:"appID"`
	Pid       string `json:"pid" bson:"pid"`
	DeviceID  string `json:"deviceID" bson:"deviceID"`
	Applytime int64  `json:"applytime" bson:"applytime"`
	Usestate  int64  `json:"usestate" bson:"usestate"`
	Ctime     int64  `json:"ctime" bson:"ctime"`
	Utime     int64  `json:"utime" bson:"utime"`
	State     int64  `json:"state" bson:"state"`
}
