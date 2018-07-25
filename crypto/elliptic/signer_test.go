package elliptic_test

import (
	"bytes"
	"encoding/hex"
	"github.com/orbs-network/orbs-network-go/crypto/elliptic"
	"testing"
)

const (
	publicKey1  = "b9a91acbf23c22123a8253cfc4325d7b4b7a620465c57f932c7943f60887308b"
	publicKey2  = "dfc06c5be24a67adee80b35ab4f147bb1a35c55ff85eda69f40ef827bddec173"
	privateKey2 = "93e919986a22477fda016789cca30cb841a135650938714f85f0000a65076bd4dfc06c5be24a67adee80b35ab4f147bb1a35c55ff85eda69f40ef827bddec173"
)

var someDataToSign = []byte("this is what we want to sign")
var expectedSigByPK2 = "b228422c0c2b384bc60c7e0b14107b609d5c0d6fe72d6c6fbdd5ade28f017d3b8bc9a3f69ae8797af20ae31b8407f814c2852d0110140ef202ce719786eabd0c"

func pkStringToBytes(t *testing.T, pk string) []byte {
	pk1bytes, err := hex.DecodeString(pk)
	if err != nil {
		t.Errorf("something went wrong with pk->bytes %s", err)
	}
	return pk1bytes
}

func TestNewPublicKeyString(t *testing.T) {
	if s, err := elliptic.NewPublicKeyString(publicKey1); err != nil {
		t.Error(err)
	} else {
		if !bytes.Equal(s.PublicKey(), pkStringToBytes(t, publicKey1)) {
			t.Error("falied to create a valid signer object")
		}
	}
}

func TestNewPublicKeyStringFailsOnInvalidPKString(t *testing.T) {
	if _, err := elliptic.NewPublicKeyString("z" + publicKey1); err == nil {
		t.Errorf("signer initialized on invalid pk string")
	}
}

func TestNewPublicKeyBytesInvalid(t *testing.T) {
	if _, err := elliptic.NewPublicKeyBytes([]byte{0}); err == nil {
		t.Errorf("signer initialized on invalid pk bytes")
	}
}

func TestNewSecretKeyStringUnsafe(t *testing.T) {
	if s, err := elliptic.NewSecretKeyStringUnsafe(privateKey2); err != nil {
		t.Error(err)
	} else {
		if !bytes.Equal(s.PublicKey(), pkStringToBytes(t, publicKey2)) {
			t.Errorf("falied to create a valid signer object, publicKey is %v, should be %v", s.PublicKey(), pkStringToBytes(t, publicKey2))
		}
		if !bytes.Equal(s.PrivateKeyUnsafe(), pkStringToBytes(t, privateKey2)) {
			t.Errorf("falied to create a valid signer object, privateKey is %v, should be %v", s.PrivateKeyUnsafe(), pkStringToBytes(t, privateKey2))
		}
	}
}

func TestNewSecretKeyStringUnsafeFailedOnInvalidPKString(t *testing.T) {
	if _, err := elliptic.NewSecretKeyStringUnsafe("z" + privateKey2); err == nil {
		t.Error("signed initilaized on invalid pk sting")
	}
}

func TestNewSecretKeyBytesUnsafeInvalid(t *testing.T) {
	if _, err := elliptic.NewSecretKeyBytesUnsafe([]byte{0}); err == nil {
		t.Errorf("signer initialized on invalid pk bytes")
	}
}

func TestSignerCanSign(t *testing.T) {
	if s, err := elliptic.NewSecretKeyStringUnsafe(privateKey2); err != nil {
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
	if s, err := elliptic.NewPublicKeyString(publicKey1); err != nil {
		t.Error(err)
	} else {
		if _, err := s.Sign(someDataToSign); err == nil {
			t.Error("signer was able to sign a message without a private key")
		}
	}
}

func TestSignerCanVerify(t *testing.T) {
	if s, err := elliptic.NewPublicKeyString(publicKey2); err != nil {
		t.Error(err)
	} else {
		if expectedSigByPK2B, err := hex.DecodeString(expectedSigByPK2); err != nil {
			t.Error(err)
		} else {
			if !s.Verify(someDataToSign, expectedSigByPK2B) {
				t.Error("verification failed")
			}
		}
	}
}

func TestSignerVerificationFailedOnIncorrectPK(t *testing.T) {
	if s, err := elliptic.NewPublicKeyString(publicKey1); err != nil {
		t.Error(err)
	} else {
		if expectedSigByPK2B, err := hex.DecodeString(expectedSigByPK2); err != nil {
			t.Error(err)
		} else {
			if s.Verify(someDataToSign, expectedSigByPK2B) {
				t.Error("verification succeeded although PK is wrong")
			}
		}
	}
}
