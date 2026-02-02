package openwsdk

import (
	"github.com/blocktree/go-openw-sdk/v2/openwsdk/dto"
	"github.com/godaddy-x/freego/utils"
	"strings"
)

func SignTradePush(key []byte, data dto.TradePushResult) (string, error) {
	// 构建推送数据的签名，包含关键字段
	// 确保推送数据的完整性和防篡改性
	var signData strings.Builder
	signData.WriteString(utils.AnyToStr(data.ID))
	signData.WriteString("|")
	signData.WriteString(data.AppID)
	signData.WriteString("|")
	signData.WriteString(data.Data)
	signData.WriteString("|")
	signData.WriteString(utils.AnyToStr(data.Ctime))
	return utils.Base64Encode(utils.HMAC_SHA256_BASE(utils.Str2Bytes(signData.String()), key)), nil
}

func SignTradeBalancePush(key []byte, data dto.TradeBalancePushResult) (string, error) {
	// 构建推送数据的签名，包含关键字段
	// 确保推送数据的完整性和防篡改性
	var signData strings.Builder
	signData.WriteString(utils.AnyToStr(data.ID))
	signData.WriteString("|")
	signData.WriteString(data.AppID)
	signData.WriteString("|")
	signData.WriteString(data.Data)
	signData.WriteString("|")
	signData.WriteString(utils.AnyToStr(data.Ctime))
	return utils.Base64Encode(utils.HMAC_SHA256_BASE(utils.Str2Bytes(signData.String()), key)), nil
}
