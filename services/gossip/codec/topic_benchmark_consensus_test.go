package codec

import (
	"github.com/orbs-network/orbs-network-go/test"
	//"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBenchmarkConsensus_BenchmarkConsensusCommitted(t *testing.T) {
	message := &gossipmessages.BenchmarkConsensusCommittedMessage{
		Status: (&gossipmessages.BenchmarkConsensusStatusBuilder{
			LastCommittedBlockHeight: 3001,
		}).Build(),
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderPublicKey: []byte{0x01, 0x02, 0x03},
			Signature:       []byte{0x04, 0x05, 0x06},
		}).Build(),
	}

	payloads, err := EncodeBenchmarkConsensusCommitted((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeBenchmarkConsensusCommitted(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	test.RequireCmpEqual(t, message, decoded, "decoded encoded should equal to original")
}

func TestBenchmarkConsensus_EmptyBenchmarkConsensusCommitted(t *testing.T) {
	decoded, err := DecodeBlockAvailabilityRequest(emptyPayloads(2))
	require.NoError(t, err, "decode should not fail")
	require.False(t, containsNil(decoded), "decoded should not contain nil fields")
}

func TestBenchmarkConsensus_DoNotFailWhenPublicKeyIsNil(t *testing.T) {
	message := &gossipmessages.BenchmarkConsensusCommittedMessage{
		Status: (&gossipmessages.BenchmarkConsensusStatusBuilder{
			LastCommittedBlockHeight: 3001,
		}).Build(),
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderPublicKey: nil,
			Signature:       []byte{0x04, 0x05, 0x06},
		}).Build(),
	}

	payloads, err := EncodeBenchmarkConsensusCommitted((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeBenchmarkConsensusCommitted(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	test.RequireCmpEqual(t, message, decoded, "decoded encoded should equal to original")
}

func TestBenchmarkConsensus_DoNotFailWhenSignatureIsNil(t *testing.T) {
	message := &gossipmessages.BenchmarkConsensusCommittedMessage{
		Status: (&gossipmessages.BenchmarkConsensusStatusBuilder{
			LastCommittedBlockHeight: 3001,
		}).Build(),
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderPublicKey: []byte{0x01, 0x02, 0x03},
			Signature:       nil,
		}).Build(),
	}

	payloads, err := EncodeBenchmarkConsensusCommitted((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeBenchmarkConsensusCommitted(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	test.RequireCmpEqual(t, message, decoded, "decoded encoded should equal to original")
}
