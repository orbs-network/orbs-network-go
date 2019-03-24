// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package signature

import (
	"encoding/hex"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/stretchr/testify/require"
	"testing"
)

var someDataToSign_Ed25519 = []byte("this is what we want to sign")
var expectedSigByKeyPair0_Ed25519 = "b228422c0c2b384bc60c7e0b14107b609d5c0d6fe72d6c6fbdd5ade28f017d3b8bc9a3f69ae8797af20ae31b8407f814c2852d0110140ef202ce719786eabd0c"

func TestSignEd25519(t *testing.T) {
	kp := keys.Ed25519KeyPairForTests(1)

	sig, err := SignEd25519(kp.PrivateKey(), someDataToSign_Ed25519)
	require.NoError(t, err)
	require.Equal(t, ED25519_SIGNATURE_SIZE_BYTES, len(sig))

	ok := VerifyEd25519(kp.PublicKey(), someDataToSign_Ed25519, sig)
	require.True(t, ok, "verification should succeed")
}

func TestSignEd25519InvalidPrivateKey(t *testing.T) {
	_, err := SignEd25519([]byte{0}, someDataToSign_Ed25519)
	require.Error(t, err, "sign with invalid pk should fail")
}

func TestVerifyEd25519(t *testing.T) {
	kp := keys.Ed25519KeyPairForTests(0)

	expectedSigBytes, err := hex.DecodeString(expectedSigByKeyPair0_Ed25519)
	require.NoError(t, err)
	ok := VerifyEd25519(kp.PublicKey(), someDataToSign_Ed25519, expectedSigBytes)
	require.True(t, ok, "verification should succeed")
}

func TestVerifyEd25519InvalidPublicKey(t *testing.T) {
	expectedSigBytes, err := hex.DecodeString(expectedSigByKeyPair0_Ed25519)
	require.NoError(t, err)
	ok := VerifyEd25519([]byte{0}, someDataToSign_Ed25519, expectedSigBytes)
	require.False(t, ok, "verification should fail")
}

func BenchmarkSignEd25519(b *testing.B) {
	kp := keys.Ed25519KeyPairForTests(1)
	for i := 0; i < b.N; i++ {
		if _, err := SignEd25519(kp.PrivateKey(), someDataToSign_Ed25519); err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkVerifyEd25519(b *testing.B) {
	b.StopTimer()
	kp := keys.Ed25519KeyPairForTests(1)

	if sig, err := SignEd25519(kp.PrivateKey(), someDataToSign_Ed25519); err != nil {
		b.Error(err)
	} else {
		b.StartTimer()
		for i := 0; i < b.N; i++ {
			if !VerifyEd25519(kp.PublicKey(), someDataToSign_Ed25519, sig) {
				b.Error("verification failed")
			}
		}
	}
}

func BenchmarkSignAndVerifyEd25519(b *testing.B) {
	kp := keys.Ed25519KeyPairForTests(1)
	for i := 0; i < b.N; i++ {
		if sig, err := SignEd25519(kp.PrivateKey(), someDataToSign_Ed25519); err != nil {
			b.Error(err)
		} else {
			if !VerifyEd25519(kp.PublicKey(), someDataToSign_Ed25519, sig) {
				b.Error("verification failed")
			}
		}
	}
}
