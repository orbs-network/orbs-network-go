// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestEthereumGetBlockNumber(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newSimulatedEthereumConnectorHarness(t).WithFakeTimeGetter()
		in := &services.EthereumGetBlockNumberInput{
			ReferenceTimestamp: primitives.TimestampNano(1505735343000000000), // should return block number 938874
		}
		o, err := h.connector.EthereumGetBlockNumber(ctx, in)
		require.NoError(t, err, "failed getting block number from timestamp")
		require.EqualValues(t, 938874, o.EthereumBlockNumber, "block number on fake data mismatch")
	})
}
