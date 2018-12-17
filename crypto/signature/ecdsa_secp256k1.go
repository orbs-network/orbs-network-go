package signature

import (
	"fmt"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/orbs-network/orbs-network-go/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
)

// if there is no go-ethereum dependency in the project
// import "github.com/orbs-network/secp256k1-go"
// instead of "github.com/ethereum/go-ethereum/crypto/secp256k1"
// we can't import it when go-ethereum is linked due to linking collisions

const (
	ECDSA_SECP256K1_SIGNATURE_SIZE_BYTES = 65 // with recovery, without recovery we can skip the last byte
)

// the given data must not be controlled by an adversary, it must be a hash over given data
func SignEcdsaSecp256K1(privateKey primitives.EcdsaSecp256K1PrivateKey, data []byte) (primitives.EcdsaSecp256K1Sig, error) {
	if len(privateKey) != keys.ECDSA_SECP256K1_PRIVATE_KEY_SIZE_BYTES {
		return nil, fmt.Errorf("cannot sign with edcsa secp256k1, private key invalid")
	}
	return secp256k1.Sign(data, []byte(privateKey))
}

func VerifyEcdsaSecp256K1(publicKey primitives.EcdsaSecp256K1PublicKey, data []byte, signature primitives.EcdsaSecp256K1Sig) bool {
	if len(signature) == ECDSA_SECP256K1_SIGNATURE_SIZE_BYTES {
		signature = signature[:len(signature)-1]
	}
	if len(publicKey) != keys.ECDSA_SECP256K1_PUBLIC_KEY_SIZE_BYTES {
		return false
	}
	publicKeyWithBytePrefix := append([]byte{0x04}, publicKey...)
	return secp256k1.VerifySignature([]byte(publicKeyWithBytePrefix), data, signature)
}

func RecoverEcdsaSecp256K1(data []byte, signature primitives.EcdsaSecp256K1Sig) (primitives.EcdsaSecp256K1PublicKey, error) {
	if len(signature) != ECDSA_SECP256K1_SIGNATURE_SIZE_BYTES {
		return nil, errors.New("invalid signature size")
	}
	publicKeyWithBytePrefix, err := secp256k1.RecoverPubkey(data, signature)
	if err != nil {
		return nil, err
	}
	if len(publicKeyWithBytePrefix) != keys.ECDSA_SECP256K1_PUBLIC_KEY_SIZE_BYTES+1 {
		return nil, errors.Errorf("secp256k1.RecoverPubkey returned pub key with len %d", len(publicKeyWithBytePrefix))
	}
	return publicKeyWithBytePrefix[1:], nil
}
