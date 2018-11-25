package codec

import (
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTransactionRelay_ForwardedTransactionsMessage(t *testing.T) {
	message := &gossipmessages.ForwardedTransactionsMessage{
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderPublicKey: []byte{0x01, 0x02, 0x03},
			Signature:       []byte{0x04, 0x05, 0x06},
		}).Build(),
		SignedTransactions: []*protocol.SignedTransaction{
			builders.Transaction().Build(),
		},
	}

	payloads := EncodeForwardedTransactions((&gossipmessages.HeaderBuilder{}).Build(), message)
	decoded, err := DecodeForwardedTransactions(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	test.RequireCmpEqual(t, message, decoded, "decoded encoded should equal to original")
}

func TestTransactionRelay_EmptyForwardedTransactionsMessage(t *testing.T) {
	_, err := DecodeForwardedTransactions(emptyPayloads(2))
	require.Error(t, err, "decode should fail and return error")
}

func TestTransactionRelay_ForwardedTransactionsMessageDoNotFailWhenSenderContainsNil(t *testing.T) {
	message := &gossipmessages.ForwardedTransactionsMessage{
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderPublicKey: nil,
			Signature:       nil,
		}).Build(),
		SignedTransactions: []*protocol.SignedTransaction{
			builders.Transaction().Build(),
		},
	}

	payloads := EncodeForwardedTransactions((&gossipmessages.HeaderBuilder{}).Build(), message)
	decoded, err := DecodeForwardedTransactions(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	test.RequireCmpEqual(t, message, decoded, "decoded encoded should equal to original")
	require.False(t, containsNil(decoded), "decoded should not contain nil fields")
}
