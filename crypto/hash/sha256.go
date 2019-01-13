package hash

import (
	"crypto/sha256"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

const (
	SHA256_HASH_SIZE_BYTES = 32
)

func CalcSha256(data ...[]byte) primitives.Sha256 {
	s := sha256.New()
	for _, d := range data {
		s.Write(d)
	}
	return s.Sum(nil)
}
