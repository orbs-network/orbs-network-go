package signature_test

import (
	"encoding/hex"
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	"github.com/orbs-network/orbs-network-go/test/crypto"
	"testing"
)

var someDataToSign = []byte("this is what we want to sign")
var expectedSigByKeyPair1 = "b228422c0c2b384bc60c7e0b14107b609d5c0d6fe72d6c6fbdd5ade28f017d3b8bc9a3f69ae8797af20ae31b8407f814c2852d0110140ef202ce719786eabd0c"

func TestNewPublicKeyString(t *testing.T) {
	kp := crypto.NewKeyPair(1)
	if s, err := signature.NewEd25519SignerPublicKeyString(hex.EncodeToString(kp.PublicKey())); err != nil {
		t.Error(err)
	} else {
		if !s.PublicKey().Equal(kp.PublicKey()) {
			t.Error("falied to create a valid signature object")
		}
	}
}

func TestNewPublicKeyStringFailsOnInvalidPKString(t *testing.T) {
	if _, err := signature.NewEd25519SignerPublicKeyString("z" + hex.EncodeToString(crypto.NewKeyPair(1).PublicKey())); err == nil {
		t.Errorf("signature initialized on invalid pk string")
	}
}

func TestNewPublicKeyBytesInvalid(t *testing.T) {
	if _, err := signature.NewEd25519SignerPublicKeyBytes([]byte{0}); err == nil {
		t.Errorf("signature initialized on invalid pk bytes")
	}
}

func TestNewSecretKeyStringUnsafe(t *testing.T) {
	kp := crypto.NewKeyPair(1)
	if s, err := signature.NewEd25519SignerSecretKeyStringUnsafe(hex.EncodeToString(kp.PrivateKeyUnsafe())); err != nil {
		t.Error(err)
	} else {
		if !s.PublicKey().Equal(kp.PublicKey()) {
			t.Errorf("falied to create a valid signature object, publicKey is %v, should be %v", s.PublicKey(), kp.PrivateKeyUnsafe())
		}
		if !s.PrivateKeyUnsafe().Equal(kp.PrivateKeyUnsafe()) {
			t.Errorf("falied to create a valid signature object, privateKey is %v, should be %v", s.PrivateKeyUnsafe(), kp.PrivateKeyUnsafe())
		}
	}
}

func TestNewSecretKeyStringUnsafeFailedOnInvalidPKString(t *testing.T) {
	if _, err := signature.NewEd25519SignerSecretKeyStringUnsafe("z" + hex.EncodeToString(crypto.NewKeyPair(1).PrivateKeyUnsafe())); err == nil {
		t.Error("signed initilaized on invalid pk sting")
	}
}

func TestNewSecretKeyBytesUnsafeInvalid(t *testing.T) {
	if _, err := signature.NewEd25519SignerSecretKeyBytesUnsafe([]byte{0}); err == nil {
		t.Errorf("signature initialized on invalid pk bytes")
	}
}

func TestSignerCanSign(t *testing.T) {
	if s, err := signature.NewEd25519SignerSecretKeyBytesUnsafe(crypto.NewKeyPair(1).PrivateKeyUnsafe()); err != nil {
		t.Error(err)
	} else {
		if sig, err := s.Sign(someDataToSign); err != nil {
			t.Error(err)
		} else {
			if !s.Verify(someDataToSign, sig) {
				t.Error("verification failed")
			}
		}
	}
}

func TestSignerFailedOnMissingPK(t *testing.T) {
	if s, err := signature.NewEd25519SignerPublicKeyBytes(crypto.NewKeyPair(1).PublicKey()); err != nil {
		t.Error(err)
	} else {
		if _, err := s.Sign(someDataToSign); err == nil {
			t.Error("signature was able to sign a message without a private key")
		}
	}
}

func TestSignerCanVerify(t *testing.T) {
	if s, err := signature.NewEd25519SignerPublicKeyBytes(crypto.NewKeyPair(1).PublicKey()); err != nil {
		t.Error(err)
	} else {
		if expectedSigByPK2B, err := hex.DecodeString(expectedSigByKeyPair1); err != nil {
			t.Error(err)
		} else {
			if !s.Verify(someDataToSign, expectedSigByPK2B) {
				t.Error("verification failed")
			}
		}
	}
}

func TestSignerVerificationFailedOnIncorrectPK(t *testing.T) {
	if s, err := signature.NewEd25519SignerPublicKeyBytes(crypto.NewKeyPair(2).PublicKey()); err != nil {
		t.Error(err)
	} else {
		if expectedSigByPK2B, err := hex.DecodeString(expectedSigByKeyPair1); err != nil {
			t.Error(err)
		} else {
			if s.Verify(someDataToSign, expectedSigByPK2B) {
				t.Error("verification succeeded although PK is wrong")
			}
		}
	}
}

func TestSignEd25519(t *testing.T) {
	kp := crypto.NewKeyPair(1)

	if sig, err := signature.SignEd25519(kp.PrivateKeyUnsafe(), someDataToSign); err != nil {
		t.Error(err)
	} else {
		if !signature.VerifyEd25519(kp.PublicKey(), someDataToSign, sig) {
			t.Error("verification failed")
		}
	}
}

func TestVerifyEd25519(t *testing.T) {
	kp := crypto.NewKeyPair(1)

	if expectedSigByPK2B, err := hex.DecodeString(expectedSigByKeyPair1); err != nil {
		t.Error(err)
	} else {
		if !signature.VerifyEd25519(kp.PublicKey(), someDataToSign, expectedSigByPK2B) {
			t.Error("verification failed")
		}
	}
}
