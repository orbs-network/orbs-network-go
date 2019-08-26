package builders

import (
	"github.com/orbs-network/orbs-network-go/crypto/hash"
)

type hashObj struct {
	size int
	firstByteValue byte
}

func HashObj() *hashObj {
	return &hashObj{size:hash.SHA256_HASH_SIZE_BYTES}
}

func EmptyHash() []byte {
	return []byte{}
}

func (h *hashObj)WithFirstByte(b byte) *hashObj {
	h.firstByteValue = b
	return h
}

func (h *hashObj) Build() []byte {
	o := make([]byte, h.size)
	o[0] = h.firstByteValue
	return o
}
