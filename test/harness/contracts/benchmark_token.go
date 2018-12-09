package contracts

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"time"
)

type BenchmarkTokenClient interface {
	DeployBenchmarkToken(ctx context.Context, ownerAddressIndex int)
	SendTransfer(ctx context.Context, nodeIndex int, amount uint64, fromAddressIndex int, toAddressIndex int) (*client.SendTransactionResponse, primitives.Sha256)
	SendTransferInBackground(ctx context.Context, nodeIndex int, amount uint64, fromAddressIndex int, toAddressIndex int) primitives.Sha256
	SendInvalidTransfer(ctx context.Context, nodeIndex int, fromAddressIndex int, toAddressIndex int) *client.SendTransactionResponse
	CallGetBalance(ctx context.Context, nodeIndex int, forAddressIndex int) uint64
}

func (c *contractClient) DeployBenchmarkToken(ctx context.Context, ownerAddressIndex int) {
	txHash := c.SendTransferInBackground(ctx, 0, 0, ownerAddressIndex, ownerAddressIndex) // deploy BenchmarkToken by running an empty transaction
	timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	c.API.WaitForTransactionInState(timeoutCtx, txHash)
}

func (c *contractClient) SendTransfer(ctx context.Context, nodeIndex int, amount uint64, fromAddressIndex int, toAddressIndex int) (*client.SendTransactionResponse, primitives.Sha256) {
	tx := builders.TransferTransaction().
		WithEd25519Signer(keys.Ed25519KeyPairForTests(fromAddressIndex)).
		WithAmountAndTargetAddress(amount, builders.AddressForEd25519SignerForTests(toAddressIndex)).
		Builder()

	return c.API.SendTransaction(ctx, tx, nodeIndex)
}

// TODO(https://github.com/orbs-network/orbs-network-go/issues/434): when publicApi supports returning as soon as SendTransaction is in the pool, switch to blocking implementation that waits for this
func (c *contractClient) SendTransferInBackground(ctx context.Context, nodeIndex int, amount uint64, fromAddressIndex int, toAddressIndex int) primitives.Sha256 {
	signerKeyPair := keys.Ed25519KeyPairForTests(fromAddressIndex)
	targetAddress := builders.AddressForEd25519SignerForTests(toAddressIndex)
	tx := builders.TransferTransaction().
		WithEd25519Signer(signerKeyPair).
		WithAmountAndTargetAddress(amount, targetAddress).
		Builder()
	builtTx := tx.Build()

	txHash := digest.CalcTxHash(builtTx.Transaction()) // transaction may not be thread safe, pre-calculate txHash
	c.API.SendTransactionInBackground(ctx, tx, nodeIndex)

	return txHash
}

func (c *contractClient) SendInvalidTransfer(ctx context.Context, nodeIndex int, fromAddressIndex int, toAddressIndex int) *client.SendTransactionResponse {
	signerKeyPair := keys.Ed25519KeyPairForTests(fromAddressIndex)
	targetAddress := builders.AddressForEd25519SignerForTests(toAddressIndex)
	tx := builders.TransferTransaction().WithEd25519Signer(signerKeyPair).WithInvalidAmount(targetAddress).Builder()

	out, _ := c.API.SendTransaction(ctx, tx, nodeIndex)
	return out
}

func (c *contractClient) CallGetBalance(ctx context.Context, nodeIndex int, forAddressIndex int) uint64 {
	tx := builders.GetBalanceTransaction().
		WithEd25519Signer(keys.Ed25519KeyPairForTests(forAddressIndex)).
		WithTargetAddress(builders.AddressForEd25519SignerForTests(forAddressIndex)).
		Builder().Transaction

	r := c.API.CallMethod(ctx, tx, nodeIndex)
	outputArgsIterator := builders.ClientCallMethodResponseOutputArgumentsDecode(r)
	return outputArgsIterator.NextArguments().Uint64Value()
}
