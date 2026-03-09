package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"time"

	"github.com/blocktree/go-openw-sdk/v2/openwsdk"
	"github.com/blocktree/go-openw-sdk/v2/openwsdk/dto"
	ecc "github.com/godaddy-x/eccrypto"
	"github.com/godaddy-x/freego/utils"
	"github.com/godaddy-x/freego/utils/sdk"
)

func handleShardingPre(wsClient *sdk.SocketSDK, subject, router string, data []byte) error {
	request := dto.CliShardingTaskReq{}
	if err := json.Unmarshal(data, &request); err != nil {
		return errors.New("handleShardingPre json unmarshal error: " + err.Error())
	}
	prk, err := ecc.CreateECDH()
	if err != nil {
		return errors.New("create ecdh error: " + err.Error())
	}
	request.PublicKey = utils.Base64Encode(ecc.GetECDHPublicKeyBytes(prk.PublicKey()))
	response := dto.CliShardingTaskRes{}
	if err := wsClient.SendWebSocketMessage("/ws/shardingPre", &request, &response, true, true, 30); err != nil {
		return errors.New("handleShardingPre send shard message error: " + err.Error())
	}
	return nil
}

func handleShardingPost(wsClient *sdk.SocketSDK, subject, router string, data []byte) error {
	request := dto.CliShardingTaskReq{}
	if err := json.Unmarshal(data, &request); err != nil {
		return errors.New("handleShardingPost json unmarshal error: " + err.Error())
	}
	response := dto.CliShardingTaskRes{}
	if err := wsClient.SendWebSocketMessage("/ws/shardingPost", &request, &response, true, true, 30); err != nil {
		return errors.New("handleShardingPost send shard message error: " + err.Error())
	}
	//dst := memguard.NewBuffer(64)
	//defer dst.Destroy()
	//_, err = ecc.Decrypt(prk, utils.Base64Decode(response.ShardKey), utils.Str2Bytes(subject), dst.Bytes())
	//if err != nil {
	//	return errors.New("handleShardingPost decrypt error: " + err.Error())
	//}
	//fmt.Println("sharding: ", utils.Base64Encode(dst.Bytes()))
	a, _ := utils.JsonMarshal(&response)
	fmt.Println(string(a))
	return nil
}

func main() {

	// 命令行参数处理
	configFile := flag.String("config", "cli_node.json", "configuration file path")
	flag.Parse()

	cliConfig := openwsdk.ReadJson(*configFile)

	cliHttp := openwsdk.NewHttpSDK(cliConfig)

	time.Sleep(2 * time.Second)

	wsClient := sdk.NewSocketSDK(cliConfig.WSDomain)
	wsClient.SetClientNo(cliConfig.ClientNo)
	_ = wsClient.SetECDSAObject(cliConfig.ClientNo, cliConfig.ClientPrk, cliConfig.ServerPub)
	wsClient.EnableReconnect()
	auth := cliHttp.GetAuth()
	wsClient.AuthToken(auth)
	wsClient.SetTokenExpiredCallback(func() {
		auth = cliHttp.GetAuth()
		wsClient.AuthToken(auth)
	})
	wsClient.SetHealthPing(10)

	if err := wsClient.ConnectWebSocket("/ws"); err != nil {
		fmt.Println(errors.New("sdk connect websocket error: " + err.Error()))
		return
	}

	fmt.Println("sdk connect websocket success: ", cliConfig.Source)

	wsClient.SetPushMessageCallback(func(router string, data []byte) {
		if router == "shardingPre" {
			if err := handleShardingPre(wsClient, cliConfig.Source, router, data); err != nil {
				fmt.Println(err)
			}
		} else if router == "shardingPost" {
			if err := handleShardingPost(wsClient, cliConfig.Source, router, data); err != nil {
				fmt.Println(err)
			}
		}
	})

	time.Sleep(2000 * time.Second)

}
