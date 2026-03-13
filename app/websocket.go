package app

import (
	"github.com/godaddy-x/freego/cache"
	"github.com/godaddy-x/freego/node"
	"github.com/godaddy-x/freego/utils"
	"github.com/godaddy-x/freego/utils/crypto"
	"github.com/godaddy-x/freego/utils/jwt"
)

var (
	server   *node.WsServer
	keyCache = cache.NewLocalCache(1, 1)
)

func NewSocket() {
	// 创建WebSocket服务器实例
	server = node.NewWsServer(node.SubjectDeviceUnique)

	// 添加JWT配置参数
	jwtConfig := GetAllConfig().GetJwtConfig(project)
	_ = server.AddJwtConfig(jwt.JwtConfig{TokenKey: jwtConfig.TokenKey, TokenAlg: jwtConfig.TokenAlg, TokenExp: jwtConfig.TokenExp, TokenTyp: jwtConfig.TokenTyp})

	// 添加系统基本参数
	serverConfig := GetAllConfig().GetServerConfig(project)

	// 添加ECDSA配置参数,服务端私钥和客户端公钥
	for _, v := range serverConfig.Keys {
		cipher, err := crypto.CreateS256ECDSAWithBase64(v.PrivateKey, v.PublicKey)
		if err != nil {
			panic("create ecdsa object error: " + err.Error())
		}
		_ = server.AddCipher(v.Name, cipher)
	}

	// 添加Local缓存参数,默认空即可
	server.AddLocalCache(nil)

	// 配置连接池
	if err := server.NewPool(100, 10, 5, 30); err != nil {
		panic(err)
	}

	if err := server.AddRouter("/ws/mpcTempPublicKey", handleTempPublicKey, &node.RouterConfig{}); err != nil {
		panic(err)
	}

	if err := server.AddRouter("/ws/mpcKeygenResult", handleMpcKeygenResult, &node.RouterConfig{}); err != nil {
		panic(err)
	}
	if err := server.AddRouter("/ws/mpcKeygenMsg", handleMpcKeygenMsg, &node.RouterConfig{}); err != nil {
		panic(err)
	}

	// 分布式签名相关路由
	if err := server.AddRouter("/ws/mpcSignResult", handleMpcSignResult, &node.RouterConfig{}); err != nil {
		panic(err)
	}
	if err := server.AddRouter("/ws/mpcSignMsg", handleMpcSignMsg, &node.RouterConfig{}); err != nil {
		panic(err)
	}

	if err := server.StartWebsocket(utils.AddStr(serverConfig.Addr, ":", serverConfig.Port+100)); err != nil {
		panic(err)
	}

}
