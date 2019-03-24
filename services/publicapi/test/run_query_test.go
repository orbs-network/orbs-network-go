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
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestRunQuery_CallsVirtualMachine(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newPublicApiHarness(ctx, t, time.Millisecond, time.Minute)

		harness.runTransactionSuccess()

		result, err := harness.papi.RunQuery(ctx, &services.RunQueryInput{
			ClientRequest: (&client.RunQueryRequestBuilder{
				SignedQuery: builders.Query().Builder(),
			}).Build(),
		})

		harness.verifyMocks(t) // contract test

		require.Equal(t, protocol.EXECUTION_RESULT_SUCCESS, result.ClientResponse.QueryResult().ExecutionResult(), "got wrong status")
		require.NoError(t, err, "error happened when it should not")
	})
}
