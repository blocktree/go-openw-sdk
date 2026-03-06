package openwsdk

import (
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/blocktree/go-openw-sdk/v2/openwsdk/dto"
	"github.com/godaddy-x/freego/utils"
	"github.com/godaddy-x/freego/utils/sdk"
)

func readJson(path string) SdkConfig {
	data, err := utils.ReadFile(path)
	if err != nil {
		panic(err)
	}
	config := SdkConfig{}
	if err := utils.JsonUnmarshal(data, &config); err != nil {
		panic(err)
	}
	return config
}

var (
	opsConfig = readJson("ops.json")
	cliConfig = readJson("cli.json")
)

var opsHttpSDK = NewHttpSDK(opsConfig)

var cliHttpSDK = NewHttpSDK(cliConfig)

func TestGetPublicKey(t *testing.T) {
	_, publicKey, _, err := opsHttpSDK.GetPublicKey()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("server key: ", publicKey)
}

func TestOpsLogin(t *testing.T) {
	requestData := dto.AppLoginReq{
		AppID: opsConfig.AppID,
		Nonce: utils.Base64Encode(utils.GetRandomSecure(32)),
		Time:  utils.UnixSecond(),
	}
	h, _ := hex.DecodeString(opsConfig.AppKey)
	requestData.Sign = utils.Base64Encode(utils.HMAC_SHA256_BASE(h, utils.Str2Bytes(utils.AddStr(requestData.Nonce, requestData.Time))))
	responseData := sdk.AuthToken{}
	if err := opsHttpSDK.PostByECC("/api/Login", &requestData, &responseData); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
}

func TestCliLogin(t *testing.T) {
	requestData := dto.AppLoginReq{
		AppID: cliConfig.AppID,
		Nonce: utils.Base64Encode(utils.GetRandomSecure(32)),
		Time:  utils.UnixSecond(),
	}
	h, _ := hex.DecodeString(cliConfig.AppKey)
	requestData.Sign = utils.Base64Encode(utils.HMAC_SHA256_BASE(h, utils.Str2Bytes(utils.AddStr(requestData.Nonce, requestData.Time))))
	responseData := sdk.AuthToken{}
	if err := cliHttpSDK.PostByECC("/api/Login", &requestData, &responseData); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
}

func TestSymbolBlockList(t *testing.T) {
	for {
		requestData := dto.SymbolBlockListReq{}
		responseData := dto.SymbolBlockListRes{}
		if err := opsHttpSDK.PostByAuth("/api/SymbolBlockList", &requestData, &responseData, true); err != nil {
			fmt.Println(err)
		}
		fmt.Println(responseData)
		time.Sleep(500 * time.Millisecond)
	}
}

func TestGetBlockStatus(t *testing.T) {
	requestData := dto.GetBlockStatusReq{
		Symbol: "BETH",
	}
	responseData := dto.GetBlockStatusRes{}
	if err := opsHttpSDK.PostByAuth("/api/GetBlockStatus", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
}

func TestFindWalletByParams(t *testing.T) {
	requestData := dto.FindWalletByParamsReq{}
	responseData := dto.FindWalletByParamsRes{}
	if err := opsHttpSDK.PostByAuth("/api/FindWalletByParams", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
}

func TestFindWalletByWalletID(t *testing.T) {
	requestData := dto.FindWalletByWalletIDReq{
		WalletID: "VyueEJphPSTCQBxdJkS68m5Pmt1BGwVzQD",
	}
	responseData := dto.FindWalletByWalletIDRes{}
	if err := opsHttpSDK.PostByAuth("/api/FindWalletByWalletID", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
}

func TestGetContracts(t *testing.T) {
	requestData := dto.GetContractsReq{}
	responseData := dto.GetContractsRes{}
	if err := opsHttpSDK.PostByAuth("/api/GetContracts", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
}

func TestFindTradeLog(t *testing.T) {
	requestData := dto.FindTradeLogReq{}
	responseData := dto.FindTradeLogRes{}
	if err := opsHttpSDK.PostByAuth("/api/FindTradeLog", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
}

func TestFindAccountByAccountID(t *testing.T) {
	requestData := dto.FindAccountByAccountIDReq{
		AccountID: "2TBCLPTaRQpbG6VwWuTNdPPgQtC8o3tzVnqUgJvTXmGs",
	}
	responseData := dto.FindAccountByAccountIDRes{}
	if err := opsHttpSDK.PostByAuth("/api/FindAccountByAccountID", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
}

func TestFindAccountByWalletID(t *testing.T) {
	requestData := dto.FindAccountByWalletIDReq{
		WalletID: "VzYK21Vem6WBXHXZmSRYGN4iaE6n2naF6z",
	}
	responseData := dto.FindAccountByWalletIDRes{}
	if err := opsHttpSDK.PostByAuth("/api/FindAccountByWalletID", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
}

func TestCreateAddress(t *testing.T) {
	requestData := dto.CreateAddressReq{
		WalletID:  "W1iZEvXUWYNgwJSgbxCmoaJkZJrsphRApb",
		AccountID: "2TBCLPTaRQpbG6VwWuTNdPPgQtC8o3tzVnqUgJvTXmGs",
		Symbol:    "BETH",
		Count:     2,
	}
	responseData := dto.CreateAccountRes{}
	if err := opsHttpSDK.PostByAuth("/api/CreateAddress", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
}

func TestFindAddressByAddress(t *testing.T) {
	requestData := dto.FindAddressByAddressReq{
		Address: "0xa6f4ddc5f8b6b6a07e1e250531f7600daa227138",
	}
	responseData := dto.FindAddressByAddressRes{}
	if err := opsHttpSDK.PostByAuth("/api/FindAddressByAddress", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
}

func TestFindAddressByAccountID(t *testing.T) {
	requestData := dto.FindAddressByAccountIDReq{
		AccountID: "8K4oVwL3dLQmLzrsj2zXzbeapCsETNsRVFtykmAKBvp6",
	}
	requestData.CountQ = true // 首次查询可以填充该参数获得总条数
	responseData := dto.FindAddressByAccountIDRes{}
	if err := opsHttpSDK.PostByAuth("/api/FindAddressByAccountID", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
}

func TestCreateSubscribe(t *testing.T) {
	requestData := dto.CreateSubscribeReq{
		SubscribeMethod:   []string{"Transfer", "Balance"},
		SubscribeContract: []string{},
	}
	responseData := dto.CreateSubscribeRes{}
	if err := opsHttpSDK.PostByAuth("/api/CreateSubscribe", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
}

func TestFindTradePushList(t *testing.T) {
	requestData := dto.FindTradePushListReq{}
	requestData.Limit = 1
	responseData := dto.FindTradePushListRes{}
	if err := opsHttpSDK.PostByAuth("/api/FindTradePushList", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
	for _, v := range responseData.Result {
		sign, err := SignTradePush(opsConfig.AppKey, v)
		if err != nil {
			panic(err)
		}
		if sign != v.DataSign {
			panic("trade push data sign invalid")
		}
	}
}

func TestFindTradeBalancePushList(t *testing.T) {
	requestData := dto.FindTradeBalancePushListReq{}
	requestData.Limit = 1
	responseData := dto.FindTradeBalancePushListRes{}
	if err := opsHttpSDK.PostByAuth("/api/FindTradeBalancePushList", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
	for _, v := range responseData.Result {
		sign, err := SignTradeBalancePush(opsConfig.AppKey, v)
		if err != nil {
			panic(err)
		}
		if sign != v.DataSign {
			panic("trade balance push data sign invalid")
		}
	}
}

func TestConfirmPushData(t *testing.T) {
	requestData := dto.ConfirmPushDataReq{
		DataType: "Balance",
		DataList: []int64{2017887484863578112},
	}
	responseData := dto.ConfirmPushDataRes{}
	if err := opsHttpSDK.PostByAuth("/api/ConfirmPushData", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
}
