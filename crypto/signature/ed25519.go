package signature

import (
	"encoding/hex"
	"fmt"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ed25519"
)

type ed25519Signer struct {
	publicKey        primitives.Ed25519PublicKey
	privateKeyUnsafe primitives.Ed25519PrivateKey
}

const (
	PUBLIC_KEY_SIZE  = 32
	PRIVATE_KEY_SIZE = 64
)

// TODO newSignerFromRandUnsafe() this should probably be removed in the future, used for debugging mainly
func newSignerFromRandUnsafe() (*ed25519Signer, error) {
	if pub, pri, err := ed25519.GenerateKey(nil); err != nil {
		return nil, errors.Wrapf(err, "cannot create new signature from random keys")
	} else {
		return &ed25519Signer{
			publicKey:        primitives.Ed25519PublicKey(pub),
			privateKeyUnsafe: primitives.Ed25519PrivateKey(pri),
		}, nil
	}
}

func NewEd25519SignerPublicKey(publicKey []byte) (*ed25519Signer, error) {
	if len(publicKey) != PUBLIC_KEY_SIZE {
		return nil, fmt.Errorf("invalid public key, length expected to be %d but data recevied was %v", PUBLIC_KEY_SIZE, publicKey)
	}
	s := &ed25519Signer{
		publicKey: publicKey,
	}

	return s, nil
}

func NewEd25519SignerSecretKeyUnsafe(privateKey []byte) (*ed25519Signer, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key, length expected to be %d but data recevied was %v", PRIVATE_KEY_SIZE, privateKey)
	}

	pub := make([]byte, 32)
	copy(pub, privateKey[32:])
	pri := make([]byte, 64)
	copy(pri, privateKey)

	return &ed25519Signer{
		publicKey:        pub,
		privateKeyUnsafe: pri,
	}, nil
}

func (e *ed25519Signer) PublicKey() primitives.Ed25519PublicKey {
	return e.publicKey
}

func (e *ed25519Signer) PublicKeyHex() string {
	return hex.EncodeToString(e.publicKey)
}

func (e *ed25519Signer) PrivateKeyUnsafe() primitives.Ed25519PrivateKey {
	return e.privateKeyUnsafe
}

func (e *ed25519Signer) PrivateKeyUnsafeHex() string {
	return hex.EncodeToString(e.privateKeyUnsafe)
}

func (e *ed25519Signer) Sign(data []byte) (primitives.Ed25519Sig, error) {
	return SignEd25519(e.privateKeyUnsafe, data)
}

func (e *ed25519Signer) Verify(data []byte, sig primitives.Ed25519Sig) bool {
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
	if len(publicKey) != PUBLIC_KEY_SIZE {
		return false
	}
	return ed25519.Verify([]byte(publicKey), data, signature)
}
