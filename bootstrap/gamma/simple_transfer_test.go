// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package gamma

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/acceptance/callcontract"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/scribe/log"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func testSimpleTransfer(t *testing.T, jsonConfig string) {
	test.WithContext(func(ctx context.Context) {
		network := NewDevelopmentNetwork(ctx, log.DefaultTestingLogger(t), nil, jsonConfig)
		contract := callcontract.NewContractClient(network)

		t.Log("doing a simple transfer")

		contract.Transfer(ctx, 0, 17, 5, 6)

		t.Log("making sure balance is correct")

		require.True(t, test.Eventually(1*time.Second, func() bool {
			return 17 == contract.GetBalance(ctx, 0, 6)
		}), "expected balance to reflect the transfer")

	})
	time.Sleep(5 * time.Millisecond) // give context dependent goroutines 5 ms to terminate gracefully
}

func TestSimpleTransfer_Benchmark(t *testing.T) {
	testSimpleTransfer(t, "")
}

func TestSimpleTransfer_LeanHelix(t *testing.T) {
	testSimpleTransfer(t, fmt.Sprintf(`{"active-consensus-algo":%d}`, consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX))
}
