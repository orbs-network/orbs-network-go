package contracts

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"time"
)

type BenchmarkTokenClient interface {
	DeployBenchmarkToken(ctx context.Context, ownerAddressIndex int)
	SendTransfer(ctx context.Context, nodeIndex int, amount uint64, fromAddressIndex int, toAddressIndex int) chan *client.SendTransactionResponse
	SendTransferInBackground(ctx context.Context, nodeIndex int, amount uint64, fromAddressIndex int, toAddressIndex int) primitives.Sha256
	SendInvalidTransfer(ctx context.Context, nodeIndex int, fromAddressIndex int, toAddressIndex int) chan *client.SendTransactionResponse
	CallGetBalance(ctx context.Context, nodeIndex int, forAddressIndex int) chan uint64
}

func (c *contractClient) DeployBenchmarkToken(ctx context.Context, ownerAddressIndex int) {
	tx := <-c.SendTransfer(ctx, 0, 0, ownerAddressIndex, ownerAddressIndex) // deploy BenchmarkToken by running an empty transaction
	for _, api := range c.apis {
		timeoutCtx, _ := context.WithTimeout(ctx, 1*time.Second)
		api.WaitForTransactionInState(timeoutCtx, tx.TransactionReceipt().Txhash())
	}
}

func (c *contractClient) SendTransfer(ctx context.Context, nodeIndex int, amount uint64, fromAddressIndex int, toAddressIndex int) chan *client.SendTransactionResponse {
	tx := builders.TransferTransaction().
		WithEd25519Signer(keys.Ed25519KeyPairForTests(fromAddressIndex)).
		WithAmountAndTargetAddress(amount, builders.AddressForEd25519SignerForTests(toAddressIndex)).
		Builder()

	return c.sendTransaction(ctx, tx, nodeIndex)
}

// TODO: when publicApi supports returning as soon as SendTransaction is in the pool, switch to blocking implementation that waits for this
func (c *contractClient) SendTransferInBackground(ctx context.Context, nodeIndex int, amount uint64, fromAddressIndex int, toAddressIndex int) primitives.Sha256 {
	signerKeyPair := keys.Ed25519KeyPairForTests(fromAddressIndex)
	targetAddress := builders.AddressForEd25519SignerForTests(toAddressIndex)
	request := (&client.SendTransactionRequestBuilder{
		SignedTransaction: builders.TransferTransaction().
			WithEd25519Signer(signerKeyPair).
			WithAmountAndTargetAddress(amount, targetAddress).
			Builder(),
	}).Build()

	supervised.GoOnce(c.logger, func() {
		publicApi := c.apis[nodeIndex].GetPublicApi()
		publicApi.SendTransaction(ctx, &services.SendTransactionInput{ // we ignore timeout here.
			ClientRequest: request,
		})
	})
	return digest.CalcTxHash(request.SignedTransaction().Transaction())
}

func (c *contractClient) SendInvalidTransfer(ctx context.Context, nodeIndex int, fromAddressIndex int, toAddressIndex int) chan *client.SendTransactionResponse {
	signerKeyPair := keys.Ed25519KeyPairForTests(fromAddressIndex)
	targetAddress := builders.AddressForEd25519SignerForTests(toAddressIndex)
	tx := builders.TransferTransaction().WithEd25519Signer(signerKeyPair).WithInvalidAmount(targetAddress).Builder()

	return c.sendTransaction(ctx, tx, nodeIndex)
}

func (c *contractClient) CallGetBalance(ctx context.Context, nodeIndex int, forAddressIndex int) chan uint64 {
	tx := builders.GetBalanceTransaction().
		WithEd25519Signer(keys.Ed25519KeyPairForTests(forAddressIndex)).
		WithTargetAddress(builders.AddressForEd25519SignerForTests(forAddressIndex)).
		Builder().Transaction

	return c.callMethod(ctx, tx, nodeIndex)
}
