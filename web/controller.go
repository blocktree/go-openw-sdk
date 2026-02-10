package webapp

import (
	"github.com/blocktree/go-openw-sdk/v2/web/common"
	"github.com/godaddy-x/freego/ex"
	ballast "github.com/godaddy-x/freego/gc"
	"github.com/godaddy-x/freego/node"
	"github.com/godaddy-x/freego/utils"
	"github.com/godaddy-x/freego/utils/crypto"
	"github.com/godaddy-x/freego/utils/jwt"
	"github.com/godaddy-x/freego/zlog"
)

const (
	project = "cli_main"
)

type WebNode struct {
	node.HttpNode
}

func api(key string) string {
	return utils.AddStr("/api/", key)
}
func addr() string {
	config := common.GetAllConfig().GetServerConfig(project)
	return utils.AddStr(config.Addr, ":", config.Port)
}

func newHTTP() *WebNode {

	var web = &WebNode{}

	// 添加JWT配置参数
	jwtConfig := common.GetAllConfig().GetJwtConfig(project)
	_ = web.AddJwtConfig(jwt.JwtConfig{TokenKey: jwtConfig.TokenKey, TokenAlg: jwtConfig.TokenAlg, TokenExp: jwtConfig.TokenExp, TokenTyp: jwtConfig.TokenTyp})

	// 添加系统基本参数
	serverConfig := common.GetAllConfig().GetServerConfig(project)
	web.SetSystem(serverConfig.Name, serverConfig.Version)

	// 配置基础GC优化参数
	ballast.GC(serverConfig.GCLimitMB*ballast.MB, serverConfig.GCPercent)

	// 添加ECDSA配置参数,服务端私钥和客户端公钥
	for _, v := range serverConfig.Keys {
		cipher, err := crypto.CreateS256ECDSAWithBase64(v.PrivateKey, v.PublicKey)
		if err != nil {
			panic("create ecdsa object error: " + err.Error())
		}
		_ = web.AddCipher(v.Name, cipher)
	}

	// 添加Local缓存参数,默认空即可
	web.AddLocalCache(nil)

	_ = web.AddErrorHandle(func(ctx *node.Context, throw ex.Throw) error {
		errMsg := throw.ErrMsg
		if throw.Err != nil {
			errMsg = throw.Err.Error()
		}
		zlog.Error("AddErrorHandle catcher", 0, zlog.String("path", ctx.Path), zlog.String("bizMsg", throw.Msg), zlog.String("errMsg", errMsg))
		return nil
	})

	return web
}

func StartHttpNode() {

	// 初始化配置文件
	common.NewBaseConfig("cli-http")

	// 创建API服务
	web := newHTTP()

	web.POST(api("PublicKey"), web.PublicKey, &node.RouterConfig{Guest: true})
	web.POST(api("Login"), web.Login, &node.RouterConfig{UseRSA: true})
	web.POST(api("FindWalletList"), web.FindWalletList, &node.RouterConfig{AesRequest: true, AesResponse: true})
	web.POST(api("CreateWallet"), web.CreateWallet, &node.RouterConfig{Guest: true})
	web.POST(api("UnlockWallet"), web.UnlockWallet, &node.RouterConfig{Guest: true})

	web.StartServer(addr())
}
