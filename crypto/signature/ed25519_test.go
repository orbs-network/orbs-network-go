package signature

import (
	"encoding/hex"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"testing"
)

var someDataToSign = []byte("this is what we want to sign")
var expectedSigByKeyPair1 = "b228422c0c2b384bc60c7e0b14107b609d5c0d6fe72d6c6fbdd5ade28f017d3b8bc9a3f69ae8797af20ae31b8407f814c2852d0110140ef202ce719786eabd0c"

func TestSignEd25519(t *testing.T) {
	kp := keys.Ed25519KeyPairForTests(1)

	if sig, err := SignEd25519(kp.PrivateKey(), someDataToSign); err != nil {
		t.Error(err)
	} else {
		if !VerifyEd25519(kp.PublicKey(), someDataToSign, sig) {
			t.Error("verification failed")
		}
	}
}

func TestSignEd25519InvalidPrivateKey(t *testing.T) {
	if _, err := SignEd25519([]byte{0}, someDataToSign); err == nil {
		t.Error("sign successed with invalid pk")
	}
}

func TestVerifyEd25519(t *testing.T) {
	kp := keys.Ed25519KeyPairForTests(0)

	if expectedSigBytes, err := hex.DecodeString(expectedSigByKeyPair1); err != nil {
		t.Error(err)
	} else {
		if !VerifyEd25519(kp.PublicKey(), someDataToSign, expectedSigBytes) {
			t.Error("verification failed")
		}
	}
}

func TestVerifyEd25519InvalidPublicKey(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("code paniced, shouldn't have: %s", r)
		}
	}()

	if expectedSigBytes, err := hex.DecodeString(expectedSigByKeyPair1); err != nil {
		t.Error(err)
	} else {
		if VerifyEd25519([]byte{0}, someDataToSign, expectedSigBytes) {
			t.Errorf("no panic happened and verification succeeded without public key")
		}
	}
}

func BenchmarkSignEd25519(b *testing.B) {
	kp := keys.Ed25519KeyPairForTests(1)
	for i := 0; i < b.N; i++ {
		if _, err := SignEd25519(kp.PrivateKey(), someDataToSign); err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkVerifyEd25519(b *testing.B) {
	b.StopTimer()
	kp := keys.Ed25519KeyPairForTests(1)

	if sig, err := SignEd25519(kp.PrivateKey(), someDataToSign); err != nil {
		b.Error(err)
	} else {
		b.StartTimer()
		for i := 0; i < b.N; i++ {
			if !VerifyEd25519(kp.PublicKey(), someDataToSign, sig) {
				b.Error("verification failed")
			}
		}
	}
}

func BenchmarkSignAndVerifyEd25519(b *testing.B) {
	kp := keys.Ed25519KeyPairForTests(1)
	for i := 0; i < b.N; i++ {
		if sig, err := SignEd25519(kp.PrivateKey(), someDataToSign); err != nil {
			b.Error(err)
		} else {
			if !VerifyEd25519(kp.PublicKey(), someDataToSign, sig) {
				b.Error("verification failed")
			}
		}
	}
}
