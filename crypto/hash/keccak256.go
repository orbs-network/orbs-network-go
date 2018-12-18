package hash

import (
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

const (
	KECCAK256_HASH_SIZE_BYTES = 32
)

func CalcKeccak256(data []byte) primitives.Keccak256 {
	return crypto.Keccak256(data)
}
