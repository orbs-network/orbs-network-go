// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package transactionpool

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
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

func (c *forwarderConfig) NodePrivateKey() primitives.EcdsaSecp256K1PrivateKey {
	return c.keyPair.PrivateKey()
}

func (c *forwarderConfig) TransactionPoolPropagationBatchSize() uint16 {
	return c.queueSize
}

func (c *forwarderConfig) TransactionPoolPropagationBatchingTimeout() time.Duration {
	return 5 * time.Millisecond
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

	test.WithContext(func(ctx context.Context) {
		gossip := &gossiptopics.MockTransactionRelay{}
		cfg := &forwarderConfig{3, testKeys.EcdsaSecp256K1KeyPairForTests(0)}

		txForwarder := NewTransactionForwarder(ctx, log.DefaultTestingLogger(t), cfg, gossip)

		tx := builders.TransferTransaction().Build()
		anotherTx := builders.TransferTransaction().Build()

		oneBigHash, _, _ := HashTransactions(tx, anotherTx)
		sig, _ := digest.SignAsNode(cfg.NodePrivateKey(), oneBigHash)

		expectTransactionsToBeForwarded(gossip, cfg.NodeAddress(), sig, tx, anotherTx)

		txForwarder.submit(tx)
		txForwarder.submit(anotherTx)

		require.NoError(t, test.EventuallyVerify(cfg.TransactionPoolPropagationBatchingTimeout()*2, gossip), "mocks were not called as expected")
	})
}

func TestForwardsTransactionAfterLimitWasReached(t *testing.T) {

	test.WithContext(func(ctx context.Context) {
		gossip := &gossiptopics.MockTransactionRelay{}
		cfg := &forwarderConfig{2, testKeys.EcdsaSecp256K1KeyPairForTests(0)}

		txForwarder := NewTransactionForwarder(ctx, log.DefaultTestingLogger(t), cfg, gossip)

		tx := builders.TransferTransaction().Build()
		anotherTx := builders.TransferTransaction().Build()

		oneBigHash, _, _ := HashTransactions(tx, anotherTx)
		sig, _ := digest.SignAsNode(cfg.NodePrivateKey(), oneBigHash)

		expectTransactionsToBeForwarded(gossip, cfg.NodeAddress(), sig, tx, anotherTx)

		txForwarder.submit(tx)
		txForwarder.submit(anotherTx)

		require.NoError(t, test.EventuallyVerify(10*time.Millisecond, gossip), "mocks were not called as expected")
	})
}
