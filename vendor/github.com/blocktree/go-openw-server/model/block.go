package model

// 区块信息
type OwBlock struct {
	Id                int64  `json:"id" bson:"_id" tb:"ow_block" mg:"true"`
	Hash              string `json:"hash" bson:"hash"`
	Confirmations     string `json:"confirmations" bson:"confirmations"`
	Merkleroot        string `json:"merkleroot" bson:"merkleroot"`
	Previousblockhash string `json:"previousblockhash" bson:"previousblockhash"`
	Height            int64  `json:"height" bson:"height"`
	Version           int64  `json:"version" bson:"version"`
	Time              int64  `json:"time" bson:"time"`
	Fork              string `json:"fork" bson:"fork"`
	Symbol            string `json:"symbol" bson:"symbol"`
	Ctime             int64  `json:"ctime" bson:"ctime"`
	Utime             int64  `json:"utime" bson:"utime"`
	State             int64  `json:"state" bson:"state"`
}
