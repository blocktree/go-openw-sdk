package openwsdk

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/blocktree/go-openw-sdk/v2/openwsdk/dto"
	"github.com/blocktree/openwallet/v2/openwallet"
	"github.com/godaddy-x/freego/utils"
	"github.com/godaddy-x/freego/utils/sdk"
	"testing"
)

func printJSON(data interface{}) {
	s, _ := utils.JsonMarshal(data)
	fmt.Println(string(s))
}

const (
	domain   = "http://localhost:8422"
	appID    = "e6a06259193fb8476ffd83a87c4fc300"
	appKey   = "43cb8a4f8c795c74426aed363aa9c12af0d065ca33b472c6ec7ce5cf7bc47c7c"
	tradeKey = "381f6b35d9acad1e744a0b75e0f64f3ccfc3802a0197a7a39d77dacd58ed7d6a"
)

var httpSDK = NewHttpSDK(domain, appID, appKey)

func NewHttpSDK(domain, appID, appKey string) *sdk.HttpSDK {
	newObject := &sdk.HttpSDK{
		Domain:    domain,
		KeyPath:   "/api/PublicKey",
		LoginPath: "/api/Login",
	}
	clientPrk := "uckgLxKoRjSHKjlsqa1gfYlHmza0DTRl/cRdV6DEaNY="
	serverPub := "BDTL1IlMt+k2glN0Rnwzt7hX8cxWougeorB7hBTTheAqNELXRGTln6oPzqvL0WMhHkruudnFGMAemYsEby8iu80="
	newObject.SetClientNo(1)
	_ = newObject.SetECDSAObject(newObject.ClientNo, clientPrk, serverPub)
	newObject.AuthObject(func() (interface{}, error) {
		requestData := dto.AppLoginReq{
			AppID: appID,
			Nonce: utils.Base64Encode(utils.GetRandomSecure(32)),
			Time:  utils.UnixSecond(),
		}
		h, err := hex.DecodeString(appKey)
		if err != nil {
			return nil, err
		}
		requestData.Sign = utils.Base64Encode(utils.HMAC_SHA256_BASE(h, utils.Str2Bytes(utils.AddStr(requestData.Nonce, requestData.Time))))
		return requestData, nil
	})
	return newObject
}

func CheckTxDataSign(appKey string, txData []*dto.TxData) error {
	if len(txData) == 0 {
		return errors.New("tx data is nil")
	}
	h, err := hex.DecodeString(appKey)
	if err != nil {
		return err
	}
	for _, v := range txData {
		checkSign := utils.HMAC_SHA256_BASE(utils.Str2Bytes(v.Data), h)
		if v.DataSign != utils.Base64Encode(checkSign) {
			return errors.New(fmt.Sprintf("tx data check sign invalid: %s", v.Data))
		}
	}
	return nil
}

func CheckTxTradeSign(tradeKey string, txData []*dto.TxData) error {
	if len(txData) == 0 {
		return errors.New("tx data is nil")
	}
	h, err := hex.DecodeString(tradeKey)
	if err != nil {
		return err
	}
	for _, v := range txData {
		checkSign := utils.HMAC_SHA256_BASE(utils.Str2Bytes(v.Data), h)
		if v.TradeSign != utils.Base64Encode(checkSign) {
			return errors.New(fmt.Sprintf("tx data check trade sign invalid: %s", v.Data))
		}
	}
	return nil
}

func TestGetPublicKey(t *testing.T) {
	_, publicKey, _, err := httpSDK.GetPublicKey()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("server key: ", publicKey)
}

func TestAppLogin(t *testing.T) {
	requestData := dto.AppLoginReq{
		AppID: appID,
		Nonce: utils.Base64Encode(utils.GetRandomSecure(32)),
		Time:  utils.UnixSecond(),
	}
	h, _ := hex.DecodeString(appKey)
	requestData.Sign = utils.Base64Encode(utils.HMAC_SHA256_BASE(h, utils.Str2Bytes(utils.AddStr(requestData.Nonce, requestData.Time))))
	responseData := sdk.AuthToken{}
	if err := httpSDK.PostByECC("/api/Login", &requestData, &responseData); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
}

func TestSymbolBlockList(t *testing.T) {
	requestData := dto.SymbolBlockListReq{}
	responseData := dto.SymbolBlockListRes{}
	if err := httpSDK.PostByAuth("/api/SymbolBlockList", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
}

func TestGetBlockStatus(t *testing.T) {
	requestData := dto.GetBlockStatusReq{
		Symbol: "BETH",
	}
	responseData := dto.GetBlockStatusRes{}
	if err := httpSDK.PostByAuth("/api/GetBlockStatus", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
}

func TestFindWalletByParams(t *testing.T) {
	requestData := dto.FindWalletByParamsReq{}
	responseData := dto.FindWalletByParamsRes{}
	if err := httpSDK.PostByAuth("/api/FindWalletByParams", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
}

func TestFindWalletByWalletID(t *testing.T) {
	requestData := dto.FindWalletByWalletIDReq{
		WalletID: "VyueEJphPSTCQBxdJkS68m5Pmt1BGwVzQD",
	}
	responseData := dto.FindWalletByWalletIDRes{}
	if err := httpSDK.PostByAuth("/api/FindWalletByWalletID", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
}

func TestCreateWallet(t *testing.T) {
	key, err := loadWalletFile()
	if err != nil {
		panic(err)
	}
	requestData := dto.CreateWalletReq{
		Alias:    key.Alias,
		WalletID: key.KeyID,
		RootPath: key.RootPath,
	}
	responseData := dto.CreateWalletRes{}
	if err := httpSDK.PostByAuth("/api/CreateWallet", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
}

func TestGetContracts(t *testing.T) {
	requestData := dto.GetContractsReq{}
	responseData := dto.GetContractsRes{}
	if err := httpSDK.PostByAuth("/api/GetContracts", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
}

func TestFindTradeLog(t *testing.T) {
	requestData := dto.FindTradeLogReq{}
	responseData := dto.FindTradeLogRes{}
	if err := httpSDK.PostByAuth("/api/FindTradeLog", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
}

func TestCreateAccount(t *testing.T) {

	key, err := loadWalletFile()
	if err != nil {
		panic(err)
	}

	account, err := CreateAccount(key, "BETH", "test account 123", -1, 3972005888)
	if err != nil {
		panic(err)
	}

	requestData := dto.CreateAccountReq{
		WalletID:     account.WalletID,
		AccountID:    account.AccountID,
		Alias:        account.Alias,
		Symbol:       account.Symbol,
		PublicKey:    account.PublicKey,
		HdPath:       account.HdPath,
		ReqSigs:      account.ReqSigs,
		AccountIndex: account.AccountIndex,
	}
	responseData := dto.CreateAccountRes{}
	if err := httpSDK.PostByAuth("/api/CreateAccount", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
}

func TestFindAccountByAccountID(t *testing.T) {
	requestData := dto.FindAccountByAccountIDReq{
		AccountID: "2TBCLPTaRQpbG6VwWuTNdPPgQtC8o3tzVnqUgJvTXmGs",
	}
	responseData := dto.FindAccountByAccountIDRes{}
	if err := httpSDK.PostByAuth("/api/FindAccountByAccountID", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
}

func TestFindAccountByWalletID(t *testing.T) {
	requestData := dto.FindAccountByWalletIDReq{
		WalletID: "W1iZEvXUWYNgwJSgbxCmoaJkZJrsphRApb",
	}
	responseData := dto.FindAccountByWalletIDRes{}
	if err := httpSDK.PostByAuth("/api/FindAccountByWalletID", &requestData, &responseData, true); err != nil {
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
	if err := httpSDK.PostByAuth("/api/CreateAddress", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
}

func TestFindAddressByAddress(t *testing.T) {
	requestData := dto.FindAddressByAddressReq{
		Address: "0xa6f4ddc5f8b6b6a07e1e250531f7600daa227138",
	}
	responseData := dto.FindAddressByAddressRes{}
	if err := httpSDK.PostByAuth("/api/FindAddressByAddress", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
}

func TestFindAddressByAccountID(t *testing.T) {
	requestData := dto.FindAddressByAccountIDReq{
		AccountID: "2TBCLPTaRQpbG6VwWuTNdPPgQtC8o3tzVnqUgJvTXmGs",
	}
	requestData.CountQ = true // 首次查询可以填充该参数获得总条数
	responseData := dto.FindAddressByAccountIDRes{}
	if err := httpSDK.PostByAuth("/api/FindAddressByAccountID", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
}

// 0x1f8fabe68b9393e25235622ea754e75b37ec3dc8 5000 MTK

func TestCreateTrade(t *testing.T) {
	requestData := dto.CreateTradeReq{
		Sid: utils.GetUUID(true),
		// 0xe5b80a358a7abb340e3126057ad4bf3a44b4b4dd 默认地址
		AccountID: "2TBCLPTaRQpbG6VwWuTNdPPgQtC8o3tzVnqUgJvTXmGs",
		Coin: dto.CoinInfo{
			Symbol: "BETH",
		},
		To: map[string]string{
			"0x4f8abf232ffd006a49a9426ed9a2ab377ce4bdce": "0.5",
		},
	}
	responseData := dto.CreateTradeRes{}
	if err := httpSDK.PostByAuth("/api/CreateTrade", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)

	if err := CheckTxDataSign(appKey, responseData.TxData); err != nil {
		panic(err)
	}

	if err := CheckTxTradeSign(tradeKey, responseData.TxData); err != nil {
		panic(err)
	}

	txData := responseData.TxData[0]

	tx := &openwallet.RawTransaction{}
	if err := utils.JsonUnmarshal(utils.Str2Bytes(txData.Data), tx); err != nil {
		fmt.Println(err)
	}
	key, err := loadWalletFile()
	if err != nil {
		panic(err)
	}
	txSignerList := map[string]string{}
	if err := SignRawTransactionExtract(tx, key, txSignerList); err != nil {
		panic(err)
	}
	fmt.Println("txData: ", tx)

	txData.SignerList = txSignerList

	requestSubmitData := dto.SubmitRawTransactionReq{
		TxData: txData,
	}
	responseSubmitData := dto.SubmitRawTransactionRes{}
	if err := httpSDK.PostByAuth("/api/SubmitTrade", &requestSubmitData, &responseSubmitData, true); err != nil {
		fmt.Println(err)
	}

	fmt.Println("trade result：", responseSubmitData)
	fmt.Println("txID: ", responseSubmitData.Txid, responseSubmitData.From[0], responseSubmitData.To[0])

}

func TestCreateContractTrade(t *testing.T) {
	requestData := dto.CreateTradeReq{
		Sid:       utils.GetUUID(true),
		AccountID: "2TBCLPTaRQpbG6VwWuTNdPPgQtC8o3tzVnqUgJvTXmGs",
		Coin: dto.CoinInfo{
			Symbol:     "BETH",
			IsContract: true,
			ContractID: "ZA+oTwXimYwVFJ5Tk7ACU6tD+6ycw7u2UsdHLVof8kg=",
		},
		To: map[string]string{
			"0xe5b80a358a7abb340e3126057ad4bf3a44b4b4dd": "0.2",
		},
	}
	responseData := dto.CreateTradeRes{}
	if err := httpSDK.PostByAuth("/api/CreateTrade", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)

	if err := CheckTxDataSign(appKey, responseData.TxData); err != nil {
		panic(err)
	}

	txData := responseData.TxData[0]

	tx := &openwallet.RawTransaction{}
	if err := utils.JsonUnmarshal(utils.Str2Bytes(txData.Data), tx); err != nil {
		fmt.Println(err)
	}
	key, err := loadWalletFile()
	if err != nil {
		panic(err)
	}
	txSignerList := map[string]string{}
	if err := SignRawTransactionExtract(tx, key, txSignerList); err != nil {
		panic(err)
	}
	fmt.Println("txData: ", tx)

	txData.SignerList = txSignerList

	requestSubmitData := dto.SubmitRawTransactionReq{
		TxData: txData,
	}
	responseSubmitData := dto.SubmitRawTransactionRes{}
	if err := httpSDK.PostByAuth("/api/SubmitTrade", &requestSubmitData, &responseSubmitData, true); err != nil {
		fmt.Println(err)
	}

	fmt.Println("trade result：", responseSubmitData)

}

func TestCreateSummaryTx(t *testing.T) {
	requestData := dto.CreateSummaryTxReq{
		Sid:             utils.GetUUID(true),
		AccountID:       "2TBCLPTaRQpbG6VwWuTNdPPgQtC8o3tzVnqUgJvTXmGs",
		MinTransfer:     "0",
		RetainedBalance: "0",
		Address:         "0xa6f4ddc5f8b6b6a07e1e250531f7600daa227138",
		Coin:            dto.CoinInfo{Symbol: "BETH"},
	}
	responseData := dto.CreateTradeRes{}
	if err := httpSDK.PostByAuth("/api/CreateSummaryTx", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	for _, v := range responseData.TxData {
		fmt.Println(v.Data)
	}
}

func TestCreateSubscribe(t *testing.T) {
	requestData := dto.CreateSubscribeReq{
		SubscribeMethod:   []string{"Transfer", "Balance"},
		SubscribeContract: []string{},
	}
	responseData := dto.CreateSubscribeRes{}
	if err := httpSDK.PostByAuth("/api/CreateSubscribe", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
}

func TestFindTradePushList(t *testing.T) {
	requestData := dto.FindTradePushListReq{}
	requestData.Limit = 1
	responseData := dto.FindTradePushListRes{}
	if err := httpSDK.PostByAuth("/api/FindTradePushList", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
}

func TestFindTradeBalancePushList(t *testing.T) {
	requestData := dto.FindTradeBalancePushListReq{}
	requestData.Limit = 1
	responseData := dto.FindTradeBalancePushListRes{}
	if err := httpSDK.PostByAuth("/api/FindTradeBalancePushList", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
}

func TestConfirmPushData(t *testing.T) {
	requestData := dto.ConfirmPushDataReq{
		DataType: "Balance",
		DataList: []int64{2017887484863578112},
	}
	responseData := dto.ConfirmPushDataRes{}
	if err := httpSDK.PostByAuth("/api/ConfirmPushData", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
}
