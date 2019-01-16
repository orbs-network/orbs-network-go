package callcontract

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"time"
)

type BenchmarkTokenClient interface {
	DeployBenchmarkToken(ctx context.Context, ownerAddressIndex int)
	Transfer(ctx context.Context, nodeIndex int, amount uint64, fromAddressIndex int, toAddressIndex int) (*client.SendTransactionResponse, primitives.Sha256)
	TransferInBackground(ctx context.Context, nodeIndex int, amount uint64, fromAddressIndex int, toAddressIndex int) primitives.Sha256
	InvalidTransfer(ctx context.Context, nodeIndex int, fromAddressIndex int, toAddressIndex int) *client.SendTransactionResponse
	GetBalance(ctx context.Context, nodeIndex int, forAddressIndex int) uint64
}

func (c *contractClient) DeployBenchmarkToken(ctx context.Context, ownerAddressIndex int) {
	txHash := c.TransferInBackground(ctx, 0, 0, ownerAddressIndex, ownerAddressIndex) // deploy BenchmarkToken by running an empty transaction
	timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	response := c.API.GetTransactionStatus(timeoutCtx, txHash, 0)
	for response.TransactionStatus() != protocol.TRANSACTION_STATUS_COMMITTED {
		response = c.API.GetTransactionStatus(timeoutCtx, txHash, 0)
	}
}

func (c *contractClient) Transfer(ctx context.Context, nodeIndex int, amount uint64, fromAddressIndex int, toAddressIndex int) (*client.SendTransactionResponse, primitives.Sha256) {
	tx := builders.TransferTransaction().
		WithEd25519Signer(keys.Ed25519KeyPairForTests(fromAddressIndex)).
		WithAmountAndTargetAddress(amount, builders.ClientAddressForEd25519SignerForTests(toAddressIndex)).
		Builder()

	return c.API.SendTransaction(ctx, tx, nodeIndex)
}

// TODO(https://github.com/orbs-network/orbs-network-go/issues/434): when publicApi supports returning as soon as SendTransaction is in the pool, switch to blocking implementation that waits for this
func (c *contractClient) TransferInBackground(ctx context.Context, nodeIndex int, amount uint64, fromAddressIndex int, toAddressIndex int) primitives.Sha256 {
	signerKeyPair := keys.Ed25519KeyPairForTests(fromAddressIndex)
	targetAddress := builders.ClientAddressForEd25519SignerForTests(toAddressIndex)
	tx := builders.TransferTransaction().
		WithEd25519Signer(signerKeyPair).
		WithAmountAndTargetAddress(amount, targetAddress).
		Builder()
	builtTx := tx.Build()

	txHash := digest.CalcTxHash(builtTx.Transaction()) // transaction may not be thread safe, pre-calculate txHash
	c.API.SendTransactionInBackground(ctx, tx, nodeIndex)

	return txHash
}

func (c *contractClient) InvalidTransfer(ctx context.Context, nodeIndex int, fromAddressIndex int, toAddressIndex int) *client.SendTransactionResponse {
	signerKeyPair := keys.Ed25519KeyPairForTests(fromAddressIndex)
	targetAddress := builders.ClientAddressForEd25519SignerForTests(toAddressIndex)
	tx := builders.TransferTransaction().WithEd25519Signer(signerKeyPair).WithInvalidAmount(targetAddress).Builder()

	out, _ := c.API.SendTransaction(ctx, tx, nodeIndex)
	return out
}

func (c *contractClient) GetBalance(ctx context.Context, nodeIndex int, forAddressIndex int) uint64 {
	query := builders.GetBalanceQuery().
		WithEd25519Signer(keys.Ed25519KeyPairForTests(forAddressIndex)).
		WithTargetAddress(builders.ClientAddressForEd25519SignerForTests(forAddressIndex)).
		Builder()

	out := c.API.RunQuery(ctx, query, nodeIndex)
	argsArray := builders.PackedArgumentArrayDecode(out.QueryResult().RawOutputArgumentArrayWithHeader())
	return argsArray.ArgumentsIterator().NextArguments().Uint64Value()
}
