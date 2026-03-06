package webapp

import (
	"errors"
	"fmt"

	"github.com/awnumar/memguard"
	"github.com/blocktree/go-openw-sdk/v2/web/common"
	"github.com/codahale/sss"
)

func ShardSeed() error {
	config := common.GetAllConfig().Extract
	secret := memguard.NewBufferRandom(64) // our secret
	var n, k byte
	if config.WalletMode == 3 {
		n = byte(3)
		k = byte(2)
	} else if config.WalletMode == 5 {
		n = byte(5)
		k = byte(3)
	} else {
		return errors.New("wallet mode not supported")
	}

	shares, err := sss.Split(n, k, secret.Bytes())
	if err != nil {
		return err
	}

	// TODO send share to target
	fmt.Println(shares)

	return err
}
