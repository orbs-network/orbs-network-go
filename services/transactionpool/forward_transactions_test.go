// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package transactionpool

import (
	"context"
	"fmt"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/crypto/signer"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type forwarderConfig struct {
	queueSize uint16
	keyPair   *testKeys.TestEcdsaSecp256K1KeyPair
}

func (c *forwarderConfig) NodeAddress() primitives.NodeAddress {
	return c.keyPair.NodeAddress()
}

func (c *forwarderConfig) TransactionPoolPropagationBatchSize() uint16 {
	return c.queueSize
}

func (c *forwarderConfig) TransactionPoolPropagationBatchingTimeout() time.Duration {
	return 5 * time.Millisecond
}

type signerConfig struct {
	keyPair *testKeys.TestEcdsaSecp256K1KeyPair
}

func (c *signerConfig) NodePrivateKey() primitives.EcdsaSecp256K1PrivateKey {
	return c.keyPair.PrivateKey()
}

func (c *signerConfig) SignerEndpoint() string {
	return ""
}

func expectTransactionsToBeForwarded(gossip *gossiptopics.MockTransactionRelay, nodeAddress primitives.NodeAddress, sig primitives.EcdsaSecp256K1Sig, transactions ...*protocol.SignedTransaction) {
	gossip.When("BroadcastForwardedTransactions", mock.Any, &gossiptopics.ForwardedTransactionsInput{
		Message: &gossipmessages.ForwardedTransactionsMessage{
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderNodeAddress: nodeAddress,
				Signature:         sig,
			}).Build(),
			SignedTransactions: transactions,
		},
	}).Return(&gossiptopics.EmptyOutput{}, nil).Times(1)
}

func TestForwardsTransactionAfterTimeout(t *testing.T) {

	test.WithConcurrencyHarness(t, func(ctx context.Context, harness *test.ConcurrencyHarness) {
		gossip := &gossiptopics.MockTransactionRelay{}
		keyPair := testKeys.EcdsaSecp256K1KeyPairForTests(0)
		cfg := &forwarderConfig{2, keyPair}
		signer, err := signer.New(&signerConfig{keyPair})
		require.NoError(t, err)

		txForwarder := NewTransactionForwarder(ctx, harness.Logger, signer, cfg, gossip)
		harness.Supervise(txForwarder)

		tx := builders.TransferTransaction().Build()
		anotherTx := builders.TransferTransaction().Build()

		oneBigHash, _, _ := HashTransactions(tx, anotherTx)
		sig, _ := signer.Sign(ctx, oneBigHash)

		expectTransactionsToBeForwarded(gossip, cfg.NodeAddress(), sig, tx, anotherTx)

		txForwarder.submit(tx)
		txForwarder.submit(anotherTx)

		require.NoError(t, test.EventuallyVerify(cfg.TransactionPoolPropagationBatchingTimeout()*2, gossip), "mocks were not called as expected")
	})
}

func TestForwardsTransactionAfterLimitWasReached(t *testing.T) {
	test.WithConcurrencyHarness(t, func(ctx context.Context, harness *test.ConcurrencyHarness) {
		gossip := &gossiptopics.MockTransactionRelay{}
		keyPair := testKeys.EcdsaSecp256K1KeyPairForTests(0)
		cfg := &forwarderConfig{2, keyPair}
		signer, err := signer.New(&signerConfig{keyPair})
		require.NoError(t, err)

		txForwarder := NewTransactionForwarder(ctx, harness.Logger, signer, cfg, gossip)
		harness.Supervise(txForwarder)

		tx := builders.TransferTransaction().Build()
		anotherTx := builders.TransferTransaction().Build()

		oneBigHash, _, _ := HashTransactions(tx, anotherTx)
		sig, _ := signer.Sign(ctx, oneBigHash)

		expectTransactionsToBeForwarded(gossip, cfg.NodeAddress(), sig, tx, anotherTx)

		txForwarder.submit(tx)
		txForwarder.submit(anotherTx)

		require.NoError(t, test.EventuallyVerify(10*time.Millisecond, gossip), "mocks were not called as expected")
	})
}

func TestForwardsTransactionWithFaultySigner(t *testing.T) {
	test.WithConcurrencyHarness(t, func(ctx context.Context, harness *test.ConcurrencyHarness) {
		harness.AllowErrorsMatching("error signing transactions")
		gossip := &gossiptopics.MockTransactionRelay{}
		keyPair := testKeys.EcdsaSecp256K1KeyPairForTests(0)
		cfg := &forwarderConfig{2, keyPair}

		signer := &FaultySigner{}
		signer.When("Sign", mock.Any, mock.Any).Return([]byte{}, fmt.Errorf("signer unavailable"))

		txForwarder := NewTransactionForwarder(ctx, harness.Logger, signer, cfg, gossip)
		harness.Supervise(txForwarder)

		tx := builders.TransferTransaction().Build()

		gossip.When("BroadcastForwardedTransactions", mock.Any, mock.Any).Return(nil, nil).Times(0)

		txForwarder.submit(tx)

		require.NoError(t, test.ConsistentlyVerify(cfg.TransactionPoolPropagationBatchingTimeout()*2, gossip), "mocks were not called as expected")

		oneBigHash, _, _ := HashTransactions(tx)
		sig, _ := signer.Sign(ctx, oneBigHash)
		expectTransactionsToBeForwarded(gossip, cfg.NodeAddress(), sig, tx)

		signer.Reset()
		signer.When("Sign", mock.Any, mock.Any).Return(sig, nil).Times(1)

		txForwarder.drainQueueAndForward(ctx)

		require.NoError(t, test.EventuallyVerify(cfg.TransactionPoolPropagationBatchingTimeout()*3, gossip), "mocks were not called as expected")
	})
}

type FaultySigner struct {
	mock.Mock
}

func (c *FaultySigner) Sign(ctx context.Context, input []byte) ([]byte, error) {
	call := c.Called(ctx, input)
	return call.Get(0).([]byte), call.Error(1)
}
