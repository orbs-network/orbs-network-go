// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test/builders"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestResponseForTransactionOnValidContract(t *testing.T) {
	newHarness().Start(t, func(t testing.TB, parent context.Context, network *NetworkHarness) {
		ctx, cancel := context.WithTimeout(parent, 1*time.Second)
		defer cancel()

		tx := builders.TransferTransaction()
		resp, _ := network.SendTransaction(ctx, tx.Builder(), 0)
		require.Equal(t, protocol.REQUEST_STATUS_COMPLETED, resp.RequestResult().RequestStatus())
		require.Equal(t, protocol.TRANSACTION_STATUS_COMMITTED, resp.TransactionStatus())
		require.Equal(t, protocol.EXECUTION_RESULT_SUCCESS, resp.TransactionReceipt().ExecutionResult())
	})
}

func TestResponseForTransactionOnContractNotDeployed(t *testing.T) {
	newHarness().Start(t, func(t testing.TB, parent context.Context, network *NetworkHarness) {
		ctx, cancel := context.WithTimeout(parent, 1*time.Second)
		defer cancel()

		tx := builders.Transaction().WithContract("UnknownContract")
		resp, _ := network.SendTransaction(ctx, tx.Builder(), 0)
		require.Equal(t, protocol.REQUEST_STATUS_BAD_REQUEST, resp.RequestResult().RequestStatus())
		require.Equal(t, protocol.TRANSACTION_STATUS_COMMITTED, resp.TransactionStatus())
		require.Equal(t, protocol.EXECUTION_RESULT_ERROR_CONTRACT_NOT_DEPLOYED, resp.TransactionReceipt().ExecutionResult())
	})
}

func TestResponseForTransactionOnContractWithBadInput(t *testing.T) {
	newHarness().Start(t, func(t testing.TB, parent context.Context, network *NetworkHarness) {
		ctx, cancel := context.WithTimeout(parent, 1*time.Second)
		defer cancel()

		tx := builders.TransferTransaction().WithArgs("bad", "types", "of", "args")
		resp, _ := network.SendTransaction(ctx, tx.Builder(), 0)
		require.Equal(t, protocol.REQUEST_STATUS_BAD_REQUEST, resp.RequestResult().RequestStatus())
		require.Equal(t, protocol.TRANSACTION_STATUS_COMMITTED, resp.TransactionStatus())
		require.Equal(t, protocol.EXECUTION_RESULT_ERROR_INPUT, resp.TransactionReceipt().ExecutionResult())
	})
}

func TestResponseForTransactionOnFailingContract(t *testing.T) {
	newHarness().Start(t, func(t testing.TB, parent context.Context, network *NetworkHarness) {
		ctx, cancel := context.WithTimeout(parent, 1*time.Second)
		defer cancel()

		tx := builders.Transaction().WithMethod(primitives.ContractName("BenchmarkContract"), primitives.MethodName("throw")).WithArgs()
		resp, _ := network.SendTransaction(ctx, tx.Builder(), 0)
		require.Equal(t, protocol.REQUEST_STATUS_COMPLETED, resp.RequestResult().RequestStatus())
		require.Equal(t, protocol.TRANSACTION_STATUS_COMMITTED, resp.TransactionStatus())
		require.Equal(t, protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT, resp.TransactionReceipt().ExecutionResult())
	})
}

func TestResponseForTransactionWithInvalidProtocolVersion(t *testing.T) {
	newHarness().Start(t, func(t testing.TB, parent context.Context, network *NetworkHarness) {
		ctx, cancel := context.WithTimeout(parent, 1*time.Second)
		defer cancel()

		tx := builders.Transaction().WithProtocolVersion(9999999)
		resp, _ := network.SendTransaction(ctx, tx.Builder(), 0)
		require.Equal(t, protocol.REQUEST_STATUS_BAD_REQUEST, resp.RequestResult().RequestStatus())
		require.Equal(t, protocol.TRANSACTION_STATUS_REJECTED_UNSUPPORTED_VERSION, resp.TransactionStatus())
		require.Empty(t, resp.TransactionReceipt().Raw())
	})
}

func TestResponseForTransactionWithBadSignature(t *testing.T) {
	newHarness().
		AllowingErrors("error validating transaction for preorder").
		Start(t, func(t testing.TB, parent context.Context, network *NetworkHarness) {
			ctx, cancel := context.WithTimeout(parent, 1*time.Second)
			defer cancel()

			tx := builders.Transaction().WithInvalidEd25519Signer(testKeys.Ed25519KeyPairForTests(1))
			resp, _ := network.SendTransaction(ctx, tx.Builder(), 0)
			require.Equal(t, protocol.REQUEST_STATUS_BAD_REQUEST, resp.RequestResult().RequestStatus())
			require.Equal(t, protocol.TRANSACTION_STATUS_REJECTED_SIGNATURE_MISMATCH, resp.TransactionStatus())
			require.Empty(t, resp.TransactionReceipt().Raw())
		})
}
