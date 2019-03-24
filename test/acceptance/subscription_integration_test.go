// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/acceptance/callcontract"
	"github.com/orbs-network/orbs-network-go/test/contracts/subscription"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSubscriptionCheck_ValidSubscription(t *testing.T) {
	newHarness().
		WithVirtualChainId(42).
		Start(t, func(t testing.TB, ctx context.Context, network *NetworkHarness) {
			sim := network.ethereumConnection
			address, _, err := sim.DeployEthereumContract(sim.GetAuth(), subscription.SubscriptionABI, subscription.SubscriptionBin)
			stringAddress := address.String()

			sim.Commit()

			require.NoError(t, err, "failed deploying subscription contract")

			contract := callcontract.NewContractClient(network)
			res, _ := contract.RefreshSubscription(ctx, 0, stringAddress)
			test.RequireSuccess(t, res, "failed refreshing subscription")

			res, _ = contract.Transfer(ctx, 0, 50, 5, 6)
			test.RequireSuccess(t, res, "failed sending transaction to a virtual chain with a valid subscription")

		})
}

func TestSubscriptionCheck_UnderpaidSubscription(t *testing.T) {
	newHarness().
		WithVirtualChainId(17).
		AllowingErrors("error validating transaction for preorder").
		Start(t, func(t testing.TB, ctx context.Context, network *NetworkHarness) {
			sim := network.ethereumConnection
			address, _, err := sim.DeployEthereumContract(sim.GetAuth(), subscription.SubscriptionABI, subscription.SubscriptionBin)
			stringAddress := address.String()

			sim.Commit()

			require.NoError(t, err, "failed deploying subscription contract")

			contract := callcontract.NewContractClient(network)
			res, _ := contract.RefreshSubscription(ctx, 0, stringAddress)
			test.RequireSuccess(t, res, "failed refreshing subscription")

			res, _ = contract.Transfer(ctx, 0, 50, 5, 6)
			require.EqualValues(t, protocol.TRANSACTION_STATUS_REJECTED_GLOBAL_PRE_ORDER.String(), res.TransactionStatus().String(), "expected transaction to fail due to invalid subscription")

		})
}
