package app

import (
	"net"

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
	config := GetAllConfig().GetServerConfig(project)
	return utils.AddStr(config.Addr, ":", config.Port)
}

type RemoteCheckFilter struct{}

func (self *RemoteCheckFilter) DoFilter(chain node.Filter, ctx *node.Context, args ...interface{}) error {
	remoteAddr := ctx.RequestCtx.RemoteAddr().String()
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return ex.Throw{Code: ex.BIZ, Msg: "forbidden"}
	}

	// 标准化 IPv4-mapped IPv6 地址（如 ::ffff:192.168.1.1 → 1972.168.1.1）
	if ip := net.ParseIP(host); ip != nil {
		if ipv4 := ip.To4(); ipv4 != nil {
			host = ipv4.String()
		}
	}

	// 检查远程白名单
	whitelist := GetAllConfig().Extract.RemoteWhitelist
	for _, ip := range whitelist {
		if host == ip {
			return chain.DoFilter(chain, ctx, args...)
		}
	}

	zlog.Error("remote access blocked: IP not in whitelist", 0,
		zlog.String("path", ctx.Path),
		zlog.String("remote_ip", host))

	return ex.Throw{Code: ex.BIZ, Msg: "forbidden"}
}

func NewHTTP() *WebNode {

	var web = &WebNode{}

	// 添加JWT配置参数
	jwtConfig := GetAllConfig().GetJwtConfig(project)
	_ = web.AddJwtConfig(jwt.JwtConfig{TokenKey: jwtConfig.TokenKey, TokenAlg: jwtConfig.TokenAlg, TokenExp: jwtConfig.TokenExp, TokenTyp: jwtConfig.TokenTyp})

	// 添加系统基本参数
	serverConfig := GetAllConfig().GetServerConfig(project)
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

	web.AddFilter(&node.FilterObject{Name: "RemoteCheckFilter", Order: 100, Filter: &RemoteCheckFilter{}, MatchPattern: []string{"/*"}})

	return web
}

func StartHttpNode(web *WebNode) {

	// 创建API服务
	web.POST(api("PublicKey"), web.PublicKey, &node.RouterConfig{Guest: true})
	web.POST(api("Login"), web.Login, &node.RouterConfig{UseRSA: true})
	web.POST(api("NodeLogin"), web.NodeLogin, &node.RouterConfig{UseRSA: true})
	web.POST(api("SignTradeKey"), web.SignTradeKey, &node.RouterConfig{UseRSA: true})
	web.POST(api("FindWalletList"), web.FindWalletList, &node.RouterConfig{AesRequest: true, AesResponse: true})
	web.POST(api("CreateAccount"), web.CreateAccount, &node.RouterConfig{AesRequest: true, AesResponse: true})
	web.POST(api("SignTransaction"), web.SignTransaction, &node.RouterConfig{AesRequest: true, AesResponse: true})

	web.StartServer(addr())

}
