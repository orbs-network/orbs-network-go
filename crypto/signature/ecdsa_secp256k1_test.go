// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package signature

import (
	"encoding/hex"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/stretchr/testify/require"
	"testing"
)

var someDataToSign_EcdsaSecp256K1 = hash.CalcSha256([]byte("this is what we want to sign"))
var expectedSigByKeyPair0_EcdsaSecp256K1 = "e894703fa57f2080ab6890fcc3bb138ff5fee7dfcb1c1ad5bc23e9ae882546ca1072b78b600afbe0f1292b648674fa4a4e8af11e1e0436f73873748ef508dae1"

func TestSignEcdsaSecp256K1(t *testing.T) {
	kp := keys.EcdsaSecp256K1KeyPairForTests(1)

	sig, err := SignEcdsaSecp256K1(kp.PrivateKey(), someDataToSign_EcdsaSecp256K1)
	require.NoError(t, err)
	require.Equal(t, ECDSA_SECP256K1_SIGNATURE_SIZE_BYTES, len(sig))

	ok := VerifyEcdsaSecp256K1(kp.PublicKey(), someDataToSign_EcdsaSecp256K1, sig)
	require.True(t, ok, "verification should succeed")
}

func TestSignEcdsaSecp256K1InvalidPrivateKey(t *testing.T) {
	_, err := SignEcdsaSecp256K1([]byte{0}, someDataToSign_EcdsaSecp256K1)
	require.Error(t, err, "sign with invalid pk should fail")
}

func TestVerifyEcdsaSecp256K1(t *testing.T) {
	kp := keys.EcdsaSecp256K1KeyPairForTests(0)

	expectedSigBytes, err := hex.DecodeString(expectedSigByKeyPair0_EcdsaSecp256K1)
	require.NoError(t, err)
	ok := VerifyEcdsaSecp256K1(kp.PublicKey(), someDataToSign_EcdsaSecp256K1, expectedSigBytes)
	require.True(t, ok, "verification should succeed")
}

func TestVerifyEcdsaSecp256K1InvalidPublicKey(t *testing.T) {
	expectedSigBytes, err := hex.DecodeString(expectedSigByKeyPair0_EcdsaSecp256K1)
	require.NoError(t, err)
	ok := VerifyEcdsaSecp256K1([]byte{0}, someDataToSign_EcdsaSecp256K1, expectedSigBytes)
	require.False(t, ok, "verification should fail")
}

func TestRecoverEcdsaSecp256K1(t *testing.T) {
	kp := keys.EcdsaSecp256K1KeyPairForTests(1)

	sig, err := SignEcdsaSecp256K1(kp.PrivateKey(), someDataToSign_EcdsaSecp256K1)
	require.NoError(t, err)
	require.Equal(t, ECDSA_SECP256K1_SIGNATURE_SIZE_BYTES, len(sig))

	publicKey, err := RecoverEcdsaSecp256K1(someDataToSign_EcdsaSecp256K1, sig)
	require.NoError(t, err)
	require.EqualValues(t, kp.PublicKey(), publicKey, "recovered public key should match original")
}

// TODO (v1) add benchmarks like in ed25519
