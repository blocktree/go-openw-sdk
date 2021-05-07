package openwsdk

import (
	"encoding/hex"
	"github.com/blocktree/go-owcrypt"
	"math/big"
)

const GROUP_ORDER = "73EDA753299D7D483339D80809A1D80553BDA402FFFE5BFEFFFFFFFF00000001"
const DEFAULT_HIDDEN_PUZZLE_HASH = "711d6c4e32c92e53179b199484cf8c897542bc57f2b22582799f9d657eec4699"


func chia_complement_bytes_to_bigint(data []byte) *big.Int {
	if data[0]&0x80 == 0 {
		return new(big.Int).SetBytes(data)
	}
	data[0] &= 0x7F
	data_big := new(big.Int).SetBytes(data)
	data_big = data_big.Sub(data_big, big.NewInt(1))
	data_bytes := data_big.Bytes()
	for i, _ := range data_bytes {
		data_bytes[i] ^= 0xFF
	}
	data_bytes[0] &= 0x7F
	data_big = new(big.Int).SetBytes(data_bytes)
	return data_big.Neg(data_big)
}


func calculate_synthetic_offset(public_key, hidden_puzzle_hash []byte) *big.Int {
	//offset := new(big.Int).SetBytes(owcrypt.Hash(append(public_key, hidden_puzzle_hash...), 0, owcrypt.HASH_ALG_SHA256))
	offset := chia_complement_bytes_to_bigint(owcrypt.Hash(append(public_key, hidden_puzzle_hash...), 0, owcrypt.HASH_ALG_SHA256))
	groupOrderBytes, _ := hex.DecodeString(GROUP_ORDER)
	groupOrder := new(big.Int).SetBytes(groupOrderBytes)
	offset = offset.Mod(offset, groupOrder)

	return offset
}

func Calculate_synthetic_secret_key(prikey []byte) []byte {
	default_hidden_puzzle_hash, _ := hex.DecodeString(DEFAULT_HIDDEN_PUZZLE_HASH)
	secret_exponent := new(big.Int).SetBytes(prikey)
	public_key, _ := owcrypt.GenPubkey(prikey, owcrypt.ECC_CURVE_BLS12381_G2_XMD_SHA_256_SSWU_RO_NUL)
	synthetic_offset := calculate_synthetic_offset(public_key, default_hidden_puzzle_hash)
	synthetic_secret_exponent := new(big.Int).Add(secret_exponent, synthetic_offset)
	groupOrderBytes, _ := hex.DecodeString(GROUP_ORDER)
	groupOrder := new(big.Int).SetBytes(groupOrderBytes)
	synthetic_secret_exponent = synthetic_secret_exponent.Mod(synthetic_secret_exponent, groupOrder)

	return synthetic_secret_exponent.Bytes()
}

