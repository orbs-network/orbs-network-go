package elliptic

import (
	"encoding/hex"
	"fmt"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ed25519"
)

type Signer struct {
	publicKey        []byte
	privateKeyUnsafe []byte
}

// TODO NewFromRandUnsafe() this should probably be removed in the future, used for debugging mainly
func NewFromRandUnsafe() (*Signer, error) {
	if pub, pri, err := ed25519.GenerateKey(nil); err != nil {
		return nil, errors.Wrapf(err, "cannot create new signer from random keys")
	} else {
		return &Signer{
			publicKey:        pub,
			privateKeyUnsafe: pri,
		}, nil
	}
}

func NewPublicKeyString(pk string) (*Signer, error) {
	pkBytes, err := hex.DecodeString(pk)
	if err != nil {
		return nil, errors.Wrapf(err, "public key hex decode failed")
	}

	return NewPublicKeyBytes(pkBytes), nil
}

func NewPublicKeyBytes(pk []byte) *Signer {
	s := &Signer{
		publicKey: pk,
	}

	return s
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
		return nil, fmt.Errorf("invalid private key")
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

func (e *Signer) PublicKey() []byte {
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

func (e *Signer) Sign(data []byte) string {
	// not implementing until we start using secure memory or other decision is made
	return ""
}

func (e *Signer) Verify(data []byte, sig string) bool {
	return false
}
