package impl

import (
	"bytes"
	"encoding/hex"
	"github.com/blocktree/go-openw-sdk/v2/web/common"
	"github.com/blocktree/go-openw-sdk/v2/web/dto"
	DIC "github.com/godaddy-x/freego/common"
	"github.com/godaddy-x/freego/ex"
	"github.com/godaddy-x/freego/utils"
	"github.com/godaddy-x/freego/utils/jwt"
)

type AppService struct{}

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

func (s *AppService) UnlockWallet(req *dto.UnlockWalletReq, res *dto.UnlockWalletRes) error {

	return nil
}
