package impl

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/awnumar/memguard"
	"github.com/blocktree/go-openw-sdk/v2/openwsdk"
	"github.com/blocktree/go-openw-sdk/v2/web/common"
	"github.com/blocktree/go-openw-sdk/v2/web/dto"
	"github.com/blocktree/openwallet/v2/hdkeystore"
	DIC "github.com/godaddy-x/freego/common"
	"github.com/godaddy-x/freego/ex"
	"github.com/godaddy-x/freego/utils"
	"github.com/godaddy-x/freego/utils/jwt"
	"github.com/godaddy-x/freego/zlog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
)

type AppService struct {
	pending atomic.Bool
}

const (
	// 设置别名(alias)和文件名(filename)的最大最小长度
	minAliasLength = 1
	maxAliasLength = 255

	// 设置认证(auth)密码的最大最小长度
	minAuthLength = 20
	maxAuthLength = 256 // 根据你的业务需求调整这个值
)

var (
	aad     = utils.GetRandomSecure(32)
	aadCall = func(keyID string) ([]byte, error) {
		return aad, nil
	}
)

func (s *AppService) UnlockWallet(filename string, res *dto.UnlockWalletRes) error {
	// === 1. 校验 filename 格式（保留原有逻辑）===
	if strings.TrimSpace(filename) == "" {
		return ex.Throw{Code: ex.BIZ, Msg: "filename is required"}
	}

	if len(filename) < minAliasLength+len(".key") || len(filename) > maxAliasLength+len("-")+maxAliasLength+len(".key") {
		return ex.Throw{Code: ex.BIZ, Msg: fmt.Sprintf("filename length must be between %d and %d", minAliasLength+len(".key"), maxAliasLength+len("-")+maxAliasLength+len(".key"))}
	}

	if !strings.HasSuffix(filename, ".key") {
		return ex.Throw{Code: ex.BIZ, Msg: "filename must end with .key"}
	}

	namePart := strings.TrimSuffix(filename, ".key")
	parts := strings.Split(namePart, "-")
	if len(parts) != 2 {
		return ex.Throw{Code: ex.BIZ, Msg: "filename must be in format: alias-keyID.key"}
	}

	aliasPart, keyIDPart := parts[0], parts[1]
	if aliasPart == "" || keyIDPart == "" {
		return ex.Throw{Code: ex.BIZ, Msg: "alias and keyID cannot be empty"}
	}

	if len(aliasPart) < minAliasLength || len(aliasPart) > maxAliasLength ||
		len(keyIDPart) < minAliasLength || len(keyIDPart) > maxAliasLength {
		return ex.Throw{Code: ex.BIZ, Msg: fmt.Sprintf("alias/keyID length must be between %d and %d", minAliasLength, maxAliasLength)}
	}

	alphaNum := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	if !alphaNum.MatchString(aliasPart) || !alphaNum.MatchString(keyIDPart) {
		return ex.Throw{Code: ex.BIZ, Msg: "alias and keyID must contain only letters and digits"}
	}

	// === 2. 并发控制 ===
	if !s.pending.CompareAndSwap(false, true) {
		return ex.Throw{Code: ex.BIZ, Msg: "operation in progress"}
	}
	defer s.pending.Store(false)

	// === 3. 读取统一密码文件（关键：固定路径）===
	pwd := common.GetAllConfig().Extract.PasswordKey // 复用 CreateWallet 的配置
	if pwd == "" {
		return ex.Throw{Code: ex.BIZ, Msg: "password file path not configured"}
	}

	file, err := os.Open(pwd)
	if err != nil {
		return ex.Throw{Code: ex.BIZ, Msg: "password file not ready"}
	}
	defer file.Close()

	// === 4. 安全加载到锁定内存 ===
	authBuf, err := memguard.NewBufferFromEntireReader(file)
	if err != nil {
		zlog.Error("failed to load unlock password into secure memory", 0,
			zlog.String("filename", filename))
		return ex.Throw{Code: ex.BIZ, Msg: "password file read error"}
	}
	defer authBuf.Destroy()

	if authBuf.Size() < minAuthLength || authBuf.Size() > maxAuthLength {
		return ex.Throw{Code: ex.BIZ, Msg: "password length must be between 20 and 256 characters"}
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

func (s *AppService) CreateWallet(alias string, res *dto.CreateWalletRes) error {
	// === 参数校验（alias）===
	if strings.TrimSpace(alias) == "" || !regexp.MustCompile(`^[a-zA-Z0-9]+$`).MatchString(alias) {
		return ex.Throw{Code: ex.BIZ, Msg: "invalid alias"}
	}

	// === 并发控制 ===
	if !s.pending.CompareAndSwap(false, true) {
		return ex.Throw{Code: ex.BIZ, Msg: "operation in progress"}
	}
	defer s.pending.Store(false)

	pwd := common.GetAllConfig().Extract.PasswordKey
	if pwd == "" {
		return ex.Throw{Code: ex.BIZ, Msg: "password file path is nil"}
	}
	// 1. 打开文件
	file, err := os.Open(pwd)
	if err != nil {
		return ex.Throw{Code: ex.BIZ, Msg: "password file not found"}
	}
	defer file.Close()

	// 3. 直接读入锁定内存
	authBuf, err := memguard.NewBufferFromEntireReader(file)
	if err != nil {
		// 注意：不记录原始 err 的完整文本，防止泄露路径等敏感信息
		zlog.Error("failed to load password into secure memory", 0,
			zlog.String("alias", alias))
		return ex.Throw{Code: ex.BIZ, Msg: "password file read error"}
	}
	defer authBuf.Destroy()

	if authBuf.Size() < minAuthLength || authBuf.Size() > maxAuthLength {
		return ex.Throw{Code: ex.BIZ, Msg: "password length must be between 20 and 256 characters"}
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

func (s *AppService) AppLogin(req *dto.AppLoginReq, res *dto.AppLoginRes) error {
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

func (s *AppService) FindWalletList(req *dto.FindWalletListReq, res *dto.FindWalletListRes) error {
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

func (s *AppService) CreateAccount(req *dto.CreateAccountReq, res *dto.CreateAccountRes) error {
	if req.WalletID == "" {
		return ex.Throw{Code: ex.BIZ, Msg: "walletID is nil"}
	}
	key := openwsdk.GetUnlockWallet(req.WalletID)
	if key == nil {
		return ex.Throw{Code: ex.BIZ, Msg: "walletID is nil or unlock: " + req.WalletID}
	}
	return nil
}
