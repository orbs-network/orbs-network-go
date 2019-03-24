// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package digest

import (
	"encoding/hex"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
)

const (
	ExampleNodePublicKey = "30fccea741dd34c7afb146a543616bcb361247148f0c8542541c01da6d6cadf186515f1d851978fc94a6a641e25dec74a6ec28c5ae04c651a0dc2e6104b3ac24"
	ExpectedNodeAddress  = "a328846cd5b4979d68a8c58a9bdfeee657b34de7"
)

func TestCalcNodeAddressFromPublicKey(t *testing.T) {
	publicKey, _ := hex.DecodeString(ExampleNodePublicKey)
	nodeAddress := CalcNodeAddressFromPublicKey(primitives.EcdsaSecp256K1PublicKey(publicKey))
	require.Equal(t, ExpectedNodeAddress, nodeAddress.String(), "result should match")
}
