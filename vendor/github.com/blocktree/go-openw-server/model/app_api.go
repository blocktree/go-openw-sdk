package model

// APP开通API关系
type OwAppApi struct {
	Id        int64  `json:"id" bson:"_id" tb:"ow_app_api" mg:"true"`
	AppID     string `json:"appID" bson:"appID"`
	ApiId     int64  `json:"apiId" bson:"apiId"`
	Usecount  int64  `json:"usecount" bson:"usecount"`
	Usestate  int64  `json:"usestate" bson:"usestate"`
	Applytime int64  `json:"applytime" bson:"applytime"`
	Ctime     int64  `json:"ctime" bson:"ctime"`
	Utime     int64  `json:"utime" bson:"utime"`
	State     int64  `json:"state" bson:"state"`
}
