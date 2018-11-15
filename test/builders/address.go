package builders

import (
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/crypto/keys"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

func AddressForEd25519SignerForTests(setIndex int) primitives.Ripmd160Sha256 {
	keyPair := testKeys.Ed25519KeyPairForTests(setIndex)
	return AddressFor(keyPair)
}

func AddressFor(keyPair *keys.Ed25519KeyPair) primitives.Ripmd160Sha256 {
	return hash.CalcRipmd160Sha256(keyPair.PublicKey())
}


