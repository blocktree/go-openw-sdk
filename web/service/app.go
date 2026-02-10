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

func (s *AppService) UnlockWallet(filename string, auth []byte, res *dto.UnlockWalletRes) error {
	// === 参数校验 ===
	if strings.TrimSpace(filename) == "" {
		return ex.Throw{Code: ex.BIZ, Msg: "filename is required"}
	}

	if len(filename) < minAliasLength+len(".key") || len(filename) > maxAliasLength+len("-")+maxAliasLength+len(".key") {
		return ex.Throw{Code: ex.BIZ, Msg: fmt.Sprintf("filename length must be between %d and %d", minAliasLength+len(".key"), maxAliasLength+len("-")+maxAliasLength+len(".key"))}
	}

	// 校验 filename 格式: alias-keyID.key
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

	if len(aliasPart) < minAliasLength || len(aliasPart) > maxAliasLength {
		return ex.Throw{Code: ex.BIZ, Msg: fmt.Sprintf("alias length must be between %d and %d", minAliasLength, maxAliasLength)}
	}

	if len(keyIDPart) < minAliasLength || len(keyIDPart) > maxAliasLength {
		return ex.Throw{Code: ex.BIZ, Msg: fmt.Sprintf("keyID length must be between %d and %d", minAliasLength, maxAliasLength)}
	}

	alphaNum := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	if !alphaNum.MatchString(aliasPart) {
		return ex.Throw{Code: ex.BIZ, Msg: "alias must contain only letters and digits"}
	}
	if !alphaNum.MatchString(keyIDPart) {
		return ex.Throw{Code: ex.BIZ, Msg: "keyID must contain only letters and digits"}
	}

	if len(auth) < minAuthLength || len(auth) > maxAuthLength {
		return ex.Throw{Code: ex.BIZ, Msg: fmt.Sprintf("auth password length must be between %d and %d", minAuthLength, maxAuthLength)}
	}

	// === 并发控制：单槽位拒绝式 ===
	if s.pending.CompareAndSwap(false, true) {
		defer s.pending.Store(false)

		ks := &hdkeystore.HDKeystore{}
		path := ks.JoinDirPath(filepath.Join(".", common.GetAllConfig().Extract.WalletDir), filename)
		authBuf := memguard.NewBufferFromBytes(auth)
		defer authBuf.Destroy()

		key, err := ks.GetLockerKey(path, authBuf, aadCall)
		if err != nil {
			zlog.Error("unlock wallet error", 0, zlog.String("errMsg", err.Error()))
			return ex.Throw{Code: ex.BIZ, Msg: err.Error()}
		}
		openwsdk.AddUnlockWallet(key)
		res.KeyID = key.KeyID
		return nil
	}

	return ex.Throw{Code: ex.BIZ, Msg: "wallet in progress"}
}

func (s *AppService) CreateWallet(alias string, auth []byte, res *dto.CreateWalletRes) error {
	// === 参数校验 ===
	if strings.TrimSpace(alias) == "" {
		return ex.Throw{Code: ex.BIZ, Msg: "alias is required"}
	}

	if len(alias) < minAliasLength || len(alias) > maxAliasLength {
		return ex.Throw{Code: ex.BIZ, Msg: fmt.Sprintf("alias length must be between %d and %d", minAliasLength, maxAliasLength)}
	}

	if !regexp.MustCompile(`^[a-zA-Z0-9]+$`).MatchString(alias) {
		return ex.Throw{Code: ex.BIZ, Msg: "alias must contain only letters and digits"}
	}

	if len(auth) < minAuthLength || len(auth) > maxAuthLength {
		return ex.Throw{Code: ex.BIZ, Msg: fmt.Sprintf("auth password length must be between %d and %d", minAuthLength, maxAuthLength)}
	}

	// === 并发控制：单槽位拒绝式 ===
	if s.pending.CompareAndSwap(false, true) {
		defer s.pending.Store(false)

		config := common.GetAllConfig()
		path := filepath.Join(".", config.Extract.WalletDir)
		authBuf := memguard.NewBufferFromBytes(auth)
		defer authBuf.Destroy()

		rootID, err := hdkeystore.StoreLockerHDKey(path, alias, authBuf)
		if err != nil {
			zlog.Error("create wallet error", 0, zlog.String("errMsg", err.Error()))
			return ex.Throw{Code: ex.BIZ, Msg: err.Error()}
		}

		res.KeyID = rootID
		return nil
	}

	return ex.Throw{Code: ex.BIZ, Msg: "wallet in progress"}
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
			KeyID:    v.KeyID,
			RootPath: v.RootPath,
		})
	}
	return nil
}
