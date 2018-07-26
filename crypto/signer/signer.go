package signer

import (
	"encoding/hex"
	"fmt"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ed25519"
)

type Signer struct {
	publicKey        primitives.Ed25519Pkey
	privateKeyUnsafe []byte
}

const (
	PUBLIC_KEY_SIZE  = 32
	PRIVATE_KEY_SIZE = 64
)

// TODO NewFromRandUnsafe() this should probably be removed in the future, used for debugging mainly
func NewFromRandUnsafe() (*Signer, error) {
	if pub, pri, err := ed25519.GenerateKey(nil); err != nil {
		return nil, errors.Wrapf(err, "cannot create new signer from random keys")
	} else {
		return &Signer{
			publicKey:        primitives.Ed25519Pkey(pub),
			privateKeyUnsafe: pri,
		}, nil
	}
}

func NewPublicKeyString(pk string) (*Signer, error) {
	pkBytes, err := hex.DecodeString(pk)
	if err != nil {
		return nil, errors.Wrapf(err, "public key hex decode failed")
	}

	return NewPublicKeyBytes(pkBytes)
}

func NewPublicKeyBytes(pk []byte) (*Signer, error) {
	if len(pk) != PUBLIC_KEY_SIZE {
		return nil, fmt.Errorf("invalid public key, length expected to be %d but data recevied was %v", PUBLIC_KEY_SIZE, pk)
	}
	s := &Signer{
		publicKey: pk,
	}

	return s, nil
}

func NewSecretKeyStringUnsafe(pk string) (*Signer, error) {
	pkBytes, err := hex.DecodeString(pk)
	if err != nil {
		return nil, errors.Wrapf(err, "public key hex invalid")
	}

	return NewSecretKeyBytesUnsafe(pkBytes)
}

func NewSecretKeyBytesUnsafe(pk []byte) (*Signer, error) {
	if len(pk) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key, length expected to be %d but data recevied was %v", PRIVATE_KEY_SIZE, pk)
	}

	pub := make([]byte, 32)
	copy(pub, pk[32:])
	pri := make([]byte, 64)
	copy(pri, pk)

	return &Signer{
		pub,
		pri,
	}, nil
}

func (e *Signer) PublicKey() primitives.Ed25519Pkey {
	return e.publicKey
}

func (e *Signer) PublicKeyHex() string {
	return hex.EncodeToString(e.publicKey)
}

func (e *Signer) PrivateKeyUnsafe() []byte {
	return e.privateKeyUnsafe
}

func (e *Signer) PrivateKeyUnsafeString() string {
	return hex.EncodeToString(e.privateKeyUnsafe)
}

func (e *Signer) Sign(data []byte) (primitives.Ed25519Sig, error) {
	if len(e.privateKeyUnsafe) != PRIVATE_KEY_SIZE {
		return nil, fmt.Errorf("cannot sign, private key invalid")
	}
	signedData := ed25519.Sign(e.PrivateKeyUnsafe(), data)
	return signedData, nil
}

func (e *Signer) Verify(data []byte, sig primitives.Ed25519Sig) bool {
	return ed25519.Verify([]byte(e.publicKey), data, sig)
}
