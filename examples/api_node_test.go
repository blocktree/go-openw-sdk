package performance_test

import (
	"fmt"
	"github.com/blocktree/go-openw-sdk/v2/openwsdk"
	"github.com/blocktree/openwallet/v2/log"
	"github.com/blocktree/openwallet/v2/owtp"
	"testing"
)

const (
	sslkey = "FXJXCtxAfHWhAvnpsnciEfVCkThn7NGMA1kBofYRECRe"
	host   = "127.0.0.1:8422"
	appid  = "e10adc3949ba59abbe56e057f20f883e"
	appkey = "8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92"
)

var api *openwsdk.APINode

func init() {
	api = testNewAPINode()
}

func testNewAPINode() *openwsdk.APINode {
	cert, _ := owtp.NewCertificate(sslkey)
	config := &openwsdk.APINodeConfig{
		AppID:              appid,
		AppKey:             appkey,
		Host:               host,
		Cert:               cert,
		ConnectType:        owtp.HTTP,
		EnableSignature:    false,
		EnableKeyAgreement: false,
		TimeoutSEC:         120,
	}
	api := openwsdk.NewAPINode(config)
	api.BindAppDevice()
	return api
}

// 获取节点配置信息
func TestAPINode_GetNotifierNodeInfo(t *testing.T) {
	pubKey, nodeId, err := api.GetNotifierNodeInfo()
	if err != nil {
		t.Logf("GetNotifierNodeInfo unexpected error: %v", err)
		return
	}

	log.Infof("pubKey: %s", pubKey)
	log.Infof("nodeID: %s", nodeId)
}

// 获取币种列表,包含推荐费率和区块高度
func TestAPINode_GetSymbolList(t *testing.T) {
	api.GetSymbolList("QTUM", 0, 1000, 0, true, func(status uint64, msg string, total int, symbols []*openwsdk.Symbol) {
		if status != owtp.StatusSuccess {
			log.Error(msg)
			return
		}
		for _, s := range symbols {
			fmt.Printf("symbol: %+v\n", s)
		}
	})
}

// 获取合约列表
func TestAPINode_GetContracts(t *testing.T) {
	api.GetContracts("", "", 0, 10, true,
		func(status uint64, msg string, tokens []*openwsdk.TokenContract) {
			if status != owtp.StatusSuccess {
				log.Error(msg)
				return
			}
			for _, s := range tokens {
				fmt.Printf("token: %+v\n", s)
			}
		})
}

/***************************** Create Wallet/Account/Address *****************************/
const symbol = "ETH"
const walletID = "WFouRqSBTH5GUxpBLDdQBDrQdY14sqe22H"

func TestAPINode_CreateWallet(t *testing.T) {
	wallet := &openwsdk.Wallet{
		Alias:    "test walelt2",
		WalletID: walletID,
	}
	api.CreateWallet(wallet, true,
		func(status uint64, msg string, wallet *openwsdk.Wallet) {
			if status != owtp.StatusSuccess {
				log.Error(msg)
				return
			}
			if wallet != nil {
				t.Logf("wallet: %+v\n", wallet)
			}
		})
}

func TestAPINode_CreateAccount(t *testing.T) {
	account := &openwsdk.Account{
		AppID:        appid,
		WalletID:     walletID,
		Alias:        "test eth account",
		Symbol:       symbol,
		AccountID:    "HPR2Fr7p5ba6NYbht1BN39948yuj2YSZnqHZvM4wkS6K",
		PublicKey:    "owpubeyoV6FsMGEtjhBiuJrun7fvfA228ZU9MvQZCeYekcMA1SxnCFNqJbytWpLxx1gLoc4FvPU8vxQwM7ztgHGP7Yj3jqMxmpPRDP7jtEFNPMpvqQhzMw",
		AccountIndex: 0,
		HdPath:       "m/44'/88'/0'",
		ReqSigs:      1,
	}
	api.CreateNormalAccount(account, true,
		func(status uint64, msg string, account *openwsdk.Account, addresses []*openwsdk.Address) {
			if status != owtp.StatusSuccess {
				log.Error(msg)
				return
			}
			if account != nil {
				log.Infof("account: %+v\n", account)
			}
			for i, a := range addresses {
				log.Infof("aAddress[%d]:%+v", i, a)
			}
		})
}

func TestAPINode_CreateAddress(t *testing.T) {
	accountID := "HPR2Fr7p5ba6NYbht1BN39948yuj2YSZnqHZvM4wkS6K"
	api.CreateAddress(symbol, walletID, accountID, 1, true,
		func(status uint64, msg string, addresses []*openwsdk.Address) {
			if status != owtp.StatusSuccess {
				log.Error(msg)
				return
			}
			for i, a := range addresses {
				log.Infof("address[%d]:%+v", i, a)
			}
		})
}

func TestAPINode_VerifyAddress(t *testing.T) {
	symbol := symbol
	address := "0xf3e2470c6f832c793f4ca364bd64ff7915fe2f2a"
	api.VerifyAddress(symbol, address, true,
		func(status uint64, msg string, flag bool) {
			if status != owtp.StatusSuccess {
				t.Errorf(msg)
				return
			}
			log.Infof("flag: %v", flag)
		})
}

/***************************** Query Wallet/Account/Address *****************************/
func TestAPINode_FindWalletByWalletID(t *testing.T) {
	api.FindWalletByWalletID(walletID, true,
		func(status uint64, msg string, wallet *openwsdk.Wallet) {
			if status != owtp.StatusSuccess {
				log.Error(msg)
				return
			}
			if wallet != nil {
				t.Logf("wallet: %+v\n", wallet)
			}
		})
}

func TestAPINode_FindAccountByWalletID(t *testing.T) {
	api.FindAccountByWalletID(symbol, walletID, 0, 10, true,
		func(status uint64, msg string, accounts []*openwsdk.Account) {
			if status != owtp.StatusSuccess {
				t.Logf("unexpected error: %v\n", msg)
				return
			}
			for i, a := range accounts {
				log.Infof("account[%d]:%v", i, a)
			}
		})
}

func TestAPINode_FindAccountByAccountID(t *testing.T) {
	accountID := "HPR2Fr7p5ba6NYbht1BN39948yuj2YSZnqHZvM4wkS6K"
	api.FindAccountByAccountID(symbol, accountID, 0, true,
		func(status uint64, msg string, a *openwsdk.Account) {
			if status != owtp.StatusSuccess {
				t.Logf("unexpected error: %v\n", msg)
				return
			}
			log.Infof("account :%v", a)
		})
}

func TestAPINode_FindAddressByAddress(t *testing.T) {
	addr := "0x3ed64aed00a24f27febd000b4e919cf80992e120"
	api.FindAddressByAddress(symbol, addr, true,
		func(status uint64, msg string, address *openwsdk.Address) {
			if status != owtp.StatusSuccess {
				log.Error(msg)
				return
			}
			log.Infof("address:%+v", address)
		})
}

func TestAPINode_FindAddressByAccountID(t *testing.T) {
	accountID := "HPR2Fr7p5ba6NYbht1BN39948yuj2YSZnqHZvM4wkS6K"
	api.FindAddressByAccountID(symbol, accountID, 0, 10, true,
		func(status uint64, msg string, addresses []*openwsdk.Address) {
			if status != owtp.StatusSuccess {
				log.Error(msg)
				return
			}
			for i, a := range addresses {
				log.Infof("Address[%d]:%+v", i, a)
			}
		})
}

/***************************** Query Balance Account/Address *****************************/
func TestAPINode_GetBalanceByAccount(t *testing.T) {
	accountID := "GFAf6QWK8a9DHNG2xn8R1qvHjEpC9XafhkK2hcrUjwgQ"
	api.GetBalanceByAccount(symbol, accountID, "", true,
		func(status uint64, msg string, balance *openwsdk.BalanceResult) {
			if status != owtp.StatusSuccess {
				log.Error(msg)
				return
			}
			log.Infof("balance: %+v", balance)
		})
}

func TestAPINode_GetBalanceByAddress(t *testing.T) {
	address := "0xc27992b757a3c00ed3cb1dfa7dfb1a59d70dbd0f"
	api.GetBalanceByAddress(symbol, address, "", true, func(status uint64, msg string, balance *openwsdk.BalanceResult) {
		if status != owtp.StatusSuccess {
			log.Error(msg)
			return
		}
		log.Infof("balance: %+v", balance)
	})
}

/***************************** Create/Submit Trade *****************************/

// createTrade
// submitTrade
// createSummaryTx

func testCreateTrade(
	accountID string,
	sid string,
	coin openwsdk.Coin,
	to map[string]string,
	feeRate string,
) (*openwsdk.RawTransaction, error) {
	var (
		retRawTx *openwsdk.RawTransaction
		err      error
	)
	api.CreateTrade(accountID, sid, coin, to, feeRate, "", "", true,
		func(status uint64, msg string, rawTx *openwsdk.RawTransaction) {
			if status != owtp.StatusSuccess {
				err = fmt.Errorf(msg)
				return
			}
			retRawTx = rawTx
		})

	return retRawTx, err
}

/***************************** Query TradeLog *****************************/
func TestAPINode_FindTradeLog(t *testing.T) {
	param := map[string]interface{}{
		"txID":   "0x1f99030f19f5adefe522df1f3e2683ee7ad017d22cd1648292ac4d163201fbfa",
		"lastID": 0,
		"limit":  10,
	}
	api.FindTradeLogByParams(param, true,
		func(status uint64, msg string, tx []*openwsdk.Transaction) {
			if status != owtp.StatusSuccess {
				log.Error(msg)
				return
			}
			for i, value := range tx {
				log.Infof("tx[%d]: %+v", i, value)
			}
		})
}
