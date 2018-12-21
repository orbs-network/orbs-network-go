package signature

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"golang.org/x/crypto/ed25519"
)

const (
	ED25519_SIGNATURE_SIZE_BYTES = 64
)

func SignEd25519(privateKey primitives.Ed25519PrivateKey, data []byte) (primitives.Ed25519Sig, error) {
	if len(privateKey) != keys.ED25519_PRIVATE_KEY_SIZE_BYTES {
		return nil, fmt.Errorf("cannot sign with ed25519, private key invalid")
	}
	signedData := ed25519.Sign([]byte(privateKey), data)
	return signedData, nil
}

func VerifyEd25519(publicKey primitives.Ed25519PublicKey, data []byte, sig primitives.Ed25519Sig) bool {
	if len(publicKey) != keys.ED25519_PUBLIC_KEY_SIZE_BYTES {
		return false
	}
	return ed25519.Verify([]byte(publicKey), data, sig)
}
