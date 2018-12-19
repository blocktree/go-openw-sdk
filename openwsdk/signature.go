package openwsdk

import (
	"encoding/hex"
	"fmt"
	"github.com/blocktree/OpenWallet/hdkeystore"
	"github.com/blocktree/go-owcdrivers/btcTransaction"
	"github.com/blocktree/go-owcrypt"
	"math/big"
	"strings"
)


//SignRawTransaction 签名交易单
func SignRawTransaction(rawTx *RawTransaction, key *hdkeystore.HDKey) error {

	keySignatures := rawTx.Signatures[rawTx.AccountID]
	if keySignatures != nil {
		for _, keySignature := range keySignatures {

			childKey, err := key.DerivedKeyWithPath(keySignature.DerivedPath, keySignature.EccType)
			keyBytes, err := childKey.GetPrivateKeyBytes()
			if err != nil {
				return err
			}
			//log.Debug("privateKey:", hex.EncodeToString(keyBytes))

			//privateKeys = append(privateKeys, keyBytes)
			txHash, err := hex.DecodeString(keySignature.Message)
			//transHash = append(transHash, txHash)

			//log.Debug("hash:", txHash)

			//签名交易
			/////////交易单哈希签名
			signature, err := signTxHash(rawTx.Coin.Symbol, txHash, keyBytes, keySignature.EccType)
			if err != nil {
				return fmt.Errorf("transaction hash sign failed, unexpected error: %v", err)
			}

			//log.Debug("Signature:", txHash)

			keySignature.Signature = hex.EncodeToString(signature)
		}
	}

	rawTx.Signatures[rawTx.AccountID] = keySignatures

	return nil
}

//signTxHash 签名交易单哈希
func signTxHash(symbol string, msg []byte, privateKey []byte, eccType uint32) ([]byte, error) {
	var sig []byte
	if strings.EqualFold(symbol, "ETH") {
		sig, err := owcrypt.ETHsignature(privateKey, msg)
		if err != owcrypt.SUCCESS {
			return nil, fmt.Errorf("ETH sign hash failed")
		}
		return sig, nil
	}

	if strings.EqualFold(symbol, "NAS") {
		sig, err := owcrypt.NAS_signature(privateKey, msg)
		if err != owcrypt.SUCCESS {
			return nil, fmt.Errorf("NAS sign hash failed")
		}
		return sig, nil
	}

	sig, err := owcrypt.Signature(privateKey, nil, 0, msg, 32, eccType)
	if err != owcrypt.SUCCESS {
		return nil, fmt.Errorf("ECC sign hash failed")
	}
	sig = serilizeS(sig)
	return sig, nil
}

//serilizeS
func serilizeS(sig []byte) []byte {
	s := sig[32:]
	numS := new(big.Int).SetBytes(s)
	numHalfOrder := new(big.Int).SetBytes(btcTransaction.HalfCurveOrder)
	if numS.Cmp(numHalfOrder) > 0 {
		numOrder := new(big.Int).SetBytes(btcTransaction.CurveOrder)
		numS.Sub(numOrder, numS)

		return append(sig[:32], numS.Bytes()...)
	}
	return sig
}
