package model

// API接口
type OwApi struct {
	Id       int64  `json:"id" bson:"_id" tb:"ow_api" mg:"true"`
	Name     string `json:"name" bson:"name"`
	Resume   string `json:"resume" bson:"resume"`
	Method   string `json:"method" bson:"method"`
	Version  string `json:"version" bson:"version"`
	Url      string `json:"url" bson:"url"`
	Usestate int64  `json:"usestate" bson:"usestate"`
	orderno  int64  `json:"orderno" bson:"orderno"`
	Ctime    int64  `json:"ctime" bson:"ctime"`
	Utime    int64  `json:"utime" bson:"utime"`
	State    int64  `json:"state" bson:"state"`
}
