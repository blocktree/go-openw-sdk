package openwsdk

import (
	"encoding/hex"
	"fmt"
	"github.com/blocktree/go-openw-sdk/v2/openwsdk/dto"
	"github.com/blocktree/openwallet/v2/hdkeystore"
	"github.com/blocktree/openwallet/v2/openwallet"
	DIC "github.com/godaddy-x/freego/common"
	"github.com/godaddy-x/freego/utils"
	"strings"
	"sync"
)

type unlockWallet struct {
	mu     sync.Mutex
	wallet map[string]*hdkeystore.HDKey
}

var (
	unlocked = &unlockWallet{
		wallet: make(map[string]*hdkeystore.HDKey, 10),
	}
)

func AddUnlockWallet(key *hdkeystore.HDKey) {
	unlocked.mu.Lock()
	defer unlocked.mu.Unlock()
	unlocked.wallet[key.KeyID] = key
}

func GetUnlockWallet(keyID string) *hdkeystore.HDKey {
	unlocked.mu.Lock()
	defer unlocked.mu.Unlock()
	return unlocked.wallet[keyID]
}

func SignTradePush(key string, data dto.TradePushResult) (string, error) {
	h, err := hex.DecodeString(key)
	if err != nil {
		return "", err
	}
	defer DIC.ClearData(h)
	hashKey := utils.SHA256_BASE(h)
	var signData strings.Builder
	signData.WriteString(utils.AnyToStr(data.ID))
	signData.WriteString("|")
	signData.WriteString(data.AppID)
	signData.WriteString("|")
	signData.WriteString(data.Data)
	signData.WriteString("|")
	signData.WriteString(utils.AnyToStr(data.Ctime))
	return utils.Base64Encode(utils.HMAC_SHA256_BASE(utils.Str2Bytes(signData.String()), hashKey)), nil
}

func SignTradeBalancePush(key string, data dto.TradeBalancePushResult) (string, error) {
	h, err := hex.DecodeString(key)
	if err != nil {
		return "", err
	}
	defer DIC.ClearData(h)
	hashKey := utils.SHA256_BASE(h)
	var signData strings.Builder
	signData.WriteString(utils.AnyToStr(data.ID))
	signData.WriteString("|")
	signData.WriteString(data.AppID)
	signData.WriteString("|")
	signData.WriteString(data.Data)
	signData.WriteString("|")
	signData.WriteString(utils.AnyToStr(data.Ctime))
	return utils.Base64Encode(utils.HMAC_SHA256_BASE(utils.Str2Bytes(signData.String()), hashKey)), nil
}

func DerivedAccount(key *hdkeystore.HDKey, lastIndex, curve int64) (*dto.AccountResult, error) {
	account := &dto.AccountResult{}
	account.ReqSigs = 1
	account.AccountIndex = lastIndex + 1

	// root/n' , 使用强化方案
	account.HdPath = fmt.Sprintf("%s/%d'", key.RootPath, account.AccountIndex)

	childKey, err := key.DerivedKeyWithPath(account.HdPath, uint32(curve))
	if err != nil {
		return nil, err
	}

	account.PublicKey = childKey.GetPublicKey().OWEncode()
	account.AccountID = openwallet.GenAccountID(account.PublicKey)
	account.AddressIndex = -1
	account.WalletID = key.KeyID

	return account, nil
}
