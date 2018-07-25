package signature

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

// TODO: insert real implementations

func SignEd25519(privateKey []byte, data []byte) primitives.Ed25519Sig {
	return []byte{0x77}
}

func VerifyEd25519(publicKey primitives.Ed25519Pkey, data []byte, signature primitives.Ed25519Sig) bool {
	if signature.Equal([]byte{0x77}) {
		return true
	} else {
		return false
	}
}
