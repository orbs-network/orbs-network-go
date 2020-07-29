// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"testing"
	"time"
)

func TestDeployContractTransactionToNode(t *testing.T) {
	NewHarness().
		WithConfigOverride(config.NodeConfigKeyValue{Key: config.COMMITTEE_GRACE_PERIOD, Value: config.NodeConfigValue{DurationValue: 12 * time.Hour}}).
		Start(t, func(t testing.TB, ctx context.Context, network *Network) {
			network.DeployBenchmarkTokenContract(ctx, 1)
		})
}
