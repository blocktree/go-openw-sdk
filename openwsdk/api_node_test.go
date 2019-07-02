package openwsdk

import (
	"fmt"
	"github.com/astaxie/beego/config"
	"github.com/blocktree/go-owcrypt"
	"github.com/blocktree/openwallet/common/file"
	"github.com/blocktree/openwallet/hdkeystore"
	"github.com/blocktree/openwallet/log"
	"github.com/blocktree/openwallet/owtp"
	"github.com/google/uuid"
	"path/filepath"
	"testing"
)

func init() {
	owtp.Debug = false
}

func testNewAPINode() *APINode {

	confFile := filepath.Join("conf", "node.ini")

	c, err := config.NewConfig("ini", confFile)
	if err != nil {
		log.Error("NewConfig error:", err)
		return nil
	}

	PrivateKey := c.String("PrivateKey")
	AppID := c.String("AppID")
	AppKey := c.String("AppKey")
	Host := c.String("Host")

	cert, _ := owtp.NewCertificate(PrivateKey)

	config := &APINodeConfig{
		AppID:              AppID,
		AppKey:             AppKey,
		Host:               Host,
		Cert:               cert,
		ConnectType:        owtp.HTTP,
		EnableSignature:    false,
		EnableKeyAgreement: true,
		EnableSSL:          true,
		TimeoutSEC:         120,
	}

	api := NewAPINode(config)
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
		"newwallet-WLN3hJo3NcsbWpsbBjezbJWoy7unZfcaGT",
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
	api.GetSymbolList(0, 1000, 0, true, func(status uint64, msg string, symbols []*Symbol) {

		for _, s := range symbols {
			fmt.Printf("symbol: %+v\n", s)
		}

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

	api.CreateWallet(name, key.KeyID, true,
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
	symbol.Coin = "TRX"
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
	walletID := "WN84dVZXpgVixsvXnU8jkFWD1qWHp15LpA"
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
	accountID := "3i26MQmtuWVVnw8GnRCVopG3pi8MaYU6RqWVV2E1hwJx"
	api := testNewAPINode()
	api.FindAccountByAccountID(accountID, 1, true,
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
	addr := "mgCzMJDyJoqa6XE3RSdNGvD5Bi5VTWudRq"
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
	accountID := "EaUEnCH9mjDPeqrsfi9q3K3jkTezZCt4cee3RTpgScJ3"
	api := testNewAPINode()
	api.FindAddressByAccountID(accountID, 0, 10, true,
		func(status uint64, msg string, addresses []*Address) {

			if status != owtp.StatusSuccess {
				return
			}
			for i, a := range addresses {
				log.Infof("Address[%d]:%+v", i, a)
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
	api.CreateTrade(accountID, sid, coin, amount, address, feeRate, "", true,
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
	walletID := "W3LxqTNAcXFqW7HGcTuERRLXKdNWu17Ccx"
	accountID := "PgHCcfMbcw1zXRNZo23NFjRdBmcN5tzrb1j5McRLJbG"
	api := testNewAPINode()
	api.FindTradeLog(walletID, accountID, "", "", "",
		0, 0, 0, 0, 0, false, 0, 200, true,
		func(status uint64, msg string, tx []*Transaction) {
			for i, value := range tx {
				log.Infof("tx[%d]: %+v", i, value)
			}
		})
}

func TestAPINode_GetContracts(t *testing.T) {
	api := testNewAPINode()
	api.GetContracts("", 0, 1000, true,
		func(status uint64, msg string, tokens []*TokenContract) {

			for _, s := range tokens {
				fmt.Printf("token: %+v\n", s)
			}

		})
}

func TestAPINode_GetTokenBalanceByAccount(t *testing.T) {
	accountID := "EaUEnCH9mjDPeqrsfi9q3K3jkTezZCt4cee3RTpgScJ3"
	contractID := "jKyfOtbSvdY57WhDZXJj885A4bs0np5eRdYcwS3ip2I="
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

func TestAPINode_CreateSummaryTx(t *testing.T) {

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
	symbol := "BTC"
	api := testNewAPINode()
	api.GetSymbolBlockList(symbol, true,
		func(status uint64, msg string, blockHeaders []*BlockHeader) {
			for _, header := range blockHeaders {
				log.Infof("blockHeader: %+v", header)
			}
		})
}
