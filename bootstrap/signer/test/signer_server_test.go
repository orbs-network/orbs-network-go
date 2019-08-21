// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/bootstrap/signer"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	crypto "github.com/orbs-network/orbs-network-go/crypto/signer"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/scribe/log"
	"github.com/stretchr/testify/require"
	"testing"
)

type signerServerConfig struct {
	privateKey primitives.EcdsaSecp256K1PrivateKey
	address    string
}

func (s *signerServerConfig) NodePrivateKey() primitives.EcdsaSecp256K1PrivateKey {
	return s.privateKey
}

func (s *signerServerConfig) HttpAddress() string {
	return s.address
}

func TestSignerServer(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		address := "localhost:9999"
		pk := keys.EcdsaSecp256K1KeyPairForTests(0).PrivateKey()

		testOutput := log.NewTestOutput(t, log.NewHumanReadableFormatter())
		testLogger := log.GetLogger().WithOutput(testOutput)

		server := signer.StartSignerServer(&signerServerConfig{pk, address}, testLogger)
		defer server.GracefulShutdown(ctx)

		c := crypto.NewSignerClient("http://" + address)

		ctx = trace.NewContext(ctx, "test")

		payload := []byte("payload")

		signed, err := digest.SignAsNode(pk, payload)
		require.NoError(t, err)

		clientSignature, err := c.Sign(ctx, payload)
		require.NoError(t, err)

		require.EqualValues(t, signed, clientSignature)
	})
}
