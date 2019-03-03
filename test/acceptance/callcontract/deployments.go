package callcontract

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
)

type DeploymentsClient interface {
	LockNativeDeployment(ctx context.Context, nodeIndex int, toAddressIndex int) (*client.SendTransactionResponse, primitives.Sha256)
	UnlockNativeDeployment(ctx context.Context, nodeIndex int, fromAddressIndex int) (*client.SendTransactionResponse, primitives.Sha256)
}

func (c *contractClient) LockNativeDeployment(ctx context.Context, nodeIndex int, toAddressIndex int) (*client.SendTransactionResponse, primitives.Sha256) {
	tx := builders.Transaction().
		WithMethod("_Deployments", "lockNativeDeployment").
		WithArgs().
		WithEd25519Signer(keys.Ed25519KeyPairForTests(toAddressIndex)).
		Builder()

	return c.API.SendTransaction(ctx, tx, nodeIndex)
}

func (c *contractClient) UnlockNativeDeployment(ctx context.Context, nodeIndex int, fromAddressIndex int) (*client.SendTransactionResponse, primitives.Sha256) {
	tx := builders.Transaction().
		WithMethod("_Deployments", "unlockNativeDeployment").
		WithArgs().
		WithEd25519Signer(keys.Ed25519KeyPairForTests(fromAddressIndex)).
		Builder()

	return c.API.SendTransaction(ctx, tx, nodeIndex)
}
