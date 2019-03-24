// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package hash

import (
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

const (
	KECCAK256_HASH_SIZE_BYTES = 32
)

func CalcKeccak256(data []byte) primitives.Keccak256 {
	return crypto.Keccak256(data)
}
