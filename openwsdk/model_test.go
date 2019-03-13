package openwsdk

import (
	"github.com/blocktree/openwallet/common/file"
	"github.com/blocktree/openwallet/hdkeystore"
	"github.com/blocktree/openwallet/log"
	"github.com/blocktree/go-owcrypt"
	"path/filepath"
	"testing"
)

func TestWallet_CreateAccount(t *testing.T) {

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
		t.Logf("StoreHDKey unexpected error: %v\n", err)
		return
	}

	wallet := &Wallet{}
	wallet.Alias = name
	wallet.WalletID = key.KeyID
	wallet.AccountIndex = -1
	wallet.RootPath = key.RootPath

	symbol := &Symbol{}
	symbol.Coin = "BTC"
	symbol.Curve = int64(owcrypt.ECC_CURVE_SECP256K1)

	account, err := wallet.CreateAccount("newacc", symbol, key)
	if err != nil {
		t.Logf("CreateAccount unexpected error: %v\n", err)
		return
	}
	log.Infof("account:%+v", account)
}
