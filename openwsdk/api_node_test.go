package openwsdk

import (
	"fmt"
	"github.com/blocktree/OpenWallet/common/file"
	"github.com/blocktree/OpenWallet/hdkeystore"
	"github.com/blocktree/OpenWallet/log"
	"github.com/blocktree/OpenWallet/owtp"
	"github.com/blocktree/go-owcrypt"
	"github.com/google/uuid"
	"path/filepath"
	"testing"
)

func init() {
	owtp.Debug = false
}

func testNewAPINode() *APINode {

	//--------------- PRIVATE KEY ---------------
	//APt4potcFAqr6aSh5XdNPgWPtvExLnRvQP9KXYWfM5rR
	//
	//--------------- PUBLIC KEY ---------------
	//APt4potcFAqr6aSh5XdNPgWPtvExLnRvQP9KXYWfM5rR
	//--------------- NODE ID ---------------
	//G6s787hxsrGyfhaFss8VNaEimXo22qWdRkFQA953eziz

	cert, _ := owtp.NewCertificate(owtp.RandomPrivateKey(), "")

	config := &APINodeConfig{
		AppID:  "b4b1962d415d4d30ec71b28769fda585",
		AppKey: "8c511cb683041f3589419440fab0a7b7710907022b0d035baea9001d529ca72f",
		Host:   "47.52.191.89",
		Cert:   cert,
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

	key, err := keystore.GetKey(
		"VysrzgpsLsgDpHM2KQMYuPY57fL3BAFU34",
		"gooaglag-VysrzgpsLsgDpHM2KQMYuPY57fL3BAFU34.key",
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
	api.GetSymbolList(0, 1000, true, func(status uint64, msg string, symbols []*Symbol) {

		for _, s := range symbols {
			fmt.Printf("symbol: %+v\n", s)
		}

	})
}

func TestAPINode_CreateWallet(t *testing.T) {
	api := testNewAPINode()

	keypath := filepath.Join("testkeys")

	file.MkdirAll(keypath)

	name := "gooaglag"
	password := "1234qwer"

	//随机生成keystore
	key, _, err := hdkeystore.StoreHDKey(
		keypath,
		name,
		password,
		hdkeystore.StandardScryptN,
		hdkeystore.StandardScryptP,
	)

	if err != nil {
		t.Logf("unexpected error: %v\n", err)
		return
	}

	api.CreateWallet(name, key.KeyID, true,
		func(status uint64, msg string, wallet *Wallet) {
			if wallet != nil {
				t.Logf("wallet: %+v\n", wallet)
			}
		})
}

func TestAPINode_FindWalletByWalletID(t *testing.T) {
	walletID := "VysrzgpsLsgDpHM2KQMYuPY57fL3BAFU34"
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
	symbol.Coin = "LTC"
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
	walletID := "VysrzgpsLsgDpHM2KQMYuPY57fL3BAFU34"
	api := testNewAPINode()
	api.FindAccountByWalletID(walletID, true,
		func(status uint64, msg string, accounts []*Account) {

			if status != owtp.StatusSuccess {
				return
			}
			for i, a := range accounts {
				log.Infof("account[%d]:%+v", i, a)
			}
		})
}

func TestAPINode_CreateAddress(t *testing.T) {
	walletID := "VysrzgpsLsgDpHM2KQMYuPY57fL3BAFU34"
	accountID := "6EPMmTGx89qEjfftMhrLVg8SHayW8HaU8BbcgDyeyYFj"
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

func TestAPINode_FindAddressByAddress(t *testing.T) {
	addr := "mxYKaDV22uqo8rB4EQKqLQFsRbGuey34vs"
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
	accountID := "6EPMmTGx89qEjfftMhrLVg8SHayW8HaU8BbcgDyeyYFj"
	api := testNewAPINode()
	api.FindAddressByAccountID(accountID, true,
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
	rawTx *RawTransaction,
) (*Transaction, error) {

	var (
		retTx *Transaction
		err   error
	)

	api := testNewAPINode()
	api.SubmitTrade(rawTx, true,
		func(status uint64, msg string, rawTx *Transaction) {
			if status != owtp.StatusSuccess {
				err = fmt.Errorf(msg)
				return
			}

			retTx = rawTx
		})

	return retTx, err
}

func TestAPINode_Send_LTC(t *testing.T) {
	accountID := "Aa7Chh2MdaGDejHdCJZAaX7AwvGNmMEMry2kZZTq114a"
	sid := uuid.New().String()
	amount := "0.01"
	address := "mkSfFCHPAaHAyx9gBokXQMGWmyRtzpk4JK"
	feeRate := "0.001"

	coin := Coin{
		Symbol:     "LTC",
		IsContract: 0,
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

	tx, err := testSubmitTrade(rawTx)

	if err != nil {
		t.Logf("SubmitTrade unexpected error: %v\n", err)
		return
	}
	log.Infof("tx: %+v", tx)
}

func TestAPINode_FindTradeLog(t *testing.T) {
	walletID := "VysrzgpsLsgDpHM2KQMYuPY57fL3BAFU34"
	accountID := "Aa7Chh2MdaGDejHdCJZAaX7AwvGNmMEMry2kZZTq114a"
	api := testNewAPINode()
	api.FindTradeLog(walletID, accountID, "", "",
		0, 0, 0, 1000, true,
		func(status uint64, msg string, tx []*Transaction) {

		})
}

func TestAPINode_GetContracts(t *testing.T) {
	api := testNewAPINode()
	api.GetContracts("ETH", 0, 1000, true,
		func(status uint64, msg string, tokens []*TokenContract) {

			for _, s := range tokens {
				fmt.Printf("token: %+v\n", s)
			}

		})
}

func TestAPINode_GetTokenBalanceByAccount(t *testing.T) {
	accountID := "Aa7Chh2MdaGDejHdCJZAaX7AwvGNmMEMry2kZZTq114a"
	contractID := "rsD1RPmcpo33cnqobvX0hBi7Lex0mmN4RwXt1bLV6fs="
	api := testNewAPINode()
	api.GetTokenBalanceByAccount(accountID, contractID, true,
		func(status uint64, msg string, balance string) {

			log.Infof("balance: %s", balance)

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
