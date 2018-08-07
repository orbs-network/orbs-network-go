package transactionpool

import (
	"testing"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/stretchr/testify/require"
		"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"fmt"
)

//TODO Sender virtual chain matches contract virtual chain and matches the transaction pool's virtual chain.
//Check Transaction timestamp:
//TODO * Only accept transactions that haven't expired.
// ** Transaction is expired if its timestamp is later than current time plus the configurable expiration window (eg. 30 min).
//TODO * Only accept transactions with timestamp in sync with the node (that aren't in the future).
// ** Transaction timestamp is in sync if it is earlier than the last committed block timestamp + configurable sync grace window (eg. 3 min).
// ** Note that a transaction may be rejected due to either future timestamp or node's loss of sync.
//TODO * Transaction (tx_id) doesn't already exist in the pending pool or committed pool (duplicate).
//TODO Verify pre order checks (like signature and subscription) by calling VirtualMachine.TransactionSetPreOrder.

func TestValidateTransaction_ValidTransaction(t *testing.T) {
	require.NoError(t,
		validateTransaction(builders.TransferTransaction().Build()),
		"a valid transaction was rejected")
}


func TestValidateTransaction_InvalidTransactions(t *testing.T) {
	tests := []struct {
		name string
		txBuilder *builders.TransferTransactionBuilder
	}{
		{"protocol version", builders.TransferTransaction().WithProtocolVersion(ProtocolVersion + 1)},
		{"signer public key", builders.TransferTransaction().WithSigner(protocol.NETWORK_TYPE_TEST_NET, primitives.Ed25519PublicKey([]byte{}))},
		{"contract name", builders.TransferTransaction().WithContract("")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Error(t,
				validateTransaction(test.txBuilder.Build()),
				fmt.Sprintf("a transaction with an invalid %s was not rejected", test.name))
		})
	}
}

