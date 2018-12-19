package openwsdk

import (
	"fmt"
	"github.com/blocktree/OpenWallet/common/file"
	"github.com/blocktree/OpenWallet/hdkeystore"
	"github.com/blocktree/OpenWallet/log"
	"github.com/blocktree/OpenWallet/owtp"
	"github.com/blocktree/go-owcrypt"
	"path/filepath"
	"testing"
)

func init() {
	owtp.Debug = true
}

func testNewAPINode() *APINode {

	//--------------- PRIVATE KEY ---------------
	//APt4potcFAqr6aSh5XdNPgWPtvExLnRvQP9KXYWfM5rR
	//
	//--------------- PUBLIC KEY ---------------
	//APt4potcFAqr6aSh5XdNPgWPtvExLnRvQP9KXYWfM5rR
	//--------------- NODE ID ---------------
	//G6s787hxsrGyfhaFss8VNaEimXo22qWdRkFQA953eziz

	cert, _ := owtp.NewCertificate("APt4potcFAqr6aSh5XdNPgWPtvExLnRvQP9KXYWfM5rR", "")

	config := &APINodeConfig{
		AppID:  "b4b1962d415d4d30ec71b28769fda585",
		AppKey: "8c511cb683041f3589419440fab0a7b7710907022b0d035baea9001d529ca72f",
		Host:   "47.52.191.89",
		Cert:   cert,
	}

	api := NewAPINode(config)
	return api
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
	keypath := filepath.Join("testkeys")
	keystore := hdkeystore.NewHDKeystore(
		keypath,
		hdkeystore.StandardScryptN,
		hdkeystore.StandardScryptP,
	)

	symbol := &Symbol{}
	symbol.Coin = "BTC"
	symbol.Curve = int64(owcrypt.ECC_CURVE_SECP256K1)

	key, err := keystore.GetKey(
		"VysrzgpsLsgDpHM2KQMYuPY57fL3BAFU34",
		"gooaglag-VysrzgpsLsgDpHM2KQMYuPY57fL3BAFU34.key",
		"1234qwer",
	)

	if err != nil {
		t.Logf("GetKey error: %v\n", err)
		return
	}

	wallet := &Wallet{}
	wallet.WalletID = key.KeyID
	wallet.AccountIndex = -1
	wallet.RootPath = key.RootPath

	newacc, err := wallet.CreateAccount("newacc", symbol, key)
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
