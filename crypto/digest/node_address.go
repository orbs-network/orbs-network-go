// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package digest

import (
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

const (
	NODE_ADDRESS_SIZE_BYTES = 20
)

func CalcNodeAddressFromPublicKey(publicKey primitives.EcdsaSecp256K1PublicKey) primitives.NodeAddress {
	return primitives.NodeAddress(hash.CalcKeccak256(publicKey)[12:])
}
