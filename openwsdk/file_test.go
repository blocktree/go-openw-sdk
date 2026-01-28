package openwsdk

import (
	"encoding/hex"
	"fmt"
	"github.com/blocktree/openwallet/v2/hdkeystore"
	"github.com/blocktree/openwallet/v2/openwallet"
	"path/filepath"
	"testing"
)

type AssetAccount struct {
	WalletID     string
	AccountID    string
	Alias        string
	AddressIndex int64
	AccountIndex int64
	PublicKey    string
	ReqSigs      int64
	Symbol       string
	HdPath       string
}

func createWalletFile() (*hdkeystore.HDKey, error) {
	path := filepath.Join(".", "keys")
	key, _, err := hdkeystore.StoreHDKey(path, "sogosdfo", "123TestGCM", hdkeystore.StandardScryptN, hdkeystore.StandardScryptP, hdkeystore.CipherAes256GCM)
	if err != nil {
		return nil, err
	}
	fmt.Println("seed:", hex.EncodeToString(key.Seed()))
	return key, nil
}

func loadWalletFile() (*hdkeystore.HDKey, error) {
	path := filepath.Join(".", "keys")
	ks := hdkeystore.NewHDKeystore(path, hdkeystore.StandardScryptN, hdkeystore.StandardScryptP, hdkeystore.CipherAes256GCM)
	key, err := ks.GetKey("W1iZEvXUWYNgwJSgbxCmoaJkZJrsphRApb", "sogosdfo-W1iZEvXUWYNgwJSgbxCmoaJkZJrsphRApb.key", "123TestGCM")
	if err != nil {
		return nil, err
	}
	return key, nil
}

func CreateAccount(key *hdkeystore.HDKey, symbol, alias string, index, curve int64) (*AssetAccount, error) {

	account := &AssetAccount{}

	account.Alias = alias
	account.Symbol = symbol
	account.ReqSigs = 1

	newAccIndex := index + 1

	// root/n' , 使用强化方案
	account.HdPath = fmt.Sprintf("%s/%d'", key.RootPath, newAccIndex)

	childKey, err := key.DerivedKeyWithPath(account.HdPath, uint32(curve))
	if err != nil {
		return nil, err
	}

	account.PublicKey = childKey.GetPublicKey().OWEncode()
	account.AccountIndex = newAccIndex
	account.AccountID = openwallet.GenAccountID(account.PublicKey)
	account.AddressIndex = -1
	account.WalletID = key.KeyID

	return account, nil

}

func TestCreateWalletFile(t *testing.T) {
	_, err := createWalletFile()
	if err != nil {
		panic(err)
	}
}

func TestLoadWalletFile(t *testing.T) {
	key, err := loadWalletFile()
	if err != nil {
		panic(err)
	}
	fmt.Println("seed:", hex.EncodeToString(key.Seed()))
}

func TestCreateAssetAccount(t *testing.T) {
	key, err := loadWalletFile()
	if err != nil {
		panic(err)
	}
	account, err := CreateAccount(key, "test account", "BETH", 0, 3972005888)
	if err != nil {
		panic(err)
	}
	fmt.Println(account)
}
