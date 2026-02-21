package openwsdk

import (
	"errors"
	"fmt"
	"github.com/blocktree/go-openw-sdk/v2/openwsdk/dto"
	"github.com/blocktree/openwallet/v2/openwallet"
	"github.com/godaddy-x/freego/utils"
	"testing"
)

// 流程：1.从CLI程序读取钱包文件列表 2.上传WalletID信息到云端
func TestCreateWallet(t *testing.T) {

	cliRequestData := dto.CliFindWalletListReq{}
	cliResponseData := dto.CliFindWalletListRes{}
	if err := cliHttpSDK.PostByAuth("/api/FindWalletList", &cliRequestData, &cliResponseData, true); err != nil {
		fmt.Println(err)
	}

	for _, v := range cliResponseData.Result {
		requestData := dto.CreateWalletReq{
			WalletID: v.WalletID,
			Alias:    v.Alias,
			RootPath: v.RootPath,
		}
		responseData := dto.CreateWalletRes{}
		if err := opsHttpSDK.PostByAuth("/api/CreateWallet", &requestData, &responseData, true); err != nil {
			fmt.Println(err)
		}
		fmt.Println(responseData)
	}

}

// 流程：1.指定WalletID发送给CLI程序进行派生AccountID 2.上传AccountID信息到云端
func TestCreateAccount(t *testing.T) {

	cliRequestData := dto.CliCreateAccountReq{
		WalletID:  "VzYK21Vem6WBXHXZmSRYGN4iaE6n2naF6z",
		LastIndex: -1,
		Curve:     3972005888,
	}
	cliResponseData := dto.CliCreateAccountRes{}
	if err := cliHttpSDK.PostByAuth("/api/CreateAccount", &cliRequestData, &cliResponseData, true); err != nil {
		fmt.Println(err)
	}

	requestData := dto.CreateAccountReq{
		WalletID:     cliResponseData.WalletID,
		AccountID:    cliResponseData.AccountID,
		Alias:        "test",
		Symbol:       "BETH",
		PublicKey:    cliResponseData.PublicKey,
		HdPath:       cliResponseData.HdPath,
		ReqSigs:      cliResponseData.ReqSigs,
		AccountIndex: cliResponseData.AccountIndex,
	}
	responseData := dto.CreateAccountRes{}
	if err := opsHttpSDK.PostByAuth("/api/CreateAccount", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
}

// 流程：1.指定帐户ID通过云端创建交易单 2.业务系统验证数据签名和校验业务数据 3.发送交易单给CLI进行验签并签名
func TestCreateTrade(t *testing.T) {

	// TODO 1.强烈推荐业务系统，首先创建交易单保存到自身系统关键字段：Symbol，Sid，AccountID，To，保证后续校验参数
	symbol := "BETH"
	accountID := "8K4oVwL3dLQmLzrsj2zXzbeapCsETNsRVFtykmAKBvp6"
	toAddress := "0x4f8abf232ffd006a49a9426ed9a2ab377ce4bdce"
	toAmount := "0.1"

	// TODO 2.发起交易单构建请求云端系统
	requestData := dto.CreateTradeReq{
		Sid: utils.GetUUID(true),
		// 0x2346f1ca41d0161d26f46ec2885721c28fbf1375 默认地址
		AccountID: accountID,
		Coin: dto.CoinInfo{
			Symbol: symbol,
		},
		To: map[string]string{
			toAddress: toAmount,
		},
	}
	responseData := dto.CreateTradeRes{}
	if err := opsHttpSDK.PostByAuth("/api/CreateTrade", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(responseData)

	// TODO 3.进行交易单JSON数据验签
	if err := CheckTxDataSign(opsConfig.AppKey, responseData.TxData); err != nil {
		fmt.Println(err)
		return
	}

	txData := responseData.TxData[0]

	// TODO 4.反序列交易单对象并进行关键字段校验Symbol，Sid，AccountID，To
	tx := &openwallet.RawTransaction{}
	if err := utils.JsonUnmarshal(utils.Str2Bytes(txData.Data), tx); err != nil {
		fmt.Println(err)
		return
	}
	if tx.Coin.Symbol != symbol {
		fmt.Println(errors.New("symbol invalid"))
		return
	}
	if tx.Account.AccountID != accountID {
		fmt.Println(errors.New("accountID invalid"))
		return
	}
	for _, v := range tx.TxTo {
		if v != utils.AddStr(toAddress, ":", toAmount) {
			fmt.Println(errors.New("toAddress/toAmount invalid"))
			return
		}
	}

	// TODO 5.交易单转发到CLI程序签名
	cliRequestData := dto.CliSignTransactionReq{
		Data:      txData.Data,
		TradeSign: txData.TradeSign,
	}
	cliResponseData := dto.CliSignTransactionRes{}
	if err := cliHttpSDK.PostByAuth("/api/SignTransaction", &cliRequestData, &cliResponseData, true); err != nil {
		fmt.Println(err)
		return
	}
	if len(cliResponseData.SignerList) == 0 {
		fmt.Println(errors.New("cli signer is nil"))
		return
	}

	// TODO 6.交易单CLI签名成功后，进行云端系统广播
	txData.SignerList = cliResponseData.SignerList
	requestSubmitData := dto.SubmitRawTransactionReq{
		TxData: txData,
	}
	responseSubmitData := dto.SubmitRawTransactionRes{}
	if err := opsHttpSDK.PostByAuth("/api/SubmitTrade", &requestSubmitData, &responseSubmitData, true); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("trade result：", responseSubmitData)
	fmt.Println("txID: ", responseSubmitData.Txid, responseSubmitData.From[0], responseSubmitData.To[0])

}

func TestCreateContractTrade(t *testing.T) {
	// 0x1f8fabe68b9393e25235622ea754e75b37ec3dc8 5000 MTK
	requestData := dto.CreateTradeReq{
		Sid:       utils.GetUUID(true),
		AccountID: "2TBCLPTaRQpbG6VwWuTNdPPgQtC8o3tzVnqUgJvTXmGs",
		Coin: dto.CoinInfo{
			Symbol:     "BETH",
			IsContract: true,
			ContractID: "ZA+oTwXimYwVFJ5Tk7ACU6tD+6ycw7u2UsdHLVof8kg=",
		},
		To: map[string]string{
			"0xdAb9c307B8B23A8fD8559f75C71F0694Da30D9F6": "0.2",
		},
	}
	responseData := dto.CreateTradeRes{}
	if err := opsHttpSDK.PostByAuth("/api/CreateTrade", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)

	if err := CheckTxDataSign(opsConfig.AppKey, responseData.TxData); err != nil {
		panic(err)
	}

	txData := responseData.TxData[0]

	tx := &openwallet.RawTransaction{}
	if err := utils.JsonUnmarshal(utils.Str2Bytes(txData.Data), tx); err != nil {
		fmt.Println(err)
	}
	//key, err := loadWalletFile()
	//if err != nil {
	//	panic(err)
	//}
	//txSignerList := map[string]string{}
	//if err := SignRawTransactionExtract(tx, key, txSignerList); err != nil {
	//	panic(err)
	//}
	//fmt.Println("txData: ", tx)
	//
	//txData.SignerList = txSignerList

	requestSubmitData := dto.SubmitRawTransactionReq{
		TxData: txData,
	}
	responseSubmitData := dto.SubmitRawTransactionRes{}
	if err := opsHttpSDK.PostByAuth("/api/SubmitTrade", &requestSubmitData, &responseSubmitData, true); err != nil {
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
	if err := opsHttpSDK.PostByAuth("/api/CreateSummaryTx", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	for _, v := range responseData.TxData {
		fmt.Println(v.Data)
	}
}
