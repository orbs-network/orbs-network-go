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

var vctx = validationContext {
	expiryWindow:                expirationWindowInterval,
	lastCommittedBlockTimestamp: lastCommittedBlockTimestamp,
	futureTimestampGrace:        futureTimestampGrace,
}

//TODO Sender virtual chain matches the node's virtual chain.
//Check Transaction timestamp:
//TODO * Only accept transactions that haven't expired.
// ** Transaction is expired if its timestamp is later than current time plus the configurable expiration window (eg. 30 min).
//TODO * Only accept transactions with timestamp in sync with the node (that aren't in the future).
// ** Transaction timestamp is in sync if it is earlier than the last committed block timestamp + configurable sync grace window (eg. 3 min).
// ** Note that a transaction may be rejected due to either future timestamp or node's loss of sync.
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
		name string
		txBuilder *builders.TransferTransactionBuilder
	}{
		{"protocol version", builders.TransferTransaction().WithProtocolVersion(ProtocolVersion + 1)},
		{"signer scheme", builders.TransferTransaction().WithInvalidSignerScheme()},
		{"empty signer public key", builders.TransferTransaction().WithSigner(protocol.NETWORK_TYPE_TEST_NET, primitives.Ed25519PublicKey([]byte{}))},
		{"signer public key (wrong length)", builders.TransferTransaction().WithSigner(protocol.NETWORK_TYPE_TEST_NET, keys.Ed25519KeyPairForTests(1).PublicKey()[1:])},
		{"contract name", builders.TransferTransaction().WithContract("")},
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
	timeBeforeExpirationWindow := expirationWindowInterval * -2
	tx := builders.TransferTransaction().WithTimestamp(time.Now().Add(timeBeforeExpirationWindow)).Build()

	require.Error(t, validateTransaction(tx, vctx), "a transaction that was created prior to the expiry window was not rejected")
}

func TestValidateTransaction_InvalidTimestamp_InTheFuture(t *testing.T) {
	timeAfterGraceWindow := time.Unix(0, int64(lastCommittedBlockTimestamp)).Add(futureTimestampGrace + 1 * time.Minute)
	tx := builders.TransferTransaction().WithTimestamp(timeAfterGraceWindow).Build()

	require.Error(t, validateTransaction(tx, vctx), "a transaction that was created after the grace period for last committed block was not rejected")
}
