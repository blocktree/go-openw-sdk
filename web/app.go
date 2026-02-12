package webapp

import (
	"github.com/blocktree/go-openw-sdk/v2/web/dto"
	impl "github.com/blocktree/go-openw-sdk/v2/web/service"
	"github.com/godaddy-x/freego/node"
	"github.com/godaddy-x/freego/utils"
	"github.com/godaddy-x/freego/utils/sdk"
)

var AppService = impl.AppService{}

func (s *WebNode) PublicKey(ctx *node.Context) error {
	pub, err := ctx.CreatePublicKey()
	if err != nil {
		return err
	}
	return s.Json(ctx, pub)
}

func (s *WebNode) FindWalletList(ctx *node.Context) error {
	req := &dto.FindWalletListReq{}
	if err := ctx.Parser(req); err != nil {
		return err
	}
	res := &dto.FindWalletListRes{}
	if err := AppService.FindWalletList(req, res); err != nil {
		return err
	}
	return s.Json(ctx, res)
}

func (s *WebNode) CreateWallet(ctx *node.Context) error {
	res := &dto.CreateWalletRes{}
	if err := AppService.CreateWallet(ctx.GetHeader("alias"), res); err != nil {
		return err
	}
	return s.Text(ctx, "wallet create success: "+res.WalletID)
}

func (s *WebNode) UnlockWallet(ctx *node.Context) error {
	res := &dto.UnlockWalletRes{}
	if err := AppService.UnlockWallet(ctx.GetHeader("filename"), res); err != nil {
		return err
	}
	return s.Text(ctx, "wallet unlock success: "+res.WalletID)
}

func (s *WebNode) Login(ctx *node.Context) error {
	req := &dto.AppLoginReq{}
	if err := ctx.Parser(req); err != nil {
		return err
	}
	res := &dto.AppLoginRes{}
	if err := AppService.AppLogin(req, res); err != nil {
		return err
	}
	config := ctx.GetJwtConfig()
	token := ctx.Subject.Create(res.Subject).Dev("API").Generate(config)
	secret := ctx.Subject.GetTokenSecret(token, config.TokenKey)
	expired := ctx.Subject.Payload.Exp
	return s.Json(ctx, &sdk.AuthToken{Token: token, Secret: utils.Base64Encode(secret), Expired: expired})
}

func (s *WebNode) CreateAccount(ctx *node.Context) error {
	req := &dto.CreateAccountReq{}
	if err := ctx.Parser(req); err != nil {
		return err
	}
	res := &dto.CreateAccountRes{}
	if err := AppService.CreateAccount(req, res); err != nil {
		return err
	}
	return s.Json(ctx, res)
}
