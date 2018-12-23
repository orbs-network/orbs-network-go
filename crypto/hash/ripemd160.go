package hash

import (
	"crypto/sha256"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"golang.org/x/crypto/ripemd160"
)

const (
	RIPEMD160_HASH_SIZE_BYTES = 20
)

func CalcRipemd160Sha256(data []byte) primitives.Ripemd160Sha256 {
	hash := sha256.Sum256(data)
	r := ripemd160.New()
	r.Write(hash[:])
	return r.Sum(nil)
}
