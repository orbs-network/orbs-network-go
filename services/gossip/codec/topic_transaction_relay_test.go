// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

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
			SenderNodeAddress: []byte{0x01, 0x02, 0x03},
			Signature:         []byte{0x04, 0x05, 0x06},
		}).Build(),
		SignedTransactions: []*protocol.SignedTransaction{
			builders.Transaction().Build(),
		},
	}

	payloads, err := EncodeForwardedTransactions((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeForwardedTransactions(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	test.RequireCmpEqual(t, message, decoded, "decoded encoded should equal to original")
}

func TestTransactionRelay_EmptyForwardedTransactionsMessage(t *testing.T) {
	_, err := DecodeForwardedTransactions(builders.EmptyPayloads(2))
	require.Error(t, err, "decode should fail and return error")
}

func TestTransactionRelay_ForwardedTransactionsMessageDoNotFailWhenSenderContainsNil(t *testing.T) {
	message := &gossipmessages.ForwardedTransactionsMessage{
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderNodeAddress: nil,
			Signature:         nil,
		}).Build(),
		SignedTransactions: []*protocol.SignedTransaction{
			builders.Transaction().Build(),
		},
	}

	payloads, err := EncodeForwardedTransactions((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeForwardedTransactions(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	test.RequireCmpEqual(t, message, decoded, "decoded encoded should equal to original")
	test.RequireDoesNotContainNil(t, decoded)
}
