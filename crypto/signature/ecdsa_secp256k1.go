// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package signature

import (
	"encoding/hex"
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
		return nil, fmt.Errorf("cannot sign with edcsa secp256k1, private key has invalid length. expected=%d actual=%d",
			keys.ECDSA_SECP256K1_PRIVATE_KEY_SIZE_BYTES, len(privateKey))
	}
	return secp256k1.Sign(data, []byte(privateKey))
}

func VerifyEcdsaSecp256K1(publicKey primitives.EcdsaSecp256K1PublicKey, data []byte, sig primitives.EcdsaSecp256K1Sig) bool {
	if len(sig) == ECDSA_SECP256K1_SIGNATURE_SIZE_BYTES {
		sig = sig[:len(sig)-1]
	}
	if len(publicKey) != keys.ECDSA_SECP256K1_PUBLIC_KEY_SIZE_BYTES {
		return false
	}
	publicKeyWithBytePrefix := append([]byte{0x04}, publicKey...)
	return secp256k1.VerifySignature([]byte(publicKeyWithBytePrefix), data, sig)
}

func RecoverEcdsaSecp256K1(data []byte, sig primitives.EcdsaSecp256K1Sig) (primitives.EcdsaSecp256K1PublicKey, error) {
	if len(sig) != ECDSA_SECP256K1_SIGNATURE_SIZE_BYTES {
		msg := fmt.Sprintf("invalid signature length: expected=%d actual=%d sig=%s",
			ECDSA_SECP256K1_SIGNATURE_SIZE_BYTES, len(sig), sig)
		return nil, errors.New(msg)
	}
	publicKeyWithBytePrefix, err := secp256k1.RecoverPubkey(data, sig)
	if err != nil {
		return nil, errors.Wrapf(err, "secp256k1.RecoverPubkey() failed")
	}
	expectedPublicKeyLen := keys.ECDSA_SECP256K1_PUBLIC_KEY_SIZE_BYTES + 1
	if len(publicKeyWithBytePrefix) != expectedPublicKeyLen {
		return nil, errors.Errorf("invalid public key length returned by secp256k1.RecoverPubkey(). expected=%d actual=%d publicKeyWithBytePrefix=%s",
			expectedPublicKeyLen, len(publicKeyWithBytePrefix), hex.EncodeToString(publicKeyWithBytePrefix))
	}
	return publicKeyWithBytePrefix[1:], nil
}
