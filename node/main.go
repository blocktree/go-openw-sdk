package main

import (
	"time"

	"github.com/blocktree/go-openw-sdk/v2/openwsdk"
	"github.com/godaddy-x/freego/utils/sdk"
	"github.com/godaddy-x/freego/zlog"
)

func main() {

	cliConfig := openwsdk.ReadJson("node/cli_node.json")

	cliHttp := openwsdk.NewHttpSDK(cliConfig)

	time.Sleep(2 * time.Second)

	auth := cliHttp.GetAuth()

	wsClient := sdk.NewSocketSDK("localhost:9422")

	wsClient.AuthToken(auth)

	wsClient.SetClientNo(cliConfig.ClientNo)
	_ = wsClient.SetECDSAObject(cliConfig.ClientNo, cliConfig.ClientPrk, cliConfig.ServerPub)
	wsClient.SetHealthPing(10)

	if err := wsClient.ConnectWebSocket("/ws"); err != nil {
		zlog.Error("sdk connect websocket error", 0, zlog.String("errMsg", err.Error()))
	}

}
