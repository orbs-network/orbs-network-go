// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package digest

import (
	"encoding/hex"
	"github.com/stretchr/testify/require"
	"testing"
)

const (
	ExampleNodePrivateKey = "901a1a0bfbe217593062a054e561e708707cb814a123474c25fd567a0fe088f8"
	DifferentNodeAddress  = "bb51846fd3b4110d6818c55a9bdfe9e657b33300"
)

var ExampleDataToSign = []byte("this is what we want to sign")

func TestVerifyNodeSignature(t *testing.T) {
	privateKey, _ := hex.DecodeString(ExampleNodePrivateKey)
	nodeAddress, _ := hex.DecodeString(ExpectedNodeAddress)

	sig, err := SignAsNode(privateKey, ExampleDataToSign)
	require.NoError(t, err)

	err = VerifyNodeSignature(nodeAddress, ExampleDataToSign, sig)
	require.NoError(t, err, "verification should succeed")
}

func TestVerifyNodeSignature_InvalidAddress(t *testing.T) {
	privateKey, _ := hex.DecodeString(ExampleNodePrivateKey)
	differentNodeAddress, _ := hex.DecodeString(DifferentNodeAddress)

	sig, err := SignAsNode(privateKey, ExampleDataToSign)
	require.NoError(t, err)

	err = VerifyNodeSignature(differentNodeAddress, ExampleDataToSign, sig)
	require.Error(t, err, "verification should fail")
}
