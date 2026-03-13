package alg_ecdsa

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/bnb-chain/tss-lib/common"
	"github.com/bnb-chain/tss-lib/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/ecdsa/signing"
	"github.com/bnb-chain/tss-lib/tss"
)

// SignResult 一次 TSS 签名的结果：R、S 及 64 字节签名（R||S）。
type SignResult struct {
	R *big.Int
	S *big.Int
	// Signature 为 R||S，64 字节（secp256k1），可直接用于链上或 hex
	Signature []byte
	// Recovery 为 recovery id（以太坊风格），可选
	Recovery byte
}

// RunSign 在内存中跑完 TSS signing，各方通过 router 交换消息。
// signingNodeIDs 为参与签名的节点 ID 列表（必须是 keygen 时 PartyIDs 的子集且顺序一致）；
// keys 为对应各方的 keygen LocalPartySaveData（与 signingNodeIDs 一一对应）；
// msgHash 为待签名消息的哈希（应已在链/业务层转为 big.Int，且 < N）。
func RunSign(
	signingNodeIDs []string,
	keys []keygen.LocalPartySaveData,
	msgHash *big.Int,
	router MessageRouter,
) (*SignResult, error) {
	n := len(signingNodeIDs)
	if n != len(keys) || n < 2 || msgHash == nil {
		return nil, errors.New("mpc: invalid sign input")
	}
	if router == nil {
		return nil, errors.New("mpc: nil MessageRouter")
	}

	sortedIDs := PartyIDs(signingNodeIDs)
	threshold := n
	if n == 3 {
		threshold = 2
	} else if n == 5 {
		threshold = 3
	}

	outCh := make(chan tss.Message, n*8)
	endCh := make(chan common.SignatureData, n)
	errCh := make(chan *tss.Error, n*2)

	parties := make([]tss.Party, n)
	for i := 0; i < n; i++ {
		params := Parameters(sortedIDs, i, threshold)
		if params == nil {
			return nil, errors.New("mpc: failed to build sign parameters")
		}
		keySubset := keygen.BuildLocalSaveDataSubset(keys[i], sortedIDs)
		p := signing.NewLocalParty(msgHash, params, keySubset, outCh, endCh)
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

	var sigData *common.SignatureData
	done := 0
	for done < n {
		select {
		case err := <-errCh:
			close(outCh)
			return nil, err
		case data := <-endCh:
			done++
			if sigData == nil {
				sigData = &data
			}
		case <-time.After(2 * time.Minute):
			close(outCh)
			return nil, errors.New("mpc: sign timeout")
		}
	}
	close(outCh)
	wg.Wait()

	if sigData == nil {
		return nil, errors.New("mpc: no signature received")
	}
	r := new(big.Int).SetBytes(sigData.GetR())
	s := new(big.Int).SetBytes(sigData.GetS())
	sig := append(Pad32(r.Bytes()), Pad32(s.Bytes())...)
	rec := byte(0)
	if len(sigData.GetSignatureRecovery()) > 0 {
		rec = sigData.GetSignatureRecovery()[0]
	}
	return &SignResult{R: r, S: s, Signature: sig, Recovery: rec}, nil
}

// deepCopyKeys 深拷贝 keys 并恢复曲线，供 RunSignWithKDD 使用以免修改原数据。
func deepCopyKeys(keys []keygen.LocalPartySaveData) ([]keygen.LocalPartySaveData, error) {
	out := make([]keygen.LocalPartySaveData, len(keys))
	for i := range keys {
		raw, err := json.Marshal(&keys[i])
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &out[i]); err != nil {
			return nil, err
		}
		if out[i].ECDSAPub != nil {
			out[i].ECDSAPub.SetCurve(tss.S256())
		}
		for _, p := range out[i].BigXj {
			if p != nil {
				p.SetCurve(tss.S256())
			}
		}
	}
	return out, nil
}

// RunSignWithKDD 使用派生子密钥签名（HD 路径派生后的账户）。
// keyDerivationDelta 与 childPub 由 DeriveChildPubFromPath 得到；会拷贝 keys 并调 UpdatePublicKeyAndAdjustBigXj，不修改传入的 keys。
func RunSignWithKDD(
	signingNodeIDs []string,
	keys []keygen.LocalPartySaveData,
	msgHash *big.Int,
	keyDerivationDelta *big.Int,
	childPub *ecdsa.PublicKey,
	router MessageRouter,
) (*SignResult, error) {
	if keyDerivationDelta == nil || childPub == nil {
		return nil, errors.New("mpc: keyDerivationDelta and childPub required for RunSignWithKDD")
	}
	keysCopy, err := deepCopyKeys(keys)
	if err != nil {
		return nil, err
	}
	if err := signing.UpdatePublicKeyAndAdjustBigXj(keyDerivationDelta, keysCopy, childPub, tss.S256()); err != nil {
		return nil, err
	}
	n := len(signingNodeIDs)
	sortedIDs := PartyIDs(signingNodeIDs)
	threshold := n
	if n == 3 {
		threshold = 2
	} else if n == 5 {
		threshold = 3
	}
	outCh := make(chan tss.Message, n*8)
	endCh := make(chan common.SignatureData, n)
	errCh := make(chan *tss.Error, n*2)
	parties := make([]tss.Party, n)
	for i := 0; i < n; i++ {
		params := Parameters(sortedIDs, i, threshold)
		if params == nil {
			return nil, errors.New("mpc: failed to build sign parameters")
		}
		keySubset := keygen.BuildLocalSaveDataSubset(keysCopy[i], sortedIDs)
		p := signing.NewLocalPartyWithKDD(msgHash, params, keySubset, keyDerivationDelta, outCh, endCh)
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
			_ = router.Send(msg.GetFrom().Index, msg)
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
	var sigData *common.SignatureData
	done := 0
	for done < n {
		select {
		case err := <-errCh:
			close(outCh)
			return nil, err
		case data := <-endCh:
			done++
			if sigData == nil {
				sigData = &data
			}
		case <-time.After(2 * time.Minute):
			close(outCh)
			return nil, errors.New("mpc: sign timeout")
		}
	}
	close(outCh)
	wg.Wait()
	if sigData == nil {
		return nil, errors.New("mpc: no signature received")
	}
	r := new(big.Int).SetBytes(sigData.GetR())
	s := new(big.Int).SetBytes(sigData.GetS())
	sig := append(Pad32(r.Bytes()), Pad32(s.Bytes())...)
	rec := byte(0)
	if len(sigData.GetSignatureRecovery()) > 0 {
		rec = sigData.GetSignatureRecovery()[0]
	}
	return &SignResult{R: r, S: s, Signature: sig, Recovery: rec}, nil
}

// MessageHashFromTxHash 将 hex 编码的 32 字节交易哈希转为 tss signing 所需的 *big.Int。
func MessageHashFromTxHash(txHashHex string) (*big.Int, error) {
	b, err := hex.DecodeString(txHashHex)
	if err != nil {
		return nil, err
	}
	if len(b) != 32 {
		return nil, errors.New("mpc: tx hash must be 32 bytes")
	}
	return new(big.Int).SetBytes(b), nil
}
