// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package hash

import (
	"crypto/sha256"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

const (
	SHA256_HASH_SIZE_BYTES = 32
)

func CalcSha256(data ...[]byte) primitives.Sha256 {
	s := sha256.New()
	for _, d := range data {
		s.Write(d)
	}
	return s.Sum(nil)
}
