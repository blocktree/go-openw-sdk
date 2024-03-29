package openwsdk

import (
	"encoding/hex"
	"fmt"
	"github.com/blocktree/go-owcrypt"
	"github.com/blocktree/openwallet/v2/hdkeystore"
	"github.com/blocktree/openwallet/v2/openwallet"
)

//SignRawTransaction 签名交易单
func SignRawTransaction(rawTx *RawTransaction, key *hdkeystore.HDKey) error {

	for accountID, keySignatures := range rawTx.Signatures {
		if keySignatures != nil {

			for _, keySignature := range keySignatures {

				if keySignature.EccType == owcrypt.ECC_CURVE_BLS12381_G2_XMD_SHA_256_SSWU_RO_AUG {

					childKey, err := key.DerivedKeyWithPath(keySignature.DerivedPath, keySignature.EccType)
					keyBytes, err := childKey.GetPrivateKeyBytes()
					if err != nil {
						return openwallet.NewError(openwallet.ErrSignRawTransactionFailed, err.Error())
					}
					message, err := hex.DecodeString(keySignature.Message)
					if err != nil {
						return err
					}
					key2 := Calculate_synthetic_secret_key(keyBytes)
					newKey := make([]byte,0)
					if len(key2) != 32{
						for i:=0 ;i< 32 - len(key2) ;i++{
							newKey = append(newKey, 0)
						}
					}
					newKey = append(newKey, key2...)
					signature, _, sigErr := owcrypt.Signature(newKey, nil, message, keySignature.EccType)
					if sigErr != owcrypt.SUCCESS {
						return fmt.Errorf("transaction hash sign failed")
					}
					keySignature.Signature = hex.EncodeToString(signature)
					continue
				}

				childKey, err := key.DerivedKeyWithPath(keySignature.DerivedPath, keySignature.EccType)
				keyBytes, err := childKey.GetPrivateKeyBytes()
				if err != nil {
					return err
				}
				//log.Debug("privateKey:", hex.EncodeToString(keyBytes))

				//privateKeys = append(privateKeys, keyBytes)
				txHash, err := hex.DecodeString(keySignature.Message)
				//transHash = append(transHash, txHash)

				//log.Infof("sign hash: %s", txHash)

				//签名交易
				/////////交易单哈希签名

				//signature, err := signatureSet.SignTxHash(rawTx.Coin.Symbol, txHash, keyBytes, keySignature.EccType)
				//if err != nil {
				//	return fmt.Errorf("transaction hash sign failed, unexpected error: %v", err)
				//}

				signature, v, sigErr := owcrypt.Signature(keyBytes, nil, txHash, keySignature.EccType)
				if sigErr != owcrypt.SUCCESS {
					return fmt.Errorf("transaction hash sign failed")
				}

				if keySignature.RSV {
					signature = append(signature, v)
				}

				//log.Debug("Signature:", txHash)

				keySignature.Signature = hex.EncodeToString(signature)
			}
		}
		rawTx.Signatures[accountID] = keySignatures
	}

	return nil
}

//SignTxHash 签名交易单Hash
func SignTxHash(signatures map[string][]*KeySignature, key *hdkeystore.HDKey) (map[string][]*KeySignature, error) {

	for accountID, keySignatures := range signatures {
		//log.Infof("accountID: %s", accountID)
		if keySignatures != nil {
			for _, keySignature := range keySignatures {
				if keySignature.EccType == owcrypt.ECC_CURVE_BLS12381_G2_XMD_SHA_256_SSWU_RO_AUG {

					childKey, err := key.DerivedKeyWithPath(keySignature.DerivedPath, keySignature.EccType)
					keyBytes, err := childKey.GetPrivateKeyBytes()
					if err != nil {
						return nil,openwallet.NewError(openwallet.ErrSignRawTransactionFailed, err.Error())
					}
					message, err := hex.DecodeString(keySignature.Message)
					if err != nil {
						return  nil,err
					}

					key2 := Calculate_synthetic_secret_key(keyBytes)
					newKey := make([]byte,0)
					if len(key2) != 32{
						for i:=0 ;i< 32 - len(key2) ;i++{
							newKey = append(newKey, 0)
						}
					}
					newKey = append(newKey, key2...)
					signature, _, sigErr := owcrypt.Signature(newKey, nil, message, keySignature.EccType)
					if sigErr != owcrypt.SUCCESS {
						return  nil,fmt.Errorf("transaction hash sign failed："+keySignature.Message)
					}
					keySignature.Signature = hex.EncodeToString(signature)
					continue
				}
				childKey, err := key.DerivedKeyWithPath(keySignature.DerivedPath, keySignature.EccType)
				keyBytes, err := childKey.GetPrivateKeyBytes()
				if err != nil {
					return nil, err
				}
				//log.Debug("privateKey:", hex.EncodeToString(keyBytes))

				//privateKeys = append(privateKeys, keyBytes)
				txHash, err := hex.DecodeString(keySignature.Message)
				//transHash = append(transHash, txHash)

				//log.Infof("sign hash: %s", txHash)

				//签名交易
				/////////交易单哈希签名

				//signature, err := signatureSet.SignTxHash(rawTx.Coin.Symbol, txHash, keyBytes, keySignature.EccType)
				//if err != nil {
				//	return fmt.Errorf("transaction hash sign failed, unexpected error: %v", err)
				//}

				signature, v, sigErr := owcrypt.Signature(keyBytes, nil, txHash, keySignature.EccType)
				if sigErr != owcrypt.SUCCESS {
					return nil, fmt.Errorf("transaction hash sign failed")
				}

				if keySignature.RSV {
					signature = append(signature, v)
				}

				//log.Debug("Signature:", txHash)

				keySignature.Signature = hex.EncodeToString(signature)
			}
		}
		signatures[accountID] = keySignatures
	}

	return signatures, nil
}
