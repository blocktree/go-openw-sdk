package main

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/awnumar/memguard"
	"github.com/blocktree/go-openw-sdk/v2/openwsdk/dto"
	"github.com/codahale/sss"
	"github.com/godaddy-x/freego/utils"
	"github.com/godaddy-x/freego/utils/sdk"
	"github.com/nbutton23/zxcvbn-go"
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

func TestSharding(t *testing.T) {
	secret := memguard.NewBufferRandom(64) // our secret
	n := byte(3)                           // create 30 shares
	k := byte(2)                           // require 2 of them to combine

	shares, err := sss.Split(n, k, secret.Data()) // split into 30 shares
	if err != nil {
		fmt.Println(err)
		return
	}

	// select a random subset of the total shares
	subset := make(map[byte][]byte, k)
	for x, y := range shares { // just iterate since maps are randomized
		subset[x] = y
		if len(subset) == int(k) {
			break
		}
	}

	// combine two shares and recover the secret
	recovered := string(sss.Combine(subset))
	fmt.Println(recovered)
}

func TestPasswordCheck(t *testing.T) {
	password := "abc123456789#@!"
	// 可选：传入用户名、邮箱等，防止用个人信息当密码
	result := zxcvbn.PasswordStrength(password, nil)

	fmt.Printf("Password: %s\n", password)
	fmt.Printf("Estimated entropy: %.1f bits\n", result.Entropy)
	fmt.Printf("Crack time (online): %s\n", result.CrackTimeDisplay)
	fmt.Printf("Score (0-4): %d\n", result.Score) // 0=weak, 4=strong
	fmt.Printf("Suggestions: %v\n", result.Score)
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
	requestData := dto.CliFindWalletListReq{}
	responseData := dto.CliFindWalletListRes{}
	if err := httpSDK.PostByAuth("/api/FindWalletList", &requestData, &responseData, true); err != nil {
		fmt.Println(err)
	}
	fmt.Println(responseData)
}

func TestCreateAccount(t *testing.T) {
	requestData := dto.CliCreateAccountReq{
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
