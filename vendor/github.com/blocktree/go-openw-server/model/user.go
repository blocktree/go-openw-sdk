package model

// APP用户
type OwUser struct {
	Id          int64  `json:"id" bson:"_id" tb:"ow_user" mg:"true"`
	Uid         string `json:"uid" bson:"uid"`
	Username    string `json:"username" bson:"username"`
	Mobile      string `json:"mobile" bson:"mobile"`
	Password    string `json:"password" bson:"password"`
	Type        int64  `json:"type" bson:"type"`
	Email       string `json:"email" bson:"email"`
	Nickname    string `json:"nickname" bson:"nickname"`
	Headimg     string `json:"headimg" bson:"headimg"`
	Realname    string `json:"realname" bson:"realname"`
	Idno        string `json:"idno" bson:"idno"`
	Company     string `json:"company" bson:"company"`
	CompanyAddr string `json:"companyAddr" bson:"companyAddr"`
	Applytime   int64  `json:"applytime" bson:"applytime"`
	Succtime    int64  `json:"succtime" bson:"succtime"`
	Dealstate   int64  `json:"dealstate" bson:"dealstate"`
	Gvalid      int64  `json:"gvalid" bson:"gvalid"`
	Ctime       int64  `json:"ctime" bson:"ctime"`
	Utime       int64  `json:"utime" bson:"utime"`
	State       int64  `json:"state" bson:"state"`
}
