package keys

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

const (
	ED25519_PUBLIC_KEY_SIZE_BYTES  = 32
	ED25519_PRIVATE_KEY_SIZE_BYTES = 64
)

type Ed25519KeyPair struct {
	publicKey  primitives.Ed25519PublicKey
	privateKey primitives.Ed25519PrivateKey
}

func NewEd25519KeyPair(publicKey primitives.Ed25519PublicKey, privateKey primitives.Ed25519PrivateKey) *Ed25519KeyPair {
	return &Ed25519KeyPair{publicKey, privateKey}
}

func (k *Ed25519KeyPair) PublicKey() primitives.Ed25519PublicKey {
	return k.publicKey
}

func (k *Ed25519KeyPair) PrivateKey() primitives.Ed25519PrivateKey {
	return k.privateKey
}
