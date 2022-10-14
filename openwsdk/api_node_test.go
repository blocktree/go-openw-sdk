package openwsdk

import (
	"fmt"
	"github.com/astaxie/beego/config"
	"github.com/blocktree/go-owcrypt"
	"github.com/blocktree/openwallet/v2/common/file"
	"github.com/blocktree/openwallet/v2/hdkeystore"
	"github.com/blocktree/openwallet/v2/log"
	"github.com/blocktree/openwallet/v2/openwallet"
	"github.com/blocktree/openwallet/v2/owtp"
	"github.com/google/uuid"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func init() {
	owtp.Debug = false
}

func testNewAPINode() *APINode {

	confFile := filepath.Join("conf", "node.ini")
	//confFile := filepath.Join("conf", "test.ini")
	c, err := config.NewConfig("ini", confFile)
	if err != nil {
		log.Error("NewConfig error:", err)
		return nil
	}

	PrivateKey := c.String("PrivateKey")
	AppID := c.String("AppID")
	AppKey := c.String("AppKey")
	Host := c.String("Host")
	EnableKeyAgreement, _ := c.Bool("EnableKeyAgreement")
	EnableSSL, _ := c.Bool("EnableSSL")

	cert, _ := owtp.NewCertificate(PrivateKey)

	config := &APINodeConfig{
		AppID:              AppID,
		AppKey:             AppKey,
		Host:               Host,
		Cert:               cert,
		ConnectType:        owtp.HTTP,
		EnableKeyAgreement: EnableKeyAgreement,
		EnableSSL:          EnableSSL,
		TimeoutSEC:         120,
	}

	api, err := NewAPINodeWithError(config)
	if err != nil {
		log.Errorf("NewAPINodeWithError: %s", err)
		return nil
	}
	api.BindAppDevice()

	return api
}

func testGetLocalKey() (*hdkeystore.HDKey, error) {
	keypath := filepath.Join("testkeys")
	keystore := hdkeystore.NewHDKeystore(
		keypath,
		hdkeystore.StandardScryptN,
		hdkeystore.StandardScryptP,
	)

	//key, err := keystore.GetKey(
	//	"WAaDbbawmypQY3XjnMjLTj43vBGvrQwB2j",
	//	"TRON-WAaDbbawmypQY3XjnMjLTj43vBGvrQwB2j.key",
	//	"1234qwer",
	//)
	key, err := keystore.GetKey(
		"WLN3hJo3NcsbWpsbBjezbJWoy7unZfcaGT",
		"newwallet-WLN3hJo3NcsbWpsbBjezbJWoy7unZfcaGT.key",
		"1234qwer",
	)

	if err != nil {
		return nil, err
	}

	return key, nil
}

func TestAPINode_BindAppDevice(t *testing.T) {
	api := testNewAPINode()
	err := api.BindAppDevice()
	fmt.Println(err)
}

func TestAPINode_GetSymbolList(t *testing.T) {
	api := testNewAPINode()
	api.GetSymbolList("", 0, 1000, 0, true, func(status uint64, msg string, total int, symbols []*Symbol) {
		symbolStrArray := make([]string, 0)
		for _, s := range symbols {
			fmt.Printf("symbol: %+v\n", s)
			symbolStrArray = append(symbolStrArray, s.Coin)
		}
		allSymbols := strings.Join(symbolStrArray, ", ")
		log.Infof("all symbols: %s", allSymbols)
	})
}

func TestAPINode_CreateWallet(t *testing.T) {
	api := testNewAPINode()

	keypath := filepath.Join("testkeys")

	file.MkdirAll(keypath)

	name := "newwallet"
	//password := "1234qwer"

	//随机生成keystore
	//key, _, err := hdkeystore.StoreHDKey(
	//	keypath,
	//	name,
	//	password,
	//	hdkeystore.StandardScryptN,
	//	hdkeystore.StandardScryptP,
	//)

	key, err := testGetLocalKey()
	if err != nil {
		t.Logf("GetKey error: %v\n", err)
		return
	}
	if err != nil {
		t.Logf("unexpected error: %v\n", err)
		return
	}
	wallet := &Wallet{
		Alias:    name,
		WalletID: key.KeyID,
	}
	api.CreateWallet(wallet, true,
		func(status uint64, msg string, wallet *Wallet) {

			if status != owtp.StatusSuccess {
				log.Error(msg)
				return
			}

			if wallet != nil {
				t.Logf("wallet: %+v\n", wallet)
			}
		})
}

func TestAPINode_FindWalletByWalletID(t *testing.T) {
	walletID := "WLN3hJo3NcsbWpsbBjezbJWoy7unZfcaGT"
	api := testNewAPINode()
	api.FindWalletByWalletID(walletID, true,
		func(status uint64, msg string, wallet *Wallet) {

			if status != owtp.StatusSuccess {
				return
			}
			if wallet != nil {
				t.Logf("wallet: %+v\n", wallet)
			}
		})
}

func TestAPINode_CreateAccount(t *testing.T) {
	api := testNewAPINode()

	key, err := testGetLocalKey()
	if err != nil {
		t.Logf("GetKey error: %v\n", err)
		return
	}

	symbol := &Symbol{}
	symbol.Coin = "ETP"
	symbol.Curve = int64(owcrypt.ECC_CURVE_SECP256K1)

	var findWallet *Wallet

	api.FindWalletByWalletID(key.KeyID, true,
		func(status uint64, msg string, wallet *Wallet) {

			if status != owtp.StatusSuccess {
				return
			}

			findWallet = wallet
		})

	if findWallet == nil {
		return
	}

	newacc, err := findWallet.CreateAccount("newacc", symbol, key)
	if err != nil {
		t.Logf("CreateAccount unexpected error: %v\n", err)
		return
	}

	api.CreateNormalAccount(newacc, true,
		func(status uint64, msg string, account *Account, addresses []*Address) {

			if status != owtp.StatusSuccess {
				return
			}

			if account != nil {
				log.Infof("account: %+v\n", account)
			}

			for i, a := range addresses {
				log.Infof("Address[%d]:%+v", i, a)
			}
		})
}

func TestAPINode_FindAccountByWalletID(t *testing.T) {
	walletID := ""
	api := testNewAPINode()
	api.FindAccountByWalletID(walletID, true,
		func(status uint64, msg string, accounts []*Account) {

			if status != owtp.StatusSuccess {
				t.Logf("unexpected error: %v\n", msg)
				return
			}
			for i, a := range accounts {
				log.Infof("account[%d] AccountID:%v", i, a.AccountID)
				log.Infof("account[%d] Symbol:%v", i, a.Symbol)
				log.Infof("account[%d] PublicKey:%v", i, a.PublicKey)
				log.Infof("account[%d] HdPath:%v", i, a.HdPath)
				log.Infof("account[%d] AccountIndex:%v", i, a.AccountIndex)
				log.Infof("account[%d] AddressIndex:%v", i, a.AddressIndex)
				log.Infof("account[%d] Balance:%v", i, a.Balance)
				log.Infof("------------------------------------------")
			}
		})
}

func TestAPINode_FindAccountByAccountID(t *testing.T) {
	accountID := "Ey6MU7v5CdPbpy9Ph18d2v4HwgNAG6UDgQEFfKCRmUZE"
	api := testNewAPINode()
	api.FindAccountByAccountID(accountID, 0, true,
		func(status uint64, msg string, a *Account) {

			if status != owtp.StatusSuccess {
				t.Logf("unexpected error: %v\n", msg)
				return
			}
			log.Infof("account AccountID:%v", a.AccountID)
			log.Infof("account Symbol:%v", a.Symbol)
			log.Infof("account PublicKey:%v", a.PublicKey)
			log.Infof("account HdPath:%v", a.HdPath)
			log.Infof("account AccountIndex:%v", a.AccountIndex)
			log.Infof("account AddressIndex:%v", a.AddressIndex)
			log.Infof("account Balance:%v", a.Balance)
			log.Infof("------------------------------------------")
		})
}

func TestAPINode_CreateAddress(t *testing.T) {
	walletID := "VysrzgpsLsgDpHM2KQMYuPY57fL3BAFU34"
	accountID := "Aa7Chh2MdaGDejHdCJZAaX7AwvGNmMEMry2kZZTq114a"
	api := testNewAPINode()
	api.CreateAddress(walletID, accountID, 2000, true,
		func(status uint64, msg string, addresses []*Address) {

			if status != owtp.StatusSuccess {
				return
			}
			for i, a := range addresses {
				log.Infof("Address[%d]:%+v", i, a)
			}
		})
}

func TestAPINode_CreateBatchAddress(t *testing.T) {
	walletID := "VysrzgpsLsgDpHM2KQMYuPY57fL3BAFU34"
	accountID := "Aa7Chh2MdaGDejHdCJZAaX7AwvGNmMEMry2kZZTq114a"
	api := testNewAPINode()
	api.CreateBatchAddress(walletID, accountID, 5000, true,
		func(status uint64, msg string, addresses []string) {

			if status != owtp.StatusSuccess {
				return
			}
			for i, a := range addresses {
				log.Infof("Address[%d]:%+v", i, a)
			}
		})
}

func TestAPINode_FindAddressByAddress(t *testing.T) {
	addr := "0xc21a2eb0eea50060efb9307627c6d0046ca57d8d"
	api := testNewAPINode()
	api.FindAddressByAddress(addr, true,
		func(status uint64, msg string, address *Address) {

			if status != owtp.StatusSuccess {
				return
			}
			log.Infof("address:%+v", address)
		})
}

func TestAPINode_FindAddressByAccountID(t *testing.T) {
	accountID := "5qEpgFJFdbUQnQccXKyZoE8YBYoKSoHCgPCPVcy9mhVL"
	api := testNewAPINode()
	api.FindAddressByAccountID(accountID, 7200, 50, true,
		func(status uint64, msg string, addresses []*Address) {

			if status != owtp.StatusSuccess {
				return
			}
			for _, a := range addresses {
				log.Infof("Address: %s, Index: %d", a.Address, a.AddrIndex)
			}
		})
}

func testCreateTrade(
	accountID string,
	sid string,
	coin Coin,
	amount string,
	address string,
	feeRate string,
) (*RawTransaction, error) {

	var (
		retRawTx *RawTransaction
		err      error
	)

	api := testNewAPINode()
	api.CreateTrade(accountID, sid, coin, amount, address, feeRate, "", "", true,
		func(status uint64, msg string, rawTx *RawTransaction) {
			if status != owtp.StatusSuccess {
				err = fmt.Errorf(msg)
				return
			}

			retRawTx = rawTx
		})

	return retRawTx, err
}

func testCreateBatchTrade(
	accountID string,
	sid string,
	coin Coin,
	to map[string]string,
	feeRate string,
) (*RawTransaction, error) {

	var (
		retRawTx *RawTransaction
		err      error
	)

	api := testNewAPINode()
	api.CreateBatchTrade(accountID, sid, coin, to, feeRate, "", "", true,
		func(status uint64, msg string, rawTx *RawTransaction) {
			if status != owtp.StatusSuccess {
				err = fmt.Errorf(msg)
				return
			}

			retRawTx = rawTx
		})

	return retRawTx, err
}

func testSubmitTrade(
	rawTx []*RawTransaction,
) ([]*Transaction, []*FailedRawTransaction, error) {

	var (
		retTx     []*Transaction
		retFailed []*FailedRawTransaction
		err       error
	)

	api := testNewAPINode()
	api.SubmitTrade(rawTx, true,
		func(status uint64, msg string, successTx []*Transaction, failedRawTxs []*FailedRawTransaction) {
			if status != owtp.StatusSuccess {
				err = fmt.Errorf(msg)
				return
			}

			retTx = successTx
			retFailed = failedRawTxs
		})

	return retTx, retFailed, err
}

func TestAPINode_Send_LTC(t *testing.T) {
	accountID := "Aa7Chh2MdaGDejHdCJZAaX7AwvGNmMEMry2kZZTq114a"
	sid := uuid.New().String()
	amount := "0.001"
	address := "mkdStRouBPVrDVpYmbE5VUJqhBgxJb3dSS"
	feeRate := "0.001"

	coin := Coin{
		Symbol:     "LTC",
		IsContract: false,
	}

	rawTx, err := testCreateTrade(accountID, sid, coin, amount, address, feeRate)
	if err != nil {
		t.Logf("CreateTrade unexpected error: %v\n", err)
		return
	}
	log.Infof("rawTx: %+v", rawTx)

	key, err := testGetLocalKey()
	if err != nil {
		t.Logf("GetKey error: %v\n", err)
		return
	}

	//签名交易单
	err = SignRawTransaction(rawTx, key)
	if err != nil {
		t.Logf("SignRawTransaction unexpected error: %v\n", err)
		return
	}

	log.Infof("signed rawTx: %+v", rawTx)

	success, fail, err := testSubmitTrade([]*RawTransaction{rawTx})
	if err != nil {
		t.Logf("SubmitTrade unexpected error: %v\n", err)
		return
	}

	log.Info("============== success ==============")

	for _, tx := range success {
		log.Infof("tx: %+v", tx)
	}

	log.Info("")

	log.Info("============== fail ==============")

	for _, tx := range fail {
		log.Infof("tx: %+v", tx.Reason)
	}
}

func TestAPINode_FindTradeLog(t *testing.T) {
	api := testNewAPINode()
	param := map[string]interface{}{
		"txid": "0x1f99030f19f5adefe522df1f3e2683ee7ad017d22cd1648292ac4d163201fbfa",
	}
	api.FindTradeLogByParams(param, true,
		func(status uint64, msg string, tx []*Transaction) {
			for i, value := range tx {
				log.Infof("tx[%d]: %+v", i, value)
			}
		})
}

func TestAPINode_GetContracts(t *testing.T) {
	api := testNewAPINode()
	api.GetContracts("QUORUM", "", 0, 1000, true,
		func(status uint64, msg string, tokens []*TokenContract) {

			for _, s := range tokens {
				fmt.Printf("token: %+v\n", s)
			}

		})
}

func TestAPINode_GetTokenBalanceByAccount(t *testing.T) {
	accountID := "AUXVkMijFjSh1jCsMV2exNj58qzJX7BN27ANnDfBTcTS"
	contractID := "vxfK989y7Mg9TcH0xrCFNSQFj/lN5WaGoEbto5WqIVc="
	api := testNewAPINode()
	api.GetTokenBalanceByAccount(accountID, contractID, true,
		func(status uint64, msg string, balance *TokenBalance) {
			log.Infof("balance: %+v", balance)
		})
}

func TestAPINode_GetAllTokenBalanceByAccount(t *testing.T) {
	accountID := "7u7CQNdkaJXVszoj528Bink88aWgfay3rDxb1rsmDywA"
	api := testNewAPINode()
	api.GetAllTokenBalanceByAccount(accountID, "ETH", true,
		func(status uint64, msg string, balance []*TokenBalance) {
			for _, b := range balance {
				log.Infof("balance: %+v", b)
			}
		})
}

func TestAPINode_GetFeeRate(t *testing.T) {
	symbol := "BTC"
	api := testNewAPINode()
	api.GetFeeRate(symbol, true,
		func(status uint64, msg string, symbol, feeRate, unit string) {

			log.Infof("balance: %s %s/%s", feeRate, symbol, unit)

		})
}

func TestAPINode_Send_TRC10(t *testing.T) {
	accountID := "EaUEnCH9mjDPeqrsfi9q3K3jkTezZCt4cee3RTpgScJ3"
	sid := uuid.New().String()
	amount := "5"
	address := "TBwVUW7Qa2jb2z2q3RMVpg8yLaBsGFvueG"
	feeRate := ""

	coin := Coin{
		Symbol:     "TRX",
		IsContract: true,
		ContractID: "jKyfOtbSvdY57WhDZXJj885A4bs0np5eRdYcwS3ip2I=",
	}

	rawTx, err := testCreateTrade(accountID, sid, coin, amount, address, feeRate)
	if err != nil {
		t.Logf("CreateTrade unexpected error: %v\n", err)
		return
	}
	log.Infof("rawTx: %+v", rawTx)

	key, err := testGetLocalKey()
	if err != nil {
		t.Logf("GetKey error: %v\n", err)
		return
	}

	//签名交易单
	err = SignRawTransaction(rawTx, key)
	if err != nil {
		t.Logf("SignRawTransaction unexpected error: %v\n", err)
		return
	}

	log.Infof("signed rawTx: %+v", rawTx)

	success, fail, err := testSubmitTrade([]*RawTransaction{rawTx})
	if err != nil {
		t.Logf("SubmitTrade unexpected error: %v\n", err)
		return
	}

	log.Info("============== success ==============")

	for _, tx := range success {
		log.Infof("tx: %+v", tx)
	}

	log.Info("")

	log.Info("============== fail ==============")

	for _, tx := range fail {
		log.Infof("tx: %+v", tx.Reason)
	}
}

func TestAPINode_Send_TRC20(t *testing.T) {
	accountID := "EaUEnCH9mjDPeqrsfi9q3K3jkTezZCt4cee3RTpgScJ3"
	sid := uuid.New().String()
	amount := "5"
	address := "TBwVUW7Qa2jb2z2q3RMVpg8yLaBsGFvueG"
	feeRate := ""

	coin := Coin{
		Symbol:     "TRX",
		IsContract: true,
		ContractID: "BEGDiEC5toNC8dyG7G40/vSPHk1FGv6JcCmyf16QOa0=",
	}

	rawTx, err := testCreateTrade(accountID, sid, coin, amount, address, feeRate)
	if err != nil {
		t.Logf("CreateTrade unexpected error: %v\n", err)
		return
	}
	log.Infof("rawTx: %+v", rawTx)

	key, err := testGetLocalKey()
	if err != nil {
		t.Logf("GetKey error: %v\n", err)
		return
	}

	//签名交易单
	err = SignRawTransaction(rawTx, key)
	if err != nil {
		t.Logf("SignRawTransaction unexpected error: %v\n", err)
		return
	}

	log.Infof("signed rawTx: %+v", rawTx)

	success, fail, err := testSubmitTrade([]*RawTransaction{rawTx})
	if err != nil {
		t.Logf("SubmitTrade unexpected error: %v\n", err)
		return
	}

	log.Info("============== success ==============")

	for _, tx := range success {
		log.Infof("tx: %+v", tx)
	}

	log.Info("")

	log.Info("============== fail ==============")

	for _, tx := range fail {
		log.Infof("tx: %+v", tx.Reason)
	}
}

func TestAPINode_GetFeeRateList(t *testing.T) {
	api := testNewAPINode()
	api.GetFeeRateList(true, func(status uint64, msg string, feeRates []SupportFeeRate) {
		for _, feeRate := range feeRates {
			log.Infof("feeRate: %+v", feeRate)
		}
	})
}

func TestAPINode_GetSymbolBlockList(t *testing.T) {
	symbol := "BETH"
	api := testNewAPINode()
	api.GetSymbolBlockList(symbol, true,
		func(status uint64, msg string, blockHeaders []*BlockHeader) {
			for _, header := range blockHeaders {
				log.Infof("blockHeader: %+v", header)
			}
		})
}

func TestAPINode_GetAllTokenBalanceByAddress(t *testing.T) {
	accountID := "BBxgBEn7AoRhNqsS7vjD625B5SafFFdY1QMX7Zq8M9jn"
	address := "WhWV4XcD7UJzt2bAcVe48PN1Cxwh8HAyoi"
	api := testNewAPINode()
	api.GetAllTokenBalanceByAddress(accountID, address, "WICC", true,
		func(status uint64, msg string, balance []*TokenBalance) {
			for _, b := range balance {
				log.Infof("balance: %+v", b)
			}
		})
}

func TestAPINode_GetAllTokenBalanceOfAddressByAccountID(t *testing.T) {
	accountID := "HDGv4GwxfRcGWZBn8Xfp3bJ4zJTTGrajWbjQrRTvMDsG"
	api := testNewAPINode()
	getAddrs := make([]*Address, 0)
	api.FindAddressByAccountID(accountID, 800, 200, true,
		func(status uint64, msg string, addresses []*Address) {

			if status != owtp.StatusSuccess {
				return
			}
			getAddrs = addresses
		})
	for _, a := range getAddrs {
		api.GetAllTokenBalanceByAddress(accountID, a.Address, "XWC", true,
			func(status uint64, msg string, balance []*TokenBalance) {
				for _, b := range balance {
					log.Infof("address: %s, index: %d, token: %s, balance: %s", a.Address, a.AddrIndex, b.Token, b.Balance.Balance)
				}
			})
	}

}

func TestAPINode_ImportAccount(t *testing.T) {
	account := &Account{
		WalletID:     "WLN3hJo3NcsbWpsbBjezbJWoy7unZfcaGT",
		AccountID:    "msa1qea6z8mzfspa3v894975r4gpgs7yfa6fptcmzgc2mnxlca5mtxpqw9111111",
		Alias:        "importBTC",
		PublicKey:    "027a785253aef82a116072d622a57ee46cb8501fbfaf76dfe95ed1f1f91b3e1111",
		Symbol:       "BTC",
		AccountIndex: 16,
		HdPath:       "m/44'/88'/16'",
	}
	api := testNewAPINode()
	api.ImportAccount(account, true, func(status uint64, msg string, account *Account, addresses []*Address) {
		if status != owtp.StatusSuccess {
			t.Errorf(msg)
			return
		}

		if account != nil {
			log.Infof("account: %+v\n", account)
		}

		for i, a := range addresses {
			log.Infof("Address[%d]:%+v", i, a)
		}
	})
}

func TestAPINode_ImportBatchAddress(t *testing.T) {
	api := testNewAPINode()
	walletID := "WLN3hJo3NcsbWpsbBjezbJWoy7unZfcaGT"
	accountID := "msa1qea6z8mzfspa3v894975r4gpgs7yfa6fptcmzgc2mnxlca5mtxpqw9111111"
	addressAndPubs := map[string]string{
		"1QGPMCCtXaop8C2J2mUf3DcofjYgiD8prd": "027a785253aef82a116072d622a57ee46cb8501fbfaf76dfe95ed1f1f91b3e1133",
		//"abcm230948209402": "kslfjlaxcvxvwe2",
		//"abcm230948209403": "kslfjlaxcvxvwe3",
		//"abcm230948209404": "kslfjlaxcvxvwe4",
	}
	err := api.ImportBatchAddress(walletID, accountID, "", addressAndPubs, false, true,
		func(status uint64, msg string, importAddresses []string) {
			if status != owtp.StatusSuccess {
				t.Errorf(msg)
				return
			}

			for i, a := range importAddresses {
				log.Infof("Address[%d]:%+v", i, a)
			}
		})
	if err != nil {
		t.Logf("ImportBatchAddress unexpected error: %v\n", err)
		return
	}
}

func TestAPINode_GetNotifierNodeInfo(t *testing.T) {
	api := testNewAPINode()
	pubKey, nodeId, err := api.GetNotifierNodeInfo()
	if err != nil {
		t.Logf("GetNotifierNodeInfo unexpected error: %v", err)
		return
	}

	log.Infof("pubKey: %s", pubKey)
	log.Infof("nodeId: %s", nodeId)
}

func TestAPINode_FindAccountByParams(t *testing.T) {
	api := testNewAPINode()
	api.FindAccountByParams(nil, 0, 100, true,
		func(status uint64, msg string, accounts []*Account) {
			if status != owtp.StatusSuccess {
				t.Errorf(msg)
				return
			}

			for _, a := range accounts {
				log.Infof("account: %+v", a)
			}
		})
}

func TestAPINode_FindWalletByParams(t *testing.T) {
	api := testNewAPINode()
	api.FindWalletByParams(nil, 0, 100, true,
		func(status uint64, msg string, wallets []*Wallet) {
			if status != owtp.StatusSuccess {
				t.Errorf(msg)
				return
			}

			for _, a := range wallets {
				log.Infof("wallet: %+v", a)
			}
		})
}

func TestAPINode_FindAddressByParams(t *testing.T) {
	api := testNewAPINode()
	param := map[string]interface{}{
		"walletIDs":  []string{"WLVEGBFFSevB5tfsdgKPVGw9zFwccYtafn"},
		"accountIDs": []string{"5qEpgFJFdbUQnQccXKyZoE8YBYoKSoHCgPCPVcy9mhVL"},
		"symbol":     "BSC",
	}
	api.FindAddressByParams(param, 7250, 50, true,
		func(status uint64, msg string, addresses []*Address) {
			if status != owtp.StatusSuccess {
				t.Errorf(msg)
				return
			}
			log.Infof("total addresses: %d", len(addresses))
			for _, a := range addresses {
				log.Infof("address: %s, index: %d", a.Address, a.AddrIndex)
			}
		})
}

func TestAPINode_CreateSummaryTx(t *testing.T) {

	var (
		retErr         *openwallet.Error
		retAccountInfo *Account
		addressLimit   = 200
	)

	api := testNewAPINode()
	accountID := "HDGv4GwxfRcGWZBn8Xfp3bJ4zJTTGrajWbjQrRTvMDsG"
	sumAddress := "XWCNjAfoTwK77dJw58VthU5YKKG9kUfLHvcbK"
	coin := Coin{
		Symbol:          "XWC",
		IsContract:      true,
		ContractID:      "BJftiCFBOsJ3Vu54QfSHXnqwhDbTzQDzF+e62mUsBe8=",
		ContractAddress: "1.3.0",
	}

	//feeSupport := &FeesSupportAccount{
	//	AccountID:         "7dSaDccksuj75DEhtw3LwSCQbybRfVPhxPv8DGDD4M9o",
	//	LowBalanceWarning: "0.001",
	//	LowBalanceStop:    "0.0001",
	//	FeesScale:         "2",
	//}

	api.FindAccountByAccountID(accountID, 0, true,
		func(status uint64, msg string, account *Account) {
			if status != owtp.StatusSuccess {
				retErr = openwallet.Errorf(status, msg)
				return
			}
			retAccountInfo = account
		})

	if retErr != nil {
		t.Errorf("error: %v", retErr)
		return
	}
	if retAccountInfo == nil {
		t.Errorf("retAccountInfo is nil")
		return
	}
	log.Infof("total address = %d", retAccountInfo.AddressIndex+1)

	for i := 0; i <= int(retAccountInfo.AddressIndex); i = i + addressLimit {
		sid := uuid.New().String()
		log.Infof("Create Summary Transaction in address range [%d...%d]", i, i+addressLimit)
		log.Infof("sid = %s", sid)
		api.CreateSummaryTx(accountID, sumAddress, coin,
			"", "0", "0",
			i, addressLimit, 1,
			sid, nil, "", true,
			func(status uint64, msg string, rawTxs []*RawTransaction) {
				if status != owtp.StatusSuccess {
					log.Errorf(msg)
					return
				}
				for _, tx := range rawTxs {
					//log.Infof("from: %+v", tx.Signatures[accountID][0].Address)
					log.Infof("tx: %+v", tx)
					for _, sigs := range tx.Signatures {
						for _, sig := range sigs {
							log.Infof("address: %s, nonce: %s", sig.Address, sig.Nonce)
						}
					}
					if tx.ErrorMsg != nil && tx.ErrorMsg.Code != 0 {
						log.Warning(tx.ErrorMsg.Err)
					}
				}
			})
	}

}

func TestAPINode_VerifyAddress(t *testing.T) {
	api := testNewAPINode()
	symbol := "EOS"
	address := "hrt3arlcl354"
	api.VerifyAddress(symbol, address, true,
		func(status uint64, msg string, flag bool) {
			if status != owtp.StatusSuccess {
				t.Errorf(msg)
				return
			}

			log.Infof("flag: %v", flag)
		})
}

func TestAPINode_CallSmartContractABI(t *testing.T) {
	api := testNewAPINode()
	accountID := "AVrCpwbnGRHEk2HCHTGJ3abFkMARLzarCLCaDpH56H3H"
	coin := Coin{
		Symbol:     "TDEX",
		IsContract: true,
		ContractID: "VVLWoJe++pOGMU8oxp174T+66n3BrP7vG6Mcyqogtqk=",
	}
	abiParam := []string{"balanceOf", "0x941f298dc3ec1728e21065e84237159958a5b20a"}
	api.CallSmartContractABI(accountID, coin, abiParam, "", 0,
		true, func(status uint64, msg string, callResult *SmartContractCallResult) {
			if status != owtp.StatusSuccess {
				t.Errorf(msg)
				return
			}

			log.Infof("callResult: %+v", callResult)
		})
}

func TestAPINode_FollowSmartContractRecepit(t *testing.T) {
	api := testNewAPINode()
	contracts := []string{"dl8WD7bM7xk4ZxRybuHCo3JDDtZn2ugPusapoKnQEWA="}
	api.FollowSmartContractReceipt(contracts,
		true, func(status uint64, msg string) {
			log.Infof("status: %v, msg: %v", status, msg)
		})
}

func TestAPINode_CreateSmartContractTrade(t *testing.T) {

	var (
		retRawTx  *SmartContractRawTransaction
		retTx     []*SmartContractReceipt
		retFailed []*FailureSmartContractLog
		err       error
		key       *hdkeystore.HDKey
		accountID = "5kTzFGuAH9UkB9yhZdmXtF8hGPh6iPt4hf8Q3DVy3Xo3"
		sid       = uuid.New().String()
		coin      = Coin{
			Symbol:     "QUORUM",
			IsContract: true,
			ContractID: "dl8WD7bM7xk4ZxRybuHCo3JDDtZn2ugPusapoKnQEWA=",
		}
		abiParam = []string{"transfer", "0x19a4b5d6ea319a5d5ad1d4cc00a5e2e28cac5ec3", "123"}
	)

	api := testNewAPINode()
	api.CreateSmartContractTrade(sid, accountID, coin, abiParam, "", 0, "", "1", true,
		func(status uint64, msg string, rawTx *SmartContractRawTransaction) {
			if status != owtp.StatusSuccess {
				err = fmt.Errorf(msg)
				return
			}
			retRawTx = rawTx
		})
	log.Infof("rawTx: %+v", retRawTx)

	key, err = testGetLocalKey()
	if err != nil {
		t.Logf("GetKey error: %v\n", err)
		return
	}

	//签名交易单
	signatures, signErr := SignTxHash(retRawTx.Signatures, key)
	if signErr != nil {
		t.Logf("SignRawTransaction unexpected error: %v\n", signErr)
		return
	}

	retRawTx.Signatures = signatures
	retRawTx.AwaitResult = true

	log.Infof("signed rawTx: %+v", retRawTx)

	api.SubmitSmartContractTrade([]*SmartContractRawTransaction{retRawTx}, true,
		func(status uint64, msg string, successTx []*SmartContractReceipt, failedRawTxs []*FailureSmartContractLog) {
			if status != owtp.StatusSuccess {
				err = fmt.Errorf(msg)
				return
			}

			retTx = successTx
			retFailed = failedRawTxs
		})

	log.Info("============== success ==============")

	for _, tx := range retTx {
		log.Infof("tx: %+v", tx)

		for i, event := range tx.Events {
			log.Std.Notice("events[%d]: %+v", i, event)
		}
	}

	log.Info("")

	log.Info("============== fail ==============")

	for _, tx := range retFailed {
		log.Infof("tx: %+v", tx.Reason)
	}
}

func TestAPINode_FindSmartContractReceiptByParams(t *testing.T) {
	api := testNewAPINode()
	param := map[string]interface{}{
		"txid": "0x3d266b3c7e9f09ab8484086fd7db082b5f4a9b756307a067b89bf91320e4edc4",
	}
	api.FindSmartContractReceiptByParams(param, true,
		func(status uint64, msg string, receipts []*SmartContractReceipt) {
			for i, value := range receipts {
				log.Infof("receipt[%d]: %+v", i, value)
				for i, event := range value.Events {
					log.Std.Notice("events[%d]: %+v", i, event)
				}
			}
		})

	api.OWTPNode().Close()
}

func TestAPINode_CreateSmartContractTradeMulti(t *testing.T) {

	multiFunc := func() {

		var (
			retRawTx  *SmartContractRawTransaction
			retTx     []*SmartContractReceipt
			retFailed []*FailureSmartContractLog
			err       error
			key       *hdkeystore.HDKey
			accountID = "5kTzFGuAH9UkB9yhZdmXtF8hGPh6iPt4hf8Q3DVy3Xo3"
			sid       = uuid.New().String()
			coin      = Coin{
				Symbol:     "QUORUM",
				IsContract: true,
				ContractID: "dl8WD7bM7xk4ZxRybuHCo3JDDtZn2ugPusapoKnQEWA=",
			}
			abiParam = []string{"transfer", "0x19a4b5d6ea319a5d5ad1d4cc00a5e2e28cac5ec3", "123"}
		)

		api := testNewAPINode()
		api.CreateSmartContractTrade(sid, accountID, coin, abiParam, "", 0, "", "0", true,
			func(status uint64, msg string, rawTx *SmartContractRawTransaction) {
				if status != owtp.StatusSuccess {
					err = fmt.Errorf(msg)
					return
				}
				retRawTx = rawTx
			})
		log.Infof("rawTx: %+v", retRawTx)

		key, err = testGetLocalKey()
		if err != nil {
			t.Logf("GetKey error: %v\n", err)
			return
		}

		//签名交易单
		signatures, signErr := SignTxHash(retRawTx.Signatures, key)
		if signErr != nil {
			t.Logf("SignRawTransaction unexpected error: %v\n", signErr)
			return
		}

		retRawTx.Signatures = signatures

		log.Infof("signed rawTx: %+v", retRawTx)

		api.SubmitSmartContractTrade([]*SmartContractRawTransaction{retRawTx}, true,
			func(status uint64, msg string, successTx []*SmartContractReceipt, failedRawTxs []*FailureSmartContractLog) {
				if status != owtp.StatusSuccess {
					err = fmt.Errorf(msg)
					return
				}

				retTx = successTx
				retFailed = failedRawTxs
			})

		if len(retTx) > 0 {
			log.Info("============== success ==============")

			for _, tx := range retTx {
				log.Infof("tx: %+v", tx)
			}
		}

		log.Info("")

		if len(retFailed) > 0 {
			log.Info("============== fail ==============")

			for _, tx := range retFailed {
				log.Infof("tx: %+v", tx.Reason)
			}
		}

	}

	var wait sync.WaitGroup

	for i := 0; i < 5; i++ {
		wait.Add(1)
		go func() {
			multiFunc()
			wait.Done()
		}()
	}

	wait.Wait()

}

func TestAPINode_CreateBatchTrade(t *testing.T) {
	accountID := "BtbecAGzVMFX4PT1QjKpQ1cDKhWtJcFqPoLP5CVi2gVt"
	sid := uuid.New().String()
	feeRate := ""

	coin := Coin{
		Symbol:     "ETP",
		IsContract: true,
		ContractID: "q7OtI9+EP6Ze8LNvJDGStyS15mzhnntduQ3wcphv9Zc=",
	}

	receivers := map[string]string{
		"M8zhymCjZD9ZzSR9skirEhJnNDEdcJBb6c": "1",
		"MC3byQPhQS9dQYkY4ME5R94j5GksWvDYTR": "1",
		"MDgW56oXUFMRfSuRhPcyoWcGwSyj81hUnM": "1",
		"MHr2w1nQ2aiGuh7McpAvi5TMvqmzVLJeNC": "1",
		"MMRbpJdtxXeNmdwRZa4JjNgraL2XKUeg4e": "1",
		"MNVJdDfesiRdPMWz1QCnyvAXTbTcfdaBun": "1",
	}

	rawTx, err := testCreateBatchTrade(accountID, sid, coin, receivers, feeRate)
	if err != nil {
		t.Logf("CreateTrade unexpected error: %v\n", err)
		return
	}
	log.Infof("rawTx: %+v", rawTx)

	key, err := testGetLocalKey()
	if err != nil {
		t.Logf("GetKey error: %v\n", err)
		return
	}

	//签名交易单
	err = SignRawTransaction(rawTx, key)
	if err != nil {
		t.Logf("SignRawTransaction unexpected error: %v\n", err)
		return
	}

	log.Infof("signed rawTx: %+v", rawTx)

	success, fail, err := testSubmitTrade([]*RawTransaction{rawTx})
	if err != nil {
		t.Logf("SubmitTrade unexpected error: %v\n", err)
		return
	}

	log.Info("============== success ==============")

	for _, tx := range success {
		log.Infof("tx: %+v", tx)
	}

	log.Info("")

	log.Info("============== fail ==============")

	for _, tx := range fail {
		log.Infof("tx: %+v", tx.Reason)
	}
}

func TestAPINode_CreateTrade(t *testing.T) {
	accountID := "B6UbwNZrz82QqUPBsSCPUkEJ1CjxSzSwnewziCseCt4c"
	sid := uuid.New().String()
	amount := "3266824611.6667"
	address := "MQFXFvYmVmr7LTCPMs61wLiZvrgg2vnN75"
	feeRate := ""

	coin := Coin{
		Symbol:          "ETP",
		IsContract:      true,
		ContractID:      "q7OtI9+EP6Ze8LNvJDGStyS15mzhnntduQ3wcphv9Zc=",
		ContractAddress: "DNA",
		ContractABI:     "",
	}

	rawTx, err := testCreateTrade(accountID, sid, coin, amount, address, feeRate)
	if err != nil {
		t.Logf("CreateTrade unexpected error: %v\n", err)
		return
	}
	log.Infof("rawTx: %+v", rawTx)
}
