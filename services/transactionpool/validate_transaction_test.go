package transactionpool

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

const expirationWindowInterval = 30 * time.Minute
const futureTimestampGrace = 3 * time.Minute

var lastCommittedBlockTimestamp = primitives.TimestampNano(time.Now().Add(-15 * time.Minute).UnixNano())

func aValidationContext() *validationContext {
	return &validationContext{
		expiryWindow:                expirationWindowInterval,
		lastCommittedBlockTimestamp: lastCommittedBlockTimestamp,
		futureTimestampGrace:        futureTimestampGrace,
		virtualChainId:              builders.DEFAULT_TEST_VIRTUAL_CHAIN_ID,
	}
}

func futureTimeAfterGracePeriod() time.Time {
	return time.Unix(0, int64(lastCommittedBlockTimestamp)).Add(futureTimestampGrace + 1*time.Minute)
}

func TestValidateTransaction_ValidTransaction(t *testing.T) {
	t.Parallel()

	err := aValidationContext().validateTransaction(aTransactionAtNodeTimestamp().Build())
	require.Nil(t, err, "a valid transaction was rejected")
}

func TestValidateTransaction_RejectsTransactionsWhenTimestampIsZero(t *testing.T) {
	vctx := &validationContext{
		expiryWindow:                expirationWindowInterval,
		lastCommittedBlockTimestamp: 0,
		futureTimestampGrace:        futureTimestampGrace,
		virtualChainId:              builders.DEFAULT_TEST_VIRTUAL_CHAIN_ID,
	}

	err := vctx.validateTransaction(builders.TransferTransaction().Build())
	require.Error(t, err,"a transaction was not rejected when the system is in zero timestamp")
}

//TODO talk to TalKol about Invalid Signer
func TestValidateTransaction_InvalidTransactions(t *testing.T) {
	tests := []struct {
		name           string
		txBuilder      *builders.TransactionBuilder
		expectedStatus protocol.TransactionStatus
	}{
		{"protocol version", aTransactionAtNodeTimestamp().WithProtocolVersion(ProtocolVersion + 1), protocol.TRANSACTION_STATUS_REJECTED_UNSUPPORTED_VERSION},
		{"signer scheme", aTransactionAtNodeTimestamp().WithInvalidSignerScheme(), protocol.TRANSACTION_STATUS_REJECTED_UNKNOWN_SIGNER_SCHEME},
		{"signer public key (wrong length)", aTransactionAtNodeTimestamp().WithInvalidPublicKey(), protocol.TRANSACTION_STATUS_REJECTED_SIGNATURE_MISMATCH},
		{"contract name", aTransactionAtNodeTimestamp().WithContract(""), protocol.TRANSACTION_STATUS_RESERVED},
		{"timestamp (created prior to the expiry window)", builders.TransferTransaction().WithTimestamp(time.Now().Add(expirationWindowInterval * -2)), protocol.TRANSACTION_STATUS_REJECTED_TIMESTAMP_WINDOW_EXCEEDED},
		{"timestamp (ahead of timestamp for last committed block)", builders.TransferTransaction().WithTimestamp(futureTimeAfterGracePeriod()), protocol.TRANSACTION_STATUS_REJECTED_TIMESTAMP_AHEAD_OF_NODE_TIME},
		{"virtual chain id", aTransactionAtNodeTimestamp().WithVirtualChainId(primitives.VirtualChainId(1)), protocol.TRANSACTION_STATUS_REJECTED_VIRTUAL_CHAIN_MISMATCH},
	}
	for i := range tests {
		test := tests[i] // this is so that we can run tests in parallel, see https://gist.github.com/posener/92a55c4cd441fc5e5e85f27bca008721
		t.Run(test.name, func(t *testing.T) {
			err := aValidationContext().validateTransaction(test.txBuilder.Build())

			require.Error(t, err, fmt.Sprintf("a transaction with an invalid %s was not rejected", test.name))
			require.Equal(t, test.expectedStatus, err.TransactionStatus, "error status differed from expected")
		})
	}
}

func aTransactionAtNodeTimestamp() *builders.TransactionBuilder {
	return builders.TransferTransaction().WithTimestamp(time.Unix(0, int64(lastCommittedBlockTimestamp+1000)))
}
