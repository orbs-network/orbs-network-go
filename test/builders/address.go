package builders

import (
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

func AddressForEd25519SignerForTests(setIndex int) primitives.Ripmd160Sha256 {
	keyPair := keys.Ed25519KeyPairForTests(setIndex)
	return hash.CalcRipmd160Sha256(keyPair.PublicKey())
}
