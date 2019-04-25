package openwsdk

import (
	"encoding/json"
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

func TestSummaryTaskUnmarshal(t *testing.T) {
	plain := `

{
	"wallets": [{
		"walletID": "WN84dVZXpgVixsvXnU8jkFWD1qWHp15LpA",
		"password": "12345678",
		"accounts": [
		{
			"accountID": "A3Mxhqm65kTgS2ybHLenNrZzZNtLGVobDFYdpc1ge4eK",
			"threshold": "1000",              
			"minTransfer": "1000",            
			"retainedBalance": "0",           
			"confirms": 1,                  
			"feeRate": "0.0001",            
        	"onlyContracts": false,         
          	"contracts": {                          
   				"all": {                            
        			"threshold": "1000",             
          			"minTransfer": "1000",            
           			"retainedBalance": "0"            
          		},         
          		"3qoe2ll2=": {                      
					"threshold": "1000",      
      				"minTransfer": "1000",
					"retainedBalance": "0"
               	}
			},
			"feesSupportAccount": {         
				"accountID": "12323",       
				"lowBalanceWarning": "0.1"  
			}
		},
		{
			"accountID": "3i26MQmtuWVVnw8GnRCVopG3pi8MaYU6RqWVV2E1hwJx",
			"feeRate": "0.001"
		}
		]
	}]
}

`
	var summaryTask SummaryTask
	err := json.Unmarshal([]byte(plain), &summaryTask)
	if err != nil {
		log.Error("json.Unmarshal error:", err)
		return
	}
}