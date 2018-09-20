package hash

import (
	"crypto/sha256"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

const (
	SHA256_HASH_SIZE_BYTES = 32
)

func CalcSha256(data []byte) primitives.Sha256 {
	hash := sha256.Sum256(data)
	return hash[:]
}
