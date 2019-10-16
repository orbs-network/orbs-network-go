// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestContractCallBadNodeConfig(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			config := &ethereumConnectorConfigForTests{
				endpoint:      "invalid_endpoint",
				privateKeyHex: "",
			}
			h := newRpcEthereumConnectorHarness(parent.Logger, config)

			input := builders.EthereumCallContractInput().Build() // don't care about specifics

			_, err := h.connector.EthereumCallContract(ctx, input)
			require.Error(t, err, "expected call to fail")
			require.Contains(t, err.Error(), "dial unix invalid_endpoint: connect: no such file or directory", "expected invalid node in config")
		})
	})
}
