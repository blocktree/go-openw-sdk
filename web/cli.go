package webapp

import (
	"github.com/blocktree/go-openw-sdk/v2/openwsdk/dto"
	"github.com/blocktree/go-openw-sdk/v2/web/common"
	impl "github.com/blocktree/go-openw-sdk/v2/web/service"
	"github.com/godaddy-x/freego/ex"
	"github.com/godaddy-x/freego/node"
	"github.com/godaddy-x/freego/utils"
	"github.com/godaddy-x/freego/utils/sdk"
)

var CliService = impl.CliService{}

func (s *WebNode) PublicKey(ctx *node.Context) error {
	pub, err := ctx.CreatePublicKey()
	if err != nil {
		return err
	}
	return s.Json(ctx, pub)
}

func (s *WebNode) FindWalletList(ctx *node.Context) error {
	req := &dto.CliFindWalletListReq{}
	if err := ctx.Parser(req); err != nil {
		return err
	}
	res := &dto.CliFindWalletListRes{}
	if err := CliService.FindWalletList(req, res); err != nil {
		return err
	}
	return s.Json(ctx, res)
}

func (s *WebNode) Login(ctx *node.Context) error {
	req := &dto.AppLoginReq{}
	if err := ctx.Parser(req); err != nil {
		return err
	}
	res := &dto.AppLoginRes{}
	if err := CliService.CliLogin(req, res); err != nil {
		return err
	}
	config := ctx.GetJwtConfig()
	token := ctx.Subject.Create(res.Subject).Dev("API").Generate(config)
	secret := ctx.Subject.GetTokenSecret(token, config.TokenKey)
	expired := ctx.Subject.Payload.Exp
	return s.Json(ctx, &sdk.AuthToken{Token: token, Secret: utils.Base64Encode(secret), Expired: expired})
}

func (s *WebNode) NodeLogin(ctx *node.Context) error {
	req := &dto.AppLoginReq{}
	if err := ctx.Parser(req); err != nil {
		return err
	}
	res := &dto.AppLoginRes{}
	if err := CliService.CliLogin(req, res); err != nil {
		return err
	}
	whitelist := common.GetAllConfig().Extract.NodeWhitelist
	nodeIP := ctx.RequestCtx.RemoteIP().String()
	if !utils.CheckStr(nodeIP, whitelist...) {
		return ex.Throw{Code: 400, Msg: "bad request"}
	}
	config := ctx.GetJwtConfig()
	token := ctx.Subject.Create(req.Source).Dev("NODE").Generate(config)
	secret := ctx.Subject.GetTokenSecret(token, config.TokenKey)
	expired := ctx.Subject.Payload.Exp
	return s.Json(ctx, &sdk.AuthToken{Token: token, Secret: utils.Base64Encode(secret), Expired: expired})
}

func (s *WebNode) CreateAccount(ctx *node.Context) error {
	req := &dto.CliCreateAccountReq{}
	if err := ctx.Parser(req); err != nil {
		return err
	}
	res := &dto.CliCreateAccountRes{}
	if err := CliService.CreateAccount(req, res); err != nil {
		return err
	}
	return s.Json(ctx, res)
}

func (s *WebNode) SignTransaction(ctx *node.Context) error {
	req := &dto.CliSignTransactionReq{}
	if err := ctx.Parser(req); err != nil {
		return err
	}
	res := &dto.CliSignTransactionRes{}
	if err := CliService.SignTransaction(req, res); err != nil {
		return err
	}
	return s.Json(ctx, res)
}

func (s *WebNode) SignTradeKey(ctx *node.Context) error {
	req := &dto.CliSignTradeKeyReq{}
	if err := ctx.Parser(req); err != nil {
		return err
	}
	res := &dto.CliSignTradeKeyRes{}
	if err := CliService.SignTradeKey(req, res); err != nil {
		return err
	}
	return s.Json(ctx, res)
}
