package openwsdk

type Message struct {
	Status string
	Msg string
}

type Wallet struct {
	AppID        string `json:"appID"`
	WalletID     string `json:"walletID"`
	Alias        string `json:"walletID"`
	IsTrust      int64  `json:"isTrust"`
	RootPath     string `json:"rootPath"`
	AccountIndex int64  `json:"accountIndex"`
	Dealstate    uint64 `json:"dealstate"`
	Applytime    int64  `json:"Applytime"`
}

type Symbol struct {
	Coin     string `json:"coin"`
	Name     string `json:"name"`
	Orderno  uint64 `json:"orderno"`
	Decimals int32  `json:"decimals"`
	Confirm  uint64 `json:"confirm"`
}
