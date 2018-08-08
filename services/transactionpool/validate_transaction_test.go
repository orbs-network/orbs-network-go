package transactionpool

import (
	"testing"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/stretchr/testify/require"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"fmt"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"time"
)

const expirationWindowInterval = 30 * time.Minute
const futureTimestampGrace = 3 * time.Minute

var lastCommittedBlockTimestamp = primitives.TimestampNano(time.Now().Add(-5 * time.Second).UnixNano())

func aValidationContextWithTransactionsInPools(transactionsInPendingPool transactions) validationContext {
	isTxInPendingPool := func(tx *protocol.SignedTransaction) bool {
		for _, t := range transactionsInPendingPool {
			if tx.Equal(t) {
				return true
			}
		}
		return false
	}

	return validationContext{
		expiryWindow:                expirationWindowInterval,
		lastCommittedBlockTimestamp: lastCommittedBlockTimestamp,
		futureTimestampGrace:        futureTimestampGrace,
		virtualChainId:              primitives.VirtualChainId(42),
		transactionInPendingPool:    isTxInPendingPool,
	}
}

func aValidationContext() validationContext {
	return aValidationContextWithTransactionsInPools(transactions{})
}

//TODO Verify pre order checks (like signature and subscription) by calling VirtualMachine.TransactionSetPreOrder.

func futureTimeAfterGracePeriod() time.Time {
	return time.Unix(0, int64(lastCommittedBlockTimestamp)).Add(futureTimestampGrace + 1*time.Minute)
}

func TestValidateTransaction_ValidTransaction(t *testing.T) {
	t.Parallel()

	require.NoError(t,
		validateTransaction(builders.TransferTransaction().Build(), aValidationContext()),
		"a valid transaction was rejected")
}

func TestValidateTransaction_InvalidTransactions(t *testing.T) {
	tests := []struct {
		name      string
		txBuilder *builders.TransferTransactionBuilder
	}{
		{"protocol version", builders.TransferTransaction().WithProtocolVersion(ProtocolVersion + 1)},
		{"signer scheme", builders.TransferTransaction().WithInvalidSignerScheme()},
		{"empty signer public key", builders.TransferTransaction().WithSigner(protocol.NETWORK_TYPE_TEST_NET, primitives.Ed25519PublicKey([]byte{}))},
		{"signer public key (wrong length)", builders.TransferTransaction().WithSigner(protocol.NETWORK_TYPE_TEST_NET, keys.Ed25519KeyPairForTests(1).PublicKey()[1:])},
		{"contract name", builders.TransferTransaction().WithContract("")},
		{"timestamp (created prior to the expiry window)", builders.TransferTransaction().WithTimestamp(time.Now().Add(expirationWindowInterval * -2))},
		{"timestamp (created after the grace period for last committed block)", builders.TransferTransaction().WithTimestamp(futureTimeAfterGracePeriod())},
		{"virtual chain id", builders.TransferTransaction().WithVirtualChainId(primitives.VirtualChainId(1))},
	}
	for i := range tests {
		test := tests[i] // this is so that we can run tests in parallel, see https://gist.github.com/posener/92a55c4cd441fc5e5e85f27bca008721
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			require.Error(t,
				validateTransaction(test.txBuilder.Build(), aValidationContext()),
				fmt.Sprintf("a transaction with an invalid %s was not rejected", test.name))
		})
	}
}

func TestValidateTransaction_DoesNotExistInPendingPool(t *testing.T) {
	t.Parallel()
	tx := builders.TransferTransaction().Build()

	require.Error(t,
		validateTransaction(tx, aValidationContextWithTransactionsInPools(transactions{tx})),
		"a transaction that exists in the pending transaction pool was not rejected")
}

