// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package hash

import (
	"github.com/stretchr/testify/require"
	"testing"
)

var someData = []byte("testing")

const (
	ExpectedSha256    = "cf80cd8aed482d5d1527d7dc72fceff84e6326592848447d2dc0b0e87dfc9a90"
	ExpectedKeccak256 = "5f16f4c7f149ac4f9510d9cf8cf384038ad348b3bcdc01915f95de12df9d1b02"
)

func TestCalcSha256(t *testing.T) {
	h := CalcSha256(someData)
	require.Equal(t, SHA256_HASH_SIZE_BYTES, len(h))
	require.Equal(t, ExpectedSha256, h.String(), "result should match")
}

func TestCalcSha256_MultipleChunks(t *testing.T) {
	h := CalcSha256(someData[:3], someData[3:])
	require.Equal(t, SHA256_HASH_SIZE_BYTES, len(h))
	require.Equal(t, ExpectedSha256, h.String(), "result should match")
}

func TestCalcKeccak256(t *testing.T) {
	h := CalcKeccak256(someData)
	require.Equal(t, KECCAK256_HASH_SIZE_BYTES, len(h))
	require.Equal(t, ExpectedKeccak256, h.String(), "result should match")
}

func BenchmarkCalcSha256(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CalcSha256(someData)
	}
}
