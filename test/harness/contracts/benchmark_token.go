package contracts

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/synchronization/supervized"
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
		api.WaitForTransactionInStateForAtMost(ctx, tx.TransactionReceipt().Txhash(), 1 * time.Second)
	}
}

func (c *contractClient) SendTransfer(ctx context.Context, nodeIndex int, amount uint64, fromAddressIndex int, toAddressIndex int) chan *client.SendTransactionResponse {
	signerKeyPair := keys.Ed25519KeyPairForTests(fromAddressIndex)
	targetAddress := builders.AddressForEd25519SignerForTests(toAddressIndex)
	request := (&client.SendTransactionRequestBuilder{
		SignedTransaction: builders.TransferTransaction().WithEd25519Signer(signerKeyPair).WithAmountAndTargetAddress(amount, targetAddress).Builder(),
	}).Build()

	ch := make(chan *client.SendTransactionResponse)
	supervized.ShortLived(c.logger, func() {
		publicApi := c.apis[nodeIndex].GetPublicApi()
		output, err := publicApi.SendTransaction(ctx, &services.SendTransactionInput{
			ClientRequest: request,
		})
		if err != nil {
			panic(fmt.Sprintf("error in transfer: %v", err)) // TODO: improve
		}
		ch <- output.ClientResponse

	})
	return ch
}

// TODO: when publicApi supports returning as soon as SendTransaction is in the pool, switch to blocking implementation that waits for this
func (c *contractClient) SendTransferInBackground(ctx context.Context, nodeIndex int, amount uint64, fromAddressIndex int, toAddressIndex int) primitives.Sha256 {
	signerKeyPair := keys.Ed25519KeyPairForTests(fromAddressIndex)
	targetAddress := builders.AddressForEd25519SignerForTests(toAddressIndex)
	request := (&client.SendTransactionRequestBuilder{
		SignedTransaction: builders.TransferTransaction().WithEd25519Signer(signerKeyPair).WithAmountAndTargetAddress(amount, targetAddress).Builder(),
	}).Build()

	supervized.ShortLived(c.logger, func() {
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
	request := (&client.SendTransactionRequestBuilder{
		SignedTransaction: builders.TransferTransaction().WithEd25519Signer(signerKeyPair).WithInvalidAmount(targetAddress).Builder(),
	}).Build()

	ch := make(chan *client.SendTransactionResponse)
	supervized.ShortLived(c.logger, func() {
		publicApi := c.apis[nodeIndex].GetPublicApi()
		output, err := publicApi.SendTransaction(ctx, &services.SendTransactionInput{
			ClientRequest: request,
		})
		if err != nil {
			panic(fmt.Sprintf("error in invalid transfer: %v", err)) // TODO: improve
		}
		ch <- output.ClientResponse
	})
	return ch
}

func (c *contractClient) CallGetBalance(ctx context.Context, nodeIndex int, forAddressIndex int) chan uint64 {
	signerKeyPair := keys.Ed25519KeyPairForTests(forAddressIndex)
	targetAddress := builders.AddressForEd25519SignerForTests(forAddressIndex)
	request := (&client.CallMethodRequestBuilder{
		Transaction: builders.GetBalanceTransaction().WithEd25519Signer(signerKeyPair).WithTargetAddress(targetAddress).Builder().Transaction,
	}).Build()

	ch := make(chan uint64)
	supervized.ShortLived(c.logger, func() {
		publicApi := c.apis[nodeIndex].GetPublicApi()
		output, err := publicApi.CallMethod(ctx, &services.CallMethodInput{
			ClientRequest: request,
		})
		if err != nil {
			panic(fmt.Sprintf("error in get balance: %v", err)) // TODO: improve
		}
		outputArgsIterator := builders.ClientCallMethodResponseOutputArgumentsDecode(output.ClientResponse)
		ch <- outputArgsIterator.NextArguments().Uint64Value()
	})
	return ch
}