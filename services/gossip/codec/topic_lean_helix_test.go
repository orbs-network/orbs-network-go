// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package codec

import (
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLeanHelix_LeanHelixMessage(t *testing.T) {
	header := (&gossipmessages.HeaderBuilder{
		Topic:         gossipmessages.HEADER_TOPIC_LEAN_HELIX,
		RecipientMode: gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
	}).Build()

	message := &gossipmessages.LeanHelixMessage{
		Content:   []byte{},
		BlockPair: builders.BlockPair().WithTransactions(5).Build(),
	}

	payloads, err := EncodeLeanHelixMessage(header, message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeLeanHelixMessage(header, payloads[1:])
	require.NoError(t, err, "decode should not fail")
	test.RequireCmpEqual(t, message, decoded, "decoded encoded should equal to original")
}

func TestLeanHelix_LeanHelixMessageWithNoBlockPair(t *testing.T) {
	header := (&gossipmessages.HeaderBuilder{
		Topic:         gossipmessages.HEADER_TOPIC_LEAN_HELIX,
		RecipientMode: gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
	}).Build()

	message := &gossipmessages.LeanHelixMessage{
		Content: []byte{},
	}

	payloads, err := EncodeLeanHelixMessage(header, message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeLeanHelixMessage(header, payloads[1:])
	require.NoError(t, err, "decode should not fail")
	test.RequireCmpEqual(t, message, decoded, "decoded encoded should equal to original")
	test.RequireDoesNotContainNil(t, decoded)
}

func TestLeanHelix_EmptyLeanHelixMessage(t *testing.T) {
	header := (&gossipmessages.HeaderBuilder{
		Topic:         gossipmessages.HEADER_TOPIC_LEAN_HELIX,
		RecipientMode: gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
	}).Build()

	decoded, err := DecodeLeanHelixMessage(header, builders.EmptyPayloads(1+NUM_HARDCODED_PAYLOADS_FOR_BLOCK_PAIR))
	require.NoError(t, err, "decode should not fail")
	test.RequireDoesNotContainNil(t, decoded)
}

func TestLeanHelix_LeanHelixMessageWithCorruptedBlockPair(t *testing.T) {
	header := (&gossipmessages.HeaderBuilder{
		Topic:         gossipmessages.HEADER_TOPIC_LEAN_HELIX,
		RecipientMode: gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
	}).Build()

	message := &gossipmessages.LeanHelixMessage{
		Content:   []byte{},
		BlockPair: builders.CorruptBlockPair().Build(),
	}

	_, err := EncodeLeanHelixMessage(header, message)
	require.Error(t, err, "encode should fail and return error")
}

func TestLeanHelix_LeanHelixMessageWithCorruptNumTransactions(t *testing.T) {
	header := (&gossipmessages.HeaderBuilder{
		Topic:         gossipmessages.HEADER_TOPIC_LEAN_HELIX,
		RecipientMode: gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
	}).Build()

	message := &gossipmessages.LeanHelixMessage{
		Content:   []byte{},
		BlockPair: builders.BlockPair().WithCorruptNumTransactions(3).Build(),
	}

	payloads, err := EncodeLeanHelixMessage(header, message)
	require.NoError(t, err, "encode should not fail")
	_, err = DecodeLeanHelixMessage(header, payloads[1:])
	require.Error(t, err, "decode should fail and return error")
}
