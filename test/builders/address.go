package builders

import (
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/crypto/keys"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

func AddressForEd25519SignerForTests(setIndex int) primitives.Ripemd160Sha256 {
	keyPair := testKeys.Ed25519KeyPairForTests(setIndex)
	return AddressFor(keyPair)
}

func AddressFor(keyPair *keys.Ed25519KeyPair) primitives.Ripemd160Sha256 {
	return hash.CalcRipemd160Sha256(keyPair.PublicKey())
}
