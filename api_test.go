package main

import (
	"encoding/hex"
	"fmt"
	"github.com/blocktree/go-openw-sdk/v2/web/dto"
	"github.com/godaddy-x/freego/utils"
	"github.com/godaddy-x/freego/utils/sdk"
	"testing"
)

const (
	domain = "http://localhost:9422"
	appID  = "e6a06259193fb8476ffd83a87c4fc300"
	appKey = "43cb8a4f8c795c74426aed363aa9c12af0d065ca33b472c6ec7ce5cf7bc47c7c"
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

func TestGetPublicKey(t *testing.T) {
	_, publicKey, _, err := httpSDK.GetPublicKey()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("server key: ", publicKey)
}

func TestUserLogin(t *testing.T) {
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

func TestFindWalletList(t *testing.T) {
	requestData := dto.FindWalletListReq{}
	responseData := dto.FindWalletListRes{}
	if err := httpSDK.PostByAuth("/api/FindWalletList", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
}

func TestCreateAccount(t *testing.T) {
	requestData := dto.CreateAccountReq{
		WalletID:  "VzYK21Vem6WBXHXZmSRYGN4iaE6n2naF6z",
		LastIndex: -1,
		Curve:     3972005888,
	}
	responseData := dto.CreateAccountRes{}
	if err := httpSDK.PostByAuth("/api/CreateAccount", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
}
