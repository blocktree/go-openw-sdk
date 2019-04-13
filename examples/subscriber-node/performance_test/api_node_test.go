package performance_test

import (
	"github.com/blocktree/openwallet/owtp"
	"testing"
	"fmt"
	"runtime"
	"sync"
	"time"
	"reflect"
	"sort"
	"go-openw-sdk/openwsdk"
)

func testNewAPINode() *openwsdk.APINode {

	//--------------- PRIVATE KEY ---------------
	//CaeQzossEasxDmDx4sS12eQC2L7zzNGVwEW2T1CKK3ZS
	//--------------- PUBLIC KEY ---------------
	//3Gve895o6aarxYzgLu8tKy3EXVFmFw6oFh1dbpVXmy8VtRaxa6tzpKRPc568549Q5jLpNJGbkXY5HqoQH5gvbg6o
	//--------------- NODE ID ---------------
	//4YBHa3d3vAceSRngPWrsm1cSPJudFQSzNAhPGschFw47

	cert, _ := owtp.NewCertificate("CaeQzossEasxDmDx4sS12eQC2L7zzNGVwEW2T1CKK3ZS", "")

	config := &openwsdk.APINodeConfig{
		AppID:  "8df7420d3917afa0172ea9c85e07ab55",
		AppKey: "faa14b5e2cf119cd6d38bda45b49eb02b333a1b1ff6f10703acb554011ebfb1e",
		Host:   "120.78.83.180",
		//AppID:  "b4b1962d415d4d30ec71b28769fda585",
		//AppKey: "8c511cb683041f3589419440fab0a7b7710907022b0d035baea9001d529ca72f",
		//Host: "192.168.27.181:8422",
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

//func testGetLocalKey() (*hdkeystore.HDKey, error) {
//	keypath := filepath.Join("testkeys")
//	keystore := hdkeystore.NewHDKeystore(
//		keypath,
//		hdkeystore.StandardScryptN,
//		hdkeystore.StandardScryptP,
//	)
//
//	key, err := keystore.GetKey(
//		"WAaDbbawmypQY3XjnMjLTj43vBGvrQwB2j",
//		"TRON-WAaDbbawmypQY3XjnMjLTj43vBGvrQwB2j.key",
//		"1234qwer",
//	)
//
//	if err != nil {
//		return nil, err
//	}
//
//	return key, nil
//}

func TestAPINode(t *testing.T) {
	api := testNewAPINode()
	//key, err := testGetLocalKey()
	//fmt.Println(key)
	//if err != nil {
	//	t.Logf("GetKey error: %v\n", err)
	//	return
	//}

	// 线程数
	numThreads := 20
	// 每个线程请求次数（总请求数=numThreads*numCalls）
	numCalls := 20

	runtime.GOMAXPROCS(runtime.NumCPU())

	var waitGroup sync.WaitGroup
	waitGroup.Add(numThreads)
	responseTimesChan := make(chan float64)

	//错误数
	var errorCount int

	startTime := time.Now()

	for i := 1; i <= numThreads; i++ {
		go func() {
			defer waitGroup.Done()

			for j := 1; j <= numCalls; j++ {
				startTime := time.Now()

				/* 不同的api调用请修改这里*/

				err := api.FindAccountByWalletID("WN84dVZXpgVixsvXnU8jkFWD1qWHp15LpA", true,
					func(status uint64, msg string, accounts []*openwsdk.Account) {

						if status != owtp.StatusSuccess {
							t.Logf("unexpected error: %v\n", msg)
							return
						}
					})

				//err := api.GetFeeRate("BTC", true,
				//	func(status uint64, msg, symbol, feeRate, unit string) {
				//		//log.Infof("balance: %s %s/%s",feeRate, symbol, unit)
				//	})

				if err != nil {
					fmt.Println(reflect.ValueOf(err).Elem())
					errorCount ++
				}

				// 响应时间(ms)
				responseTime := time.Now().Sub(startTime).Seconds() * 1000
				responseTimesChan <- responseTime
			}

		}()
	}

	// 监听线程是否全部结束
	go func() {
		waitGroup.Wait()
		close(responseTimesChan)
	}()

	var totalTime float64
	responseTimes := make([]float64, 0, numThreads*numCalls)
	for responseTime := range responseTimesChan {
		responseTimes = append(responseTimes, responseTime)
		totalTime += responseTime
	}

	subTime := time.Now().Sub(startTime)
	// 吞吐量 (请求/秒)
	throughput := float64(numThreads*numCalls) / subTime.Seconds()
	// 平均响应时间(ms)
	responseTime := totalTime / float64(numThreads*numCalls)
	// 响应时间min/max/90%/95% (ms)
	//fmt.Printf("%.0f\n\n", responseTimes)
	sort.Slice(responseTimes, func(i, j int) bool { return responseTimes[i] < responseTimes[j] })
	//fmt.Printf("%.0f\n\n", responseTimes)
	responseTimesMin := responseTimes[0]
	responseTimesMax := responseTimes[len(responseTimes)-1]
	indexRt90 := int(float64(numThreads*numCalls) * 0.9)
	indexRt95 := int(float64(numThreads*numCalls) * 0.95)

	fmt.Println("==================== 运行结果 ===========================")
	fmt.Printf(">>> 并发数：%d \n", numThreads)
	fmt.Printf(">>> 请求数：%d \n", numThreads*numCalls)
	fmt.Printf(">>> 总耗时：%.2f s\n", subTime.Seconds())
	fmt.Printf(">>> 吞吐量： %.2f tps\n", throughput)
	fmt.Printf(">>> 平均响应时间：%.0f ms\n", responseTime)
	fmt.Printf(">>> 响应时间(90%% line)：%.0f ms\n", responseTimes[indexRt90])
	fmt.Printf(">>> 响应时间(95%% line)：%.0f ms\n", responseTimes[indexRt95])
	fmt.Printf(">>> 响应时间min：%.0f ms\n", responseTimesMin)
	fmt.Printf(">>> 响应时间max：%.0f ms\n", responseTimesMax)

	fmt.Printf(">>> 错误数：%d \n", errorCount)
	fmt.Printf(">>> 错误率：%.4f%% \n", float64(errorCount)/float64(numThreads*numCalls)*100)

}
