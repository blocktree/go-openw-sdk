package alg_ecdsa

import (
	"errors"
	"sync"
	"time"

	"github.com/bnb-chain/tss-lib/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/tss"
)

// KeygenConfig 配置 keygen 运行参数。
type KeygenConfig struct {
	// PreParamsTimeout 生成 PreParams 的超时（可选；若不生成则传 0 表示使用已有）
	PreParamsTimeout time.Duration
	// Concurrency 生成 PreParams 的并发数（可选）
	Concurrency int
}

// KeygenResult 一次 keygen 的结果：每方的 LocalPartySaveData 与公钥 KeyID。
type KeygenResult struct {
	// SaveData 每方一份，顺序与 PartyIDs 一致
	SaveData []keygen.LocalPartySaveData
	// KeyID 公钥标识（可由 KeyIDFromSaveData(ECDSAPub.X(), ECDSAPub.Y()) 得到）
	KeyID string
}

// RunKeygen 在内存中跑完 TSS keygen，各方通过 router 交换消息。
// nodeIDs 为参与方 ID 列表（会排序后生成 PartyIDs），threshold 为门限 t（如 2-of-3 则 threshold=2）。
// preParams 可选，若为 nil 且 config.PreParamsTimeout > 0 则现场生成（较慢）。
func RunKeygen(
	nodeIDs []string,
	threshold int,
	router MessageRouter,
	config KeygenConfig,
	preParams []*keygen.LocalPreParams,
) (*KeygenResult, error) {
	n := len(nodeIDs)
	if n < 2 || threshold < 1 || threshold > n {
		return nil, errors.New("mpc: invalid (threshold, n)")
	}
	if router == nil {
		return nil, errors.New("mpc: nil MessageRouter")
	}

	sortedIDs := PartyIDs(nodeIDs)
	outCh := make(chan tss.Message, n*4)
	endCh := make(chan keygen.LocalPartySaveData, n)
	errCh := make(chan *tss.Error, n*2)

	parties := make([]tss.Party, n)
	for i := 0; i < n; i++ {
		params := Parameters(sortedIDs, i, threshold)
		if params == nil {
			return nil, errors.New("mpc: failed to build parameters")
		}
		var opts []keygen.LocalPreParams
		if preParams != nil && i < len(preParams) && preParams[i] != nil && preParams[i].ValidateWithProof() {
			opts = append(opts, *preParams[i])
		} else if config.PreParamsTimeout > 0 {
			gen, err := keygen.GeneratePreParams(config.PreParamsTimeout, config.Concurrency)
			if err != nil {
				return nil, err
			}
			opts = append(opts, *gen)
		}
		p := keygen.NewLocalParty(params, outCh, endCh, opts...)
		parties[i] = p
	}

	inProcess, ok := router.(*InProcessRouter)
	if ok {
		inProcess.Parties = parties
		inProcess.SortedIDs = sortedIDs
		inProcess.ErrCh = errCh
		inProcess.partyByKey = make(map[string]int)
		for i, p := range parties {
			inProcess.partyByKey[string(p.PartyID().Key)] = i
		}
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for msg := range outCh {
			fromIdx := msg.GetFrom().Index
			_ = router.Send(fromIdx, msg)
		}
	}()

	for i := 0; i < n; i++ {
		go func(p tss.Party) {
			if err := p.Start(); err != nil {
				select {
				case errCh <- err:
				default:
				}
			}
		}(parties[i])
	}

	saveDataList := make([]keygen.LocalPartySaveData, n)
	var keyID string
	received := 0
	for received < n {
		select {
		case err := <-errCh:
			close(outCh)
			return nil, err
		case save := <-endCh:
			idx, err := save.OriginalIndex()
			if err != nil {
				close(outCh)
				return nil, err
			}
			saveDataList[idx] = save
			received++
			if save.ECDSAPub != nil {
				keyID = KeyIDFromSaveData(save.ECDSAPub.X(), save.ECDSAPub.Y())
			}
		case <-time.After(5 * time.Minute):
			close(outCh)
			return nil, errors.New("mpc: keygen timeout")
		}
	}
	close(outCh)
	wg.Wait()

	return &KeygenResult{SaveData: saveDataList, KeyID: keyID}, nil
}
