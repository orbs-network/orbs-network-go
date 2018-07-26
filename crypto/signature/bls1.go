package signature

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

// TODO: insert real implementations

func SignBls1(privateKey primitives.Bls1PrivateKey, data []byte) primitives.Bls1Sig {
	return []byte{0x88}
}

func VerifyBls1(publicKey primitives.Bls1PublicKey, data []byte, signature primitives.Bls1Sig) bool {
	if signature.Equal([]byte{0x88}) {
		return true
	} else {
		return false
	}
}
