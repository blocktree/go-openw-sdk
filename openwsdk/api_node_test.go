package openwsdk

import (
	"fmt"
	"github.com/blocktree/OpenWallet/common/file"
	"github.com/blocktree/OpenWallet/hdkeystore"
	"github.com/blocktree/OpenWallet/owtp"
	"path/filepath"
	"testing"
)

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
		AppID: "b4b1962d415d4d30ec71b28769fda585",
		AppKey: "8c511cb683041f3589419440fab0a7b7710907022b0d035baea9001d529ca72f",
		HostNodeID: "openw-server",
		ConnectType: owtp.HTTP,
		Address: "47.52.191.89",
		EnableSignature: true,
		Cert: cert,
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

	api.CreateWallet(name, key.KeyID, hdkeystore.OpenwCoinTypePath, true,
		func(status uint64, msg string, wallet *Wallet) {
		if wallet != nil {
			t.Logf("wallet: %+v\n", wallet)
		}
	})
}

func TestAPINode_FindAccountByWalletID(t *testing.T) {
	walletID := "VysrzgpsLsgDpHM2KQMYuPY57fL3BAFU34"
	api := testNewAPINode()
	api.FindWalletByWalletID(walletID, true,
		func(status uint64, msg string, wallet *Wallet) {
			if wallet != nil {
				t.Logf("wallet: %+v\n", wallet)
			}
		})
}
