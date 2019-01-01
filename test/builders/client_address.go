package builders

import (
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

func ClientAddressForEd25519SignerForTests(setIndex int) primitives.ClientAddress {
	keyPair := testKeys.Ed25519KeyPairForTests(setIndex)
	signer := (&protocol.SignerBuilder{
		Scheme: protocol.SIGNER_SCHEME_EDDSA,
		Eddsa: &protocol.EdDSA01SignerBuilder{
			NetworkType:     protocol.NETWORK_TYPE_TEST_NET,
			SignerPublicKey: keyPair.PublicKey(),
		},
	}).Build()

	res, _ := digest.CalcClientAddressOfEd25519Signer(signer)
	return res
}
