package openwsdk

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/blocktree/go-openw-sdk/v2/openwsdk/dto"
	"github.com/blocktree/openwallet/v2/hdkeystore"
	"github.com/blocktree/openwallet/v2/openwallet"
	DIC "github.com/godaddy-x/freego/common"
	"github.com/godaddy-x/freego/utils"
	"github.com/godaddy-x/freego/utils/sdk"
	"strings"
	"sync"
)

type SdkConfig struct {
	Domain    string `json:"domain"`
	AppID     string `json:"appID"`
	AppKey    string `json:"appKey"`
	ClientPrk string `json:"clientPrk"`
	ServerPub string `json:"serverPub"`
	ClientNo  int64  `json:"clientNo"`
	TradeKey  string `json:"tradeKey"`
}

func NewHttpSDK(config SdkConfig) *sdk.HttpSDK {
	newObject := &sdk.HttpSDK{
		Domain:    config.Domain,
		KeyPath:   "/api/PublicKey",
		LoginPath: "/api/Login",
	}
	clientPrk := config.ClientPrk
	serverPub := config.ServerPub
	newObject.SetClientNo(1)
	_ = newObject.SetECDSAObject(newObject.ClientNo, clientPrk, serverPub)
	newObject.AuthObject(func() (interface{}, error) {
		requestData := dto.AppLoginReq{
			AppID: config.AppID,
			Nonce: utils.Base64Encode(utils.GetRandomSecure(32)),
			Time:  utils.UnixSecond(),
		}
		h, err := hex.DecodeString(config.AppKey)
		if err != nil {
			return nil, err
		}
		requestData.Sign = utils.Base64Encode(utils.HMAC_SHA256_BASE(h, utils.Str2Bytes(utils.AddStr(requestData.Nonce, requestData.Time))))
		return requestData, nil
	})
	return newObject
}

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

func GetUnlockWalletSize() int {
	unlocked.mu.Lock()
	defer unlocked.mu.Unlock()
	return len(unlocked.wallet)
}

func DestroyUnlockWallet() {
	unlocked.mu.Lock()
	defer unlocked.mu.Unlock()
	for _, v := range unlocked.wallet {
		v.DestroySeed()
	}
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

func CheckTxDataSign(appKey string, txData []*dto.TxData) error {
	if len(txData) == 0 {
		return errors.New("tx data is nil")
	}
	h, err := hex.DecodeString(appKey)
	if err != nil {
		return err
	}
	for _, v := range txData {
		checkSign := utils.HMAC_SHA256_BASE(utils.Str2Bytes(v.Data), h)
		if v.DataSign != utils.Base64Encode(checkSign) {
			return errors.New(fmt.Sprintf("tx data check sign invalid: %s", v.Data))
		}
	}
	return nil
}

func CheckTxTradeSign(tradeKey string, txData []*dto.TxData) error {
	if len(txData) == 0 {
		return errors.New("tx data is nil")
	}
	h, err := hex.DecodeString(tradeKey)
	if err != nil {
		return err
	}
	for _, v := range txData {
		checkSign := utils.HMAC_SHA256_BASE(utils.Str2Bytes(v.Data), h)
		if v.TradeSign != utils.Base64Encode(checkSign) {
			return errors.New(fmt.Sprintf("tx data check trade sign invalid: %s", v.Data))
		}
	}
	return nil
}

func CheckOneTxTradeSign(tradeKey, data, sign string) error {
	if len(data) == 0 {
		return errors.New("tx data is nil")
	}
	h, err := hex.DecodeString(tradeKey)
	if err != nil {
		return err
	}
	checkSign := utils.HMAC_SHA256_BASE(utils.Str2Bytes(data), h)
	if sign != utils.Base64Encode(checkSign) {
		return errors.New(fmt.Sprintf("tx data check trade sign invalid: %s, %s", data, sign))
	}
	return nil
}
