package performance_test

import (
	"fmt"
	"github.com/blocktree/go-openw-sdk/v2/openwsdk"
	"github.com/blocktree/openwallet/v2/log"
	"github.com/blocktree/openwallet/v2/owtp"
	"strings"
	"testing"
)

const (
	sslkey = "FXJXCtxAfHWhAvnpsnciEfVCkThn7NGMA1kBofYRECRe"
	host   = "127.0.0.1:8422"
	appid  = "e10adc3949ba59abbe56e057f20f883e"
	appkey = "8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92"
)

var api *openwsdk.APINode

func init() {
	api = testNewAPINode()
}

func testNewAPINode() *openwsdk.APINode {
	cert, _ := owtp.NewCertificate(sslkey)
	config := &openwsdk.APINodeConfig{
		AppID:              appid,
		AppKey:             appkey,
		Host:               host,
		Cert:               cert,
		ConnectType:        owtp.HTTP,
		EnableSignature:    false,
		EnableKeyAgreement: false,
		TimeoutSEC:         120,
	}
	api := openwsdk.NewAPINode(config)
	api.BindAppDevice()
	return api
}

// 获取节点配置信息
func TestAPINode_GetNotifierNodeInfo(t *testing.T) {
	pubKey, nodeId, err := api.GetNotifierNodeInfo()
	if err != nil {
		t.Logf("GetNotifierNodeInfo unexpected error: %v", err)
		return
	}

	log.Infof("pubKey: %s", pubKey)
	log.Infof("nodeID: %s", nodeId)
}

// 获取币种列表,包含推荐费率和区块高度
func TestAPINode_GetSymbolList(t *testing.T) {
	api := testNewAPINode()
	api.GetSymbolList("QTUM", 0, 1000, 0, true, func(status uint64, msg string, total int, symbols []*openwsdk.Symbol) {
		symbolStrArray := make([]string, 0)
		for _, s := range symbols {
			fmt.Printf("symbol: %+v\n", s)
			symbolStrArray = append(symbolStrArray, s.Symbol)
		}
		allSymbols := strings.Join(symbolStrArray, ", ")
		log.Infof("all symbols: %s", allSymbols)
	})
}
