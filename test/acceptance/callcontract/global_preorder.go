package callcontract

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
)

type GlobalPreOrderClient interface {
	UnsafeTests_SetSubscriptionOk(ctx context.Context, nodeIndex int) (*client.SendTransactionResponse, primitives.Sha256)
	UnsafeTests_SetSubscriptionProblem(ctx context.Context, nodeIndex int) (*client.SendTransactionResponse, primitives.Sha256)
}

func (c *contractClient) UnsafeTests_SetSubscriptionOk(ctx context.Context, nodeIndex int) (*client.SendTransactionResponse, primitives.Sha256) {
	tx := builders.Transaction().
		WithMethod("_GlobalPreOrder", "unsafetests_setSubscriptionOk").
		WithArgs().
		WithEd25519Signer(keys.Ed25519KeyPairForTests(0)).
		Builder()

	return c.API.SendTransaction(ctx, tx, nodeIndex)
}

func (c *contractClient) UnsafeTests_SetSubscriptionProblem(ctx context.Context, nodeIndex int) (*client.SendTransactionResponse, primitives.Sha256) {
	tx := builders.Transaction().
		WithMethod("_GlobalPreOrder", "unsafetests_setSubscriptionProblem").
		WithArgs().
		WithEd25519Signer(keys.Ed25519KeyPairForTests(0)).
		Builder()

	return c.API.SendTransaction(ctx, tx, nodeIndex)
}

func (c *contractClient) RefreshSubscription(ctx context.Context, nodeIndex int, ethContractAddress string) (*client.SendTransactionResponse, primitives.Sha256) {
	tx := builders.Transaction().
		WithMethod("_GlobalPreOrder", "refreshSubscription").
		WithArgs(ethContractAddress).
		WithEd25519Signer(keys.Ed25519KeyPairForTests(0)).
		Builder()

	return c.API.SendTransaction(ctx, tx, nodeIndex)
}
