package transactionpool

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func expectTransactionsToBeForwarded(gossip *gossiptopics.MockTransactionRelay, publicKey primitives.Ed25519PublicKey, sig primitives.Ed25519Sig, transactions ...*protocol.SignedTransaction) {
	gossip.When("BroadcastForwardedTransactions", &gossiptopics.ForwardedTransactionsInput{
		Message: &gossipmessages.ForwardedTransactionsMessage{
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: publicKey,
				Signature:       sig,
			}).Build(),
			SignedTransactions: transactions,
		},
	}).Return(&gossiptopics.EmptyOutput{}, nil).Times(1)
}

func TestForwardsTransactionAfterTimeout(t *testing.T) {
	t.Parallel()

	test.WithContext(func(ctx context.Context) {
		gossip := &gossiptopics.MockTransactionRelay{}

		// FIXME factory method for specific config
		cfg := config.ForTransactionPoolTests(0, keys.Ed25519KeyPairForTests(0))

		txForwarder := NewTransactionForwarder(ctx, log.GetLogger(), cfg, gossip)

		tx := builders.TransferTransaction().Build()
		anotherTx := builders.TransferTransaction().Build()

		oneBigHash, _ := HashTransactions(tx, anotherTx)
		sig, _ := signature.SignEd25519(cfg.NodePrivateKey(), oneBigHash)

		expectTransactionsToBeForwarded(gossip, cfg.NodePublicKey(), sig, tx, anotherTx)

		txForwarder.enqueueTransactionToBeForwarded(tx)
		txForwarder.enqueueTransactionToBeForwarded(anotherTx)

		require.NoError(t, test.EventuallyVerify(10*time.Millisecond, gossip), "mocks were not called as expected")
	})
}
