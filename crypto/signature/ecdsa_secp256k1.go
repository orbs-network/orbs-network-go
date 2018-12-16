package signature

import (
	"fmt"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/orbs-network/orbs-network-go/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

// if there is no go-ethereum dependency in the project
// import "github.com/orbs-network/secp256k1-go"
// instead of "github.com/ethereum/go-ethereum/crypto/secp256k1"
// we can't import it when go-ethereum is linked due to linking collisions

const (
	ECDSA_SECP256K1_SIGNATURE_SIZE_BYTES = 64
)

// the given data must not be controlled by an adversary, it must be a hash over given data
func SignEcdsaSecp256K1(privateKey primitives.EcdsaSecp256K1PrivateKey, data []byte) (primitives.EcdsaSecp256K1Sig, error) {
	if len(privateKey) != keys.ECDSA_SECP256K1_PRIVATE_KEY_SIZE_BYTES {
		return nil, fmt.Errorf("cannot sign with edcsa secp256k1, private key invalid")
	}
	sig, err := secp256k1.Sign(data, []byte(privateKey))
	if err != nil {
		return nil, err
	}
	return sig[:len(sig)-1], nil // remove recid to get a 64 byte sig instead of 65
}

func VerifyEcdsaSecp256K1(publicKey primitives.EcdsaSecp256K1PublicKey, data []byte, signature primitives.EcdsaSecp256K1Sig) bool {
	if len(publicKey) != keys.ECDSA_SECP256K1_PUBLIC_KEY_SIZE_BYTES {
		return false
	}
	return secp256k1.VerifySignature([]byte(publicKey), data, signature)
}
