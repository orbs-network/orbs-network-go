package signature

import (
	"encoding/hex"
	"fmt"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ed25519"
)

type Ed25519Signer struct {
	publicKey        primitives.Ed25519PublicKey
	privateKeyUnsafe primitives.Ed25519PrivateKey
}

const (
	PUBLIC_KEY_SIZE  = 32
	PRIVATE_KEY_SIZE = 64
)

// TODO NewFromRandUnsafe() this should probably be removed in the future, used for debugging mainly
func NewFromRandUnsafe() (*Ed25519Signer, error) {
	if pub, pri, err := ed25519.GenerateKey(nil); err != nil {
		return nil, errors.Wrapf(err, "cannot create new signature from random keys")
	} else {
		return &Ed25519Signer{
			publicKey:        primitives.Ed25519PublicKey(pub),
			privateKeyUnsafe: primitives.Ed25519PrivateKey(pri),
		}, nil
	}
}

func NewEd25519SignerPublicKeyString(pk string) (*Ed25519Signer, error) {
	pkBytes, err := hex.DecodeString(pk)
	if err != nil {
		return nil, errors.Wrapf(err, "public key hex decode failed")
	}

	return NewEd25519SignerPublicKeyBytes(pkBytes)
}

func NewEd25519SignerPublicKeyBytes(pk []byte) (*Ed25519Signer, error) {
	if len(pk) != PUBLIC_KEY_SIZE {
		return nil, fmt.Errorf("invalid public key, length expected to be %d but data recevied was %v", PUBLIC_KEY_SIZE, pk)
	}
	s := &Ed25519Signer{
		publicKey: pk,
	}

	return s, nil
}

func NewEd25519SignerSecretKeyStringUnsafe(pk string) (*Ed25519Signer, error) {
	pkBytes, err := hex.DecodeString(pk)
	if err != nil {
		return nil, errors.Wrapf(err, "public key hex invalid")
	}

	return NewEd25519SignerSecretKeyBytesUnsafe(pkBytes)
}

func NewEd25519SignerSecretKeyBytesUnsafe(pk []byte) (*Ed25519Signer, error) {
	if len(pk) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key, length expected to be %d but data recevied was %v", PRIVATE_KEY_SIZE, pk)
	}

	pub := make([]byte, 32)
	copy(pub, pk[32:])
	pri := make([]byte, 64)
	copy(pri, pk)

	return &Ed25519Signer{
		publicKey:        pub,
		privateKeyUnsafe: pri,
	}, nil
}

func (e *Ed25519Signer) PublicKey() primitives.Ed25519PublicKey {
	return e.publicKey
}

func (e *Ed25519Signer) PublicKeyHex() string {
	return hex.EncodeToString(e.publicKey)
}

func (e *Ed25519Signer) PrivateKeyUnsafe() primitives.Ed25519PrivateKey {
	return e.privateKeyUnsafe
}

func (e *Ed25519Signer) PrivateKeyUnsafeString() string {
	return hex.EncodeToString(e.privateKeyUnsafe)
}

func (e *Ed25519Signer) Sign(data []byte) (primitives.Ed25519Sig, error) {
	return SignEd25519(e.privateKeyUnsafe, data)
}

func (e *Ed25519Signer) Verify(data []byte, sig primitives.Ed25519Sig) bool {
	return VerifyEd25519(e.publicKey, data, sig)
}

func SignEd25519(privateKey primitives.Ed25519PrivateKey, data []byte) (primitives.Ed25519Sig, error) {
	if len(privateKey) != PRIVATE_KEY_SIZE {
		return nil, fmt.Errorf("cannot sign, private key invalid")
	}
	signedData := ed25519.Sign([]byte(privateKey), data)
	return signedData, nil
}

func VerifyEd25519(publicKey primitives.Ed25519PublicKey, data []byte, signature primitives.Ed25519Sig) bool {
	return ed25519.Verify([]byte(publicKey), data, signature)
}
