package openwsdk

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/blocktree/go-openw-sdk/v2/openwsdk/dto"
	"github.com/blocktree/openwallet/v2/openwallet"
	"github.com/godaddy-x/freego/utils"
)

var (
	opsConfig = ReadJson("ops.json")
	cliConfig = ReadJson("cli.json")
)

var opsHttpSDK = NewHttpSDK(opsConfig)

var cliHttpSDK = NewHttpSDK(cliConfig)

// 流程：1.从CLI程序读取钱包文件列表 2.上传WalletID信息到云端
func TestCreateWallet(t *testing.T) {

	cliRequestData := dto.CliFindWalletListReq{}
	cliResponseData := dto.CliFindWalletListRes{}
	if err := cliHttpSDK.PostByAuth("/api/FindWalletList", &cliRequestData, &cliResponseData, true); err != nil {
		fmt.Println(err)
	}

	//for _, v := range cliResponseData.Result {
	//	requestData := dto.CreateWalletReq{
	//		WalletID: v.WalletID,
	//		Alias:    v.Alias,
	//		RootPath: v.RootPath,
	//	}
	//	responseData := dto.CreateWalletRes{}
	//	if err := opsHttpSDK.PostByAuth("/api/CreateWallet", &requestData, &responseData, true); err != nil {
	//		fmt.Println(err)
	//	}
	fmt.Println(cliResponseData)
	//}

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

	// TODO 1.强烈推荐业务系统，首先创建交易单保存到自身系统关键字段：Symbol，Sid，AccountID，To，ContractID 保证后续校验参数
	sid := utils.GetUUID(true)
	symbol := "BETH"
	accountID := "8K4oVwL3dLQmLzrsj2zXzbeapCsETNsRVFtykmAKBvp6"
	toAddress := "0xdAb9c307B8B23A8fD8559f75C71F0694Da30D9F6"
	toAmount := "0.1"
	contractID := "ZA+oTwXimYwVFJ5Tk7ACU6tD+6ycw7u2UsdHLVof8kg=" // 该参数不为空则认为是合约交易单

	// TODO 2.发起交易单构建请求云端系统
	requestData := dto.CreateTradeReq{
		Sid: sid,
		// 0x2346f1ca41d0161d26f46ec2885721c28fbf1375 默认地址
		AccountID: accountID,
		Coin: dto.CoinInfo{
			Symbol:     symbol,
			ContractID: contractID,
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

	// TODO 4.反序列交易单对象并进行关键字段校验Symbol，Sid，AccountID，To, ContractID
	tx := &openwallet.RawTransaction{}
	if err := utils.JsonUnmarshal(utils.Str2Bytes(txData.Data), tx); err != nil {
		fmt.Println(err)
		return
	}

	if tx.Sid != sid {
		fmt.Println(errors.New("sid invalid"))
		return
	}

	if tx.Coin.Symbol != symbol {
		fmt.Println(errors.New("symbol invalid"))
		return
	}

	if tx.Coin.ContractID != contractID {
		fmt.Println(errors.New("contractID invalid"))
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
		Type:      0,
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

func TestCreateSummaryTx(t *testing.T) {

	// TODO 1.强烈推荐业务系统，首先创建交易单保存到自身系统关键字段：Symbol，Sid，AccountID，To，ContractID 保证后续校验参数
	sid := utils.GetUUID(true)
	symbol := "BETH"
	accountID := "8K4oVwL3dLQmLzrsj2zXzbeapCsETNsRVFtykmAKBvp6"
	toAddress := "0xdAb9c307B8B23A8fD8559f75C71F0694Da30D9F6"
	contractID := "ZA+oTwXimYwVFJ5Tk7ACU6tD+6ycw7u2UsdHLVof8kg=" // 该参数不为空则认为是合约交易单

	// TODO 2.发起汇总交易单构建请求云端系统
	requestData := dto.CreateSummaryTxReq{
		Sid:             sid,
		AccountID:       accountID,
		MinTransfer:     "0",
		RetainedBalance: "0",
		Address:         toAddress,
		Coin:            dto.CoinInfo{Symbol: symbol, ContractID: contractID},
	}
	responseData := dto.CreateTradeRes{}
	if err := opsHttpSDK.PostByAuth("/api/CreateSummaryTx", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}

	// TODO 3.进行交易单JSON数据验签
	if err := CheckTxDataSign(opsConfig.AppKey, responseData.TxData); err != nil {
		fmt.Println(err)
		return
	}

	for _, v := range responseData.TxData {
		// TODO 4.反序列交易单对象并进行关键字段校验Symbol，Sid，AccountID，To, ContractID
		txErr := &openwallet.RawTransactionWithError{}
		if err := utils.JsonUnmarshal(utils.Str2Bytes(v.Data), txErr); err != nil {
			fmt.Println(err)
			return
		}

		tx := txErr.RawTx

		if !strings.HasPrefix(tx.Sid, utils.AddStr(sid, "#")) {
			fmt.Println(errors.New("sid invalid"))
			return
		}

		if tx.Coin.Symbol != symbol {
			fmt.Println(errors.New("symbol invalid"))
			return
		}

		if tx.Coin.ContractID != contractID {
			fmt.Println(errors.New("contractID invalid"))
			return
		}

		if tx.Account.AccountID != accountID {
			fmt.Println(errors.New("accountID invalid"))
			return
		}

		for _, to := range tx.TxTo {
			if !strings.HasPrefix(to, utils.AddStr(toAddress, ":")) {
				fmt.Println(errors.New("toAddress/toAmount invalid"))
				return
			}
		}

		// TODO 5.交易单转发到CLI程序签名
		cliRequestData := dto.CliSignTransactionReq{
			Type:      1,
			Data:      v.Data,
			TradeSign: v.TradeSign,
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

	}
}
