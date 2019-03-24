// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package builders

import (
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

/// Test builders for: primitives.ClientAddress

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
