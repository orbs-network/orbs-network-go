// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package keys

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	cryptorand "crypto/rand"
	"encoding/hex"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
)

// if there is no go-ethereum dependency in the project
// import "github.com/orbs-network/secp256k1-go"
// instead of "github.com/ethereum/go-ethereum/crypto/secp256k1"
// we can't import it when go-ethereum is linked due to linking collisions

const (
	ECDSA_SECP256K1_PUBLIC_KEY_SIZE_BYTES  = 64
	ECDSA_SECP256K1_PRIVATE_KEY_SIZE_BYTES = 32
)

type EcdsaSecp256K1KeyPair struct {
	publicKey  primitives.EcdsaSecp256K1PublicKey
	privateKey primitives.EcdsaSecp256K1PrivateKey
}

func NewEcdsaSecp256K1KeyPair(publicKey primitives.EcdsaSecp256K1PublicKey, privateKey primitives.EcdsaSecp256K1PrivateKey) *EcdsaSecp256K1KeyPair {
	return &EcdsaSecp256K1KeyPair{publicKey, privateKey}
}

func (k *EcdsaSecp256K1KeyPair) PublicKey() primitives.EcdsaSecp256K1PublicKey {
	return k.publicKey
}

func (k *EcdsaSecp256K1KeyPair) PrivateKey() primitives.EcdsaSecp256K1PrivateKey {
	return k.privateKey
}

func (k *EcdsaSecp256K1KeyPair) PublicKeyHex() string {
	return hex.EncodeToString(k.publicKey)
}

func (k *EcdsaSecp256K1KeyPair) PrivateKeyHex() string {
	return hex.EncodeToString(k.privateKey)
}

func GenerateEcdsaSecp256K1Key() (*EcdsaSecp256K1KeyPair, error) {
	pri, err := ecdsa.GenerateKey(secp256k1.S256(), cryptorand.Reader)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot create key pair")
	}
	publicKeyWithBytePrefix := elliptic.Marshal(pri.PublicKey.Curve, pri.PublicKey.X, pri.PublicKey.Y)
	privateKey := math.PaddedBigBytes(pri.D, pri.Params().BitSize/8)
	return NewEcdsaSecp256K1KeyPair(publicKeyWithBytePrefix[1:], privateKey), nil
}
