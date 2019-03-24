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

func TestBenchmarkConsensus_BenchmarkConsensusCommitMessage(t *testing.T) {
	message := &gossipmessages.BenchmarkConsensusCommitMessage{
		BlockPair: builders.BenchmarkConsensusBlockPair().WithTransactions(5).Build(),
	}

	payloads, err := EncodeBenchmarkConsensusCommitMessage((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeBenchmarkConsensusCommitMessage(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	test.RequireCmpEqual(t, message, decoded, "decoded encoded should equal to original")
}

func TestBenchmarkConsensus_BenchmarkConsensusCommitMessageWithCorruptNumTransactions(t *testing.T) {
	message := &gossipmessages.BenchmarkConsensusCommitMessage{
		BlockPair: builders.BenchmarkConsensusBlockPair().WithCorruptNumTransactions(5).Build(),
	}

	payloads, err := EncodeBenchmarkConsensusCommitMessage((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	_, err = DecodeBenchmarkConsensusCommitMessage(payloads[1:])
	require.Error(t, err, "decode should fail and return error")
}

func TestBenchmarkConsensus_BenchmarkConsensusCommitMessageWithCorruptBlockPair(t *testing.T) {
	message := &gossipmessages.BenchmarkConsensusCommitMessage{
		BlockPair: builders.CorruptBlockPair().Build(),
	}

	_, err := EncodeBenchmarkConsensusCommitMessage((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.Error(t, err, "encode should fail and return error")

}

func TestBenchmarkConsensus_BenchmarkConsensusCommittedMessage(t *testing.T) {
	message := &gossipmessages.BenchmarkConsensusCommittedMessage{
		Status: (&gossipmessages.BenchmarkConsensusStatusBuilder{
			LastCommittedBlockHeight: 3001,
		}).Build(),
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderNodeAddress: []byte{0x01, 0x02, 0x03},
			Signature:         []byte{0x04, 0x05, 0x06},
		}).Build(),
	}

	payloads, err := EncodeBenchmarkConsensusCommittedMessage((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeBenchmarkConsensusCommittedMessage(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	test.RequireCmpEqual(t, message, decoded, "decoded encoded should equal to original")
}

func TestBenchmarkConsensus_EmptyBenchmarkConsensusCommittedMessage(t *testing.T) {
	_, err := DecodeBenchmarkConsensusCommittedMessage(builders.EmptyPayloads(2))
	require.Error(t, err, "decode should fail and return error")
}

func TestBenchmarkConsensus_BenchmarkConsensusCommittedMessageDoNotFailWhenSenderContainsNil(t *testing.T) {
	message := &gossipmessages.BenchmarkConsensusCommittedMessage{
		Status: (&gossipmessages.BenchmarkConsensusStatusBuilder{
			LastCommittedBlockHeight: 3001,
		}).Build(),
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderNodeAddress: nil,
			Signature:         nil,
		}).Build(),
	}

	payloads, err := EncodeBenchmarkConsensusCommittedMessage((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeBenchmarkConsensusCommittedMessage(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	test.RequireCmpEqual(t, message, decoded, "decoded encoded should equal to original")
	test.RequireDoesNotContainNil(t, decoded)
}
