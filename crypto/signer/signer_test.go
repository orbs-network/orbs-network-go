// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package signer

import (
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLocalSigner(t *testing.T) {
	pk := keys.EcdsaSecp256K1KeyPairForTests(0).PrivateKey()
	c := NewLocalSigner(pk)

	payload := []byte("payload")

	signed, err := digest.SignAsNode(pk, payload)
	require.NoError(t, err)

	clientSignature, err := c.Sign(payload)
	require.NoError(t, err)

	require.EqualValues(t, signed, clientSignature)
}
