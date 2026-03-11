package main

import (
	"crypto/ecdh"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"time"

	"github.com/blocktree/go-openw-sdk/v2/openwsdk"
	"github.com/blocktree/go-openw-sdk/v2/openwsdk/dto"
	ecc "github.com/godaddy-x/eccrypto"
	"github.com/godaddy-x/freego/cache"
	"github.com/godaddy-x/freego/utils"
	"github.com/godaddy-x/freego/utils/sdk"
)

var (
	keygenCache = cache.NewLocalCache(1, 1)
)

func getTempPrivateKey(mod, subject, taskID string) (*ecdh.PrivateKey, error) {
	key := utils.FNV1a64(utils.AddStr(subject, ":", taskID, ":", mod, ":tempPrivateKey"))
	value, b, err := keygenCache.Get(key, nil)
	if err != nil {
		return nil, err
	}
	if b && value != nil {
		return value.(*ecdh.PrivateKey), nil
	}
	return nil, nil
}

func handleTempPublicKey(wsClient *sdk.SocketSDK, subject, router string, data []byte) error {
	request := dto.CliMPCTempPublicKeyReq{}
	if err := json.Unmarshal(data, &request); err != nil {
		return errors.New("handleTempPublicKey json unmarshal error: " + err.Error())
	}
	if request.Module == "" {
		return errors.New("handleTempPublicKey invalid module")
	}
	prk, err := ecc.CreateECDH()
	if err != nil {
		return errors.New("handleTempPublicKey create ecdh error: " + err.Error())
	}
	request.PublicKey = utils.Base64Encode(ecc.GetECDHPublicKeyBytes(prk.PublicKey()))
	response := dto.CliMPCTempPublicKeyRes{}
	if err := wsClient.SendWebSocketMessage("/ws/mpcTempPublicKey", &request, &response, true, true, 30); err != nil {
		return errors.New("handleTempPublicKey send shard message error: " + err.Error())
	}
	if response.Success {
		cacheKey := utils.FNV1a64(utils.AddStr(subject, ":", request.TaskID, ":", request.Module, ":tempPrivateKey"))
		if err := keygenCache.Put(cacheKey, prk, 600); err != nil {
			return errors.New("handleTempPublicKey put tempPrivateKey error: " + err.Error())
		}
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

func RunMPCNode(cliConfig openwsdk.SdkConfig) {
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
		if router == "mpcTempPublicKey" {
			if err := handleTempPublicKey(wsClient, cliConfig.Source, router, data); err != nil {
				fmt.Println(err)
			}
		} else if router == "shardingPost" {
			if err := handleShardingPost(wsClient, cliConfig.Source, router, data); err != nil {
				fmt.Println(err)
			}
		} else if router == "mpcKeygenStart" {
			go func() {
				if err := HandleMpcKeygenStart(wsClient, cliConfig.Source, router, data); err != nil {
					fmt.Println("mpc keygen error:", err)
				} else {
					fmt.Println("mpc keygen done, result submitted")
				}
			}()
		} else if router == "mpcKeygenMsg" {
			fmt.Printf("[mpc-keygen] Push received: router=%s len=%d\n", router, len(data))
			if err := DeliverMpcKeygenMsg(wsClient, cliConfig.Source, router, data); err != nil && err.Error() != "Error is nil" {
				fmt.Println("mpcKeygenMsg deliver error:", err)
			}
		}
	})
}

func main() {

	// 命令行参数处理
	configFile := flag.String("config", "cli_node.json", "configuration file path")
	flag.Parse()

	cliConfig := openwsdk.ReadJson(*configFile)

	RunMPCNode(cliConfig)

	time.Sleep(2000 * time.Second)

}
