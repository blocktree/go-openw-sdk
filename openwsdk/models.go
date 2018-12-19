package openwsdk

import (
	"fmt"
	"github.com/blocktree/OpenWallet/hdkeystore"
	"github.com/blocktree/go-openw-server/model"
)

type Wallet struct {
	model.OwWallet
}

type Symbol struct {
	model.OwSymbol
}

type Account struct {
	model.OwAccount
}

type Address struct {
	model.OwAddress
}

func (wallet *Wallet) CreateAccount(alias string, symbol *Symbol, key *hdkeystore.HDKey) (*Account, error) {

	var (
		account = &Account{}
	)

	account.Alias = alias
	account.Symbol = symbol.Coin
	account.ReqSigs = 1

	newAccIndex := wallet.AccountIndex + 1

	// root/n' , 使用强化方案
	account.HdPath = fmt.Sprintf("%s/%d'", wallet.RootPath, newAccIndex)

	childKey, err := key.DerivedKeyWithPath(account.HdPath, uint32(symbol.Curve))
	if err != nil {
		return nil, err
	}

	account.PublicKey = childKey.GetPublicKey().OWEncode()
	account.AccountIndex = newAccIndex
	account.AccountID = account.GetAccountID()
	account.AddressIndex = -1
	account.WalletID = wallet.WalletID

	return account, nil

}
