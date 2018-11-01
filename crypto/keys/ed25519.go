package keys

import (
	"encoding/hex"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ed25519"
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

func (k *Ed25519KeyPair) PublicKeyHex() string {
	return hex.EncodeToString(k.publicKey)
}

func (k *Ed25519KeyPair) PrivateKeyHex() string {
	return hex.EncodeToString(k.privateKey)
}

func GenerateEd25519Key() (*Ed25519KeyPair, error) {
	if pub, pri, err := ed25519.GenerateKey(nil); err != nil {
		return nil, errors.Wrapf(err, "cannot create new signature from random keys")
	} else {
		return NewEd25519KeyPair(primitives.Ed25519PublicKey(pub), primitives.Ed25519PrivateKey(pri)), nil
	}
}
