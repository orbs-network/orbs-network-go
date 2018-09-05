package transactionpool

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

const expirationWindowInterval = 30 * time.Minute
const futureTimestampGrace = 3 * time.Minute

var lastCommittedBlockTimestamp = primitives.TimestampNano(time.Now().Add(-5 * time.Second).UnixNano())

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

	err := aValidationContext().validateTransaction(builders.TransferTransaction().Build())
	require.Nil(t, err, "a valid transaction was rejected")
}

//TODO talk to TalKol about Invalid Signer
func TestValidateTransaction_InvalidTransactions(t *testing.T) {
	tests := []struct {
		name      string
		txBuilder *builders.TransactionBuilder
	}{
		{"protocol version", builders.TransferTransaction().WithProtocolVersion(ProtocolVersion + 1)},
		{"signer scheme", builders.TransferTransaction().WithInvalidSignerScheme()},
		{"signer public key (wrong length)", builders.TransferTransaction().WithInvalidPublicKey()},
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
				aValidationContext().validateTransaction(test.txBuilder.Build()),
				fmt.Sprintf("a transaction with an invalid %s was not rejected", test.name))
		})
	}
}
