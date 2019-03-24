// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package callcontract

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
)

type BenchmarkTokenClient interface {
	Transfer(ctx context.Context, nodeIndex int, amount uint64, fromAddressIndex int, toAddressIndex int) (*client.SendTransactionResponse, primitives.Sha256)
	TransferInBackground(ctx context.Context, nodeIndex int, amount uint64, fromAddressIndex int, toAddressIndex int) primitives.Sha256
	InvalidTransfer(ctx context.Context, nodeIndex int, fromAddressIndex int, toAddressIndex int) *client.SendTransactionResponse
	GetBalance(ctx context.Context, nodeIndex int, forAddressIndex int) uint64
}

func (c *contractClient) ATransferTransaction() *builders.TransactionBuilder {
	return builders.TransferTransaction().WithVirtualChainId(c.API.GetVirtualChainId())
}

func (c *contractClient) Transfer(ctx context.Context, nodeIndex int, amount uint64, fromAddressIndex int, toAddressIndex int) (*client.SendTransactionResponse, primitives.Sha256) {
	tx := c.ATransferTransaction().
		WithEd25519Signer(keys.Ed25519KeyPairForTests(fromAddressIndex)).
		WithAmountAndTargetAddress(amount, builders.ClientAddressForEd25519SignerForTests(toAddressIndex)).
		Builder()

	return c.API.SendTransaction(ctx, tx, nodeIndex)
}

// TODO(https://github.com/orbs-network/orbs-network-go/issues/434): when publicApi supports returning as soon as SendTransaction is in the pool, switch to blocking implementation that waits for this
func (c *contractClient) TransferInBackground(ctx context.Context, nodeIndex int, amount uint64, fromAddressIndex int, toAddressIndex int) primitives.Sha256 {
	signerKeyPair := keys.Ed25519KeyPairForTests(fromAddressIndex)
	targetAddress := builders.ClientAddressForEd25519SignerForTests(toAddressIndex)
	tx := c.ATransferTransaction().
		WithEd25519Signer(signerKeyPair).
		WithAmountAndTargetAddress(amount, targetAddress).
		WithVirtualChainId(c.API.GetVirtualChainId()).
		Builder()
	builtTx := tx.Build()

	txHash := digest.CalcTxHash(builtTx.Transaction()) // transaction may not be thread safe, pre-calculate txHash
	c.API.SendTransactionInBackground(ctx, tx, nodeIndex)

	return txHash
}

func (c *contractClient) InvalidTransfer(ctx context.Context, nodeIndex int, fromAddressIndex int, toAddressIndex int) *client.SendTransactionResponse {
	signerKeyPair := keys.Ed25519KeyPairForTests(fromAddressIndex)
	targetAddress := builders.ClientAddressForEd25519SignerForTests(toAddressIndex)
	tx := c.ATransferTransaction().WithEd25519Signer(signerKeyPair).WithInvalidAmount(targetAddress).Builder()

	out, _ := c.API.SendTransaction(ctx, tx, nodeIndex)
	return out
}

func (c *contractClient) GetBalance(ctx context.Context, nodeIndex int, forAddressIndex int) uint64 {
	query := builders.GetBalanceQuery().
		WithVirtualChainId(c.API.GetVirtualChainId()).
		WithEd25519Signer(keys.Ed25519KeyPairForTests(forAddressIndex)).
		WithTargetAddress(builders.ClientAddressForEd25519SignerForTests(forAddressIndex)).
		Builder()

	out := c.API.RunQuery(ctx, query, nodeIndex)
	if out.RequestResult().RequestStatus() != protocol.REQUEST_STATUS_COMPLETED || out.QueryResult().ExecutionResult() != protocol.EXECUTION_RESULT_SUCCESS {
		panic(fmt.Sprintf("query failed; nested error is %s", out.String()))
	}
	argsArray := builders.PackedArgumentArrayDecode(out.QueryResult().RawOutputArgumentArrayWithHeader())
	arguments := argsArray.ArgumentsIterator().NextArguments()
	if !arguments.IsTypeUint64Value() {
		panic(fmt.Sprintf("expected exactly one output argument of type uint64 but found %s, in %s", arguments.String(), out.String()))
	}
	return arguments.Uint64Value()
}
