package impl

import (
	"bytes"
	"encoding/hex"
	"github.com/awnumar/memguard"
	"github.com/blocktree/go-openw-sdk/v2/openwsdk"
	"github.com/blocktree/go-openw-sdk/v2/openwsdk/dto"
	"github.com/blocktree/go-openw-sdk/v2/web/common"
	"github.com/blocktree/openwallet/v2/hdkeystore"
	"github.com/blocktree/openwallet/v2/openwallet"
	DIC "github.com/godaddy-x/freego/common"
	"github.com/godaddy-x/freego/ex"
	"github.com/godaddy-x/freego/utils"
	"github.com/godaddy-x/freego/utils/jwt"
	"github.com/godaddy-x/freego/zlog"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
)

type CliService struct {
	pending atomic.Bool
}

const (
	// 设置别名(alias)和文件名(filename)的最大最小长度
	minAliasLength = 1
	maxAliasLength = 255

	// 设置认证(auth)密码的最大最小长度
	minAuthLength = 8
	maxAuthLength = 256 // 根据你的业务需求调整这个值
)

var (
	aad     = utils.GetRandomSecure(32)
	aadCall = func(keyID string) ([]byte, error) {
		return aad, nil
	}
)

func (s *CliService) UnlockWallet(filename string, password []byte, res *dto.CliUnlockWalletRes) error {

	// === 2. 并发控制 ===
	if !s.pending.CompareAndSwap(false, true) {
		return ex.Throw{Code: ex.BIZ, Msg: "operation in progress"}
	}
	defer s.pending.Store(false)

	// === 4. 安全加载到锁定内存 ===
	authBuf := memguard.NewBufferFromBytes(password)
	defer authBuf.Destroy()
	DIC.ClearData(password)

	if authBuf.Size() < minAuthLength || authBuf.Size() > maxAuthLength {
		return ex.Throw{Code: ex.BIZ, Msg: "password length must be between 8 and 256 characters"}
	}

	// === 5. 执行解锁 ===
	ks := &hdkeystore.HDKeystore{}
	walletPath := ks.JoinDirPath(filepath.Join(".", common.GetAllConfig().Extract.WalletDir), filename)

	key, err := ks.GetLockerKey(walletPath, authBuf, aadCall)
	if err != nil {
		zlog.Error("unlock wallet failed", 0,
			zlog.String("filename", filename))
		// 不暴露具体原因（防侧信道）
		return ex.Throw{Code: ex.BIZ, Msg: "unlock failed: incorrect password or invalid wallet"}
	}

	openwsdk.AddUnlockWallet(key)
	res.WalletID = key.KeyID
	return nil
}

func (s *CliService) CreateWallet(alias string, password []byte, res *dto.CliCreateWalletRes) error {
	// === 参数校验（alias）===
	if strings.TrimSpace(alias) == "" || !regexp.MustCompile(`^[a-zA-Z0-9]+$`).MatchString(alias) {
		return ex.Throw{Code: ex.BIZ, Msg: "invalid alias"}
	}

	// === 并发控制 ===
	if !s.pending.CompareAndSwap(false, true) {
		return ex.Throw{Code: ex.BIZ, Msg: "operation in progress"}
	}
	defer s.pending.Store(false)

	// 3. 直接读入锁定内存
	authBuf := memguard.NewBufferFromBytes(password)
	defer authBuf.Destroy()
	DIC.ClearData(password)

	if authBuf.Size() < minAuthLength || authBuf.Size() > maxAuthLength {
		return ex.Throw{Code: ex.BIZ, Msg: "password length must be between 8 and 256 characters"}
	}

	// === 4. 执行创建逻辑 ===
	config := common.GetAllConfig()
	path := filepath.Join(".", config.Extract.WalletDir)
	rootID, err := hdkeystore.StoreLockerHDKey(path, alias, authBuf)
	if err != nil {
		zlog.Error("create wallet failed", 0, zlog.AddError(err))
		return ex.Throw{Code: ex.BIZ, Msg: err.Error()}
	}

	res.WalletID = rootID
	return nil
}

func (s *CliService) CliLogin(req *dto.AppLoginReq, res *dto.AppLoginRes) error {
	if len(req.AppID) == 0 {
		return ex.Throw{Code: ex.BIZ, Msg: "appID is empty"}
	}
	if len(req.Sign) == 0 {
		return ex.Throw{Code: ex.BIZ, Msg: "sign is empty"}
	}
	if len(req.Nonce) == 0 {
		return ex.Throw{Code: ex.BIZ, Msg: "nonce is empty"}
	}
	if utils.MathAbs(utils.UnixSecond()-req.Time) > jwt.FIVE_MINUTES {
		return ex.Throw{Code: ex.BIZ, Msg: "time invalid"}
	}
	// 通过配置的服务端秘钥解码应用的KEY，进行签名验证
	config := common.GetAllConfig()
	// 方法结束清除内存中的密钥
	decrypt, err := hex.DecodeString(config.Extract.AppKey)
	if err != nil {
		return ex.Throw{Code: ex.DATA, Msg: "api password decoder invalid", Err: err}
	}
	defer DIC.ClearData(decrypt)
	// 使用应用密钥验签失败则响应错误
	if !bytes.Equal(utils.HMAC_SHA256_BASE(decrypt, utils.Str2Bytes(utils.AddStr(req.Nonce, req.Time))), utils.Base64Decode(req.Sign)) {
		return ex.Throw{Code: ex.BIZ, Msg: "sign invalid"}
	}
	res.Subject = config.Extract.AppID
	return nil
}

func (s *CliService) FindWalletList(req *dto.CliFindWalletListReq, res *dto.CliFindWalletListRes) error {
	config := common.GetAllConfig().Extract
	fileList, err := common.ReadAllFilesInDir(config.WalletDir)
	if err != nil {
		return err
	}
	res.Result = make([]dto.WalletResult, 0, len(fileList))
	for _, v := range fileList {
		res.Result = append(res.Result, dto.WalletResult{
			Alias:    v.Alias,
			WalletID: v.KeyID,
			RootPath: v.RootPath,
		})
	}
	return nil
}

func (s *CliService) CreateAccount(req *dto.CliCreateAccountReq, res *dto.CliCreateAccountRes) error {
	if req.WalletID == "" {
		return ex.Throw{Code: ex.BIZ, Msg: "walletID is nil"}
	}
	if req.LastIndex < -1 {
		return ex.Throw{Code: ex.BIZ, Msg: "lastIndex invalid"}
	}
	if req.Curve < 0 {
		return ex.Throw{Code: ex.BIZ, Msg: "curve invalid"}
	}
	key := openwsdk.GetUnlockWallet(req.WalletID)
	if key == nil {
		return ex.Throw{Code: ex.BIZ, Msg: "walletID is nil or unlock: " + req.WalletID}
	}
	account, err := openwsdk.DerivedAccount(key, req.LastIndex, req.Curve)
	if err != nil {
		return ex.Throw{Code: ex.BIZ, Msg: "create account error", Err: err}
	}
	res.WalletID = account.WalletID
	res.AccountID = account.AccountID
	res.OtherOwnerKeys = account.OtherOwnerKeys
	res.ReqSigs = account.ReqSigs
	res.PublicKey = account.PublicKey
	res.HdPath = account.HdPath
	res.AccountIndex = account.AccountIndex
	res.AddressIndex = account.AddressIndex
	return nil
}

func (s *CliService) SignTransaction(req *dto.CliSignTransactionReq, res *dto.CliSignTransactionRes) error {
	if req.Data == "" {
		return ex.Throw{Code: ex.BIZ, Msg: "data is nil"}
	}
	if req.TradeSign == "" {
		return ex.Throw{Code: ex.BIZ, Msg: "tradeSign is nil"}
	}
	if !utils.CheckInt64(req.Type, 0, 1) {
		return ex.Throw{Code: ex.BIZ, Msg: "type invalid"}
	}

	if err := openwsdk.CheckOneTxTradeSign(common.GetAllConfig().Extract.TradeKey, req.Data, req.TradeSign); err != nil {
		return ex.Throw{Code: ex.BIZ, Msg: "trade sign invalid", Err: err}
	}

	var tx *openwallet.RawTransaction

	if req.Type == 0 {
		tx = &openwallet.RawTransaction{}
		if err := utils.JsonUnmarshal(utils.Str2Bytes(req.Data), tx); err != nil {
			return ex.Throw{Code: ex.BIZ, Msg: "tx decode error", Err: err}
		}
	} else {
		txErr := &openwallet.RawTransactionWithError{}
		if err := utils.JsonUnmarshal(utils.Str2Bytes(req.Data), txErr); err != nil {
			return ex.Throw{Code: ex.BIZ, Msg: "tx decode error", Err: err}
		}
		if txErr.Error != nil {
			return ex.Throw{Code: ex.BIZ, Msg: "tx error: " + txErr.Error.Error()}
		}

		tx = txErr.RawTx
	}

	if tx.TxType != req.Type {

	}

	if utils.UnixMilli()-tx.CreateTime > 86400000 {
		return ex.Throw{Code: ex.BIZ, Msg: "tx create time invalid"}
	}

	if tx.TxType == 0 { // 普通交易单，校验黑名单
		blacklist := common.GetAllConfig().Extract.SubmitBlacklist
		for to, _ := range tx.To {
			if utils.CheckStr(to, blacklist...) {
				return ex.Throw{Code: ex.BIZ, Msg: "tx submit blacklist invalid: " + to}
			}
		}
	} else if tx.TxType == 1 { // 汇总交易单，校验白名单
		if len(tx.To) > 1 {
			return ex.Throw{Code: ex.BIZ, Msg: "tx submit target address > 1 invalid"}
		}
		whitelist := common.GetAllConfig().Extract.SummaryWhitelist
		for to, _ := range tx.To {
			if !utils.CheckStr(to, whitelist...) {
				return ex.Throw{Code: ex.BIZ, Msg: "tx submit blacklist invalid: " + to}
			}
		}
	} else {
		return ex.Throw{Code: ex.BIZ, Msg: "tx type invalid"}
	}

	key := openwsdk.GetUnlockWallet(tx.Account.WalletID)
	if key == nil {
		return ex.Throw{Code: ex.BIZ, Msg: "walletID is nil or unlock: " + tx.Account.WalletID}
	}

	txSignerList := map[string]string{}
	if err := openwsdk.SignRawTransactionExtract(tx, key, txSignerList); err != nil {
		return ex.Throw{Code: ex.BIZ, Msg: "sign tx error: " + tx.Account.WalletID, Err: err}
	}
	res.SignerList = txSignerList

	return nil
}
