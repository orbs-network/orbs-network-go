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

var vctx = validationContext{
	expiryWindow:                expirationWindowInterval,
	lastCommittedBlockTimestamp: lastCommittedBlockTimestamp,
	futureTimestampGrace:        futureTimestampGrace,
	virtualChainId:              primitives.VirtualChainId(42),
}

//TODO * Transaction (tx_id) doesn't already exist in the pending pool or committed pool (duplicate).
//TODO Verify pre order checks (like signature and subscription) by calling VirtualMachine.TransactionSetPreOrder.
//TODO assert signer scheme is Eddsa and public key is correct size

func TestValidateTransaction_ValidTransaction(t *testing.T) {
	require.NoError(t,
		validateTransaction(builders.TransferTransaction().Build(), vctx),
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
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Error(t,
				validateTransaction(test.txBuilder.Build(), vctx),
				fmt.Sprintf("a transaction with an invalid %s was not rejected", test.name))
		})
	}
}

func TestValidateTransaction_InvalidTimestamp_InThePast(t *testing.T) {
	tx := builders.TransferTransaction().WithTimestamp(time.Now().Add(expirationWindowInterval * -2)).Build()

	require.Error(t, validateTransaction(tx, vctx), "a transaction that was created prior to the expiry window was not rejected")
}

func futureTimeAfterGracePeriod() time.Time {
	return time.Unix(0, int64(lastCommittedBlockTimestamp)).Add(futureTimestampGrace + 1*time.Minute)
}
