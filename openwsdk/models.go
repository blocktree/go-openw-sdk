package openwsdk

import (
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