// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package codec

import (
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestHeaderSync_HeaderAvailabilityRequest(t *testing.T) {
	message := &gossipmessages.HeaderAvailabilityRequestMessage{
		SignedBatchRange: (&gossipmessages.HeaderSyncRangeBuilder{
			HeaderType:                gossipmessages.HEADER_TYPE_RESULTS_BLOCK_HEADER_WITH_PROOF,
			FirstBlockHeight:         1001,
			LastBlockHeight:          2001,
			LastCommittedBlockHeight: 3001,
		}).Build(),
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderNodeAddress: []byte{0x01, 0x02, 0x03},
			Signature:         []byte{0x04, 0x05, 0x06},
		}).Build(),
	}

	payloads, err := EncodeHeaderAvailabilityRequest((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeHeaderAvailabilityRequest(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	test.RequireCmpEqual(t, message, decoded, "decoded encoded should equal to original")
}

func TestHeaderSync_EmptyHeaderAvailabilityRequest(t *testing.T) {
	_, err := DecodeHeaderAvailabilityRequest(builders.EmptyPayloads(2))
	require.Error(t, err, "decode should fail and return error")
}

func TestHeaderSync_HeaderAvailabilityRequestDoNotFailWhenSenderContainsNil(t *testing.T) {
	message := &gossipmessages.HeaderAvailabilityRequestMessage{
		SignedBatchRange: (&gossipmessages.HeaderSyncRangeBuilder{
			HeaderType:                gossipmessages.HEADER_TYPE_RESULTS_BLOCK_HEADER_WITH_PROOF,
			FirstBlockHeight:         1001,
			LastBlockHeight:          2001,
			LastCommittedBlockHeight: 3001,
		}).Build(),
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderNodeAddress: nil,
			Signature:         nil,
		}).Build(),
	}

	payloads, err := EncodeHeaderAvailabilityRequest((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeHeaderAvailabilityRequest(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	test.RequireCmpEqual(t, message, decoded, "decoded encoded should equal to original")
	test.RequireDoesNotContainNil(t, decoded)
}

func TestHeaderSync_HeaderAvailabilityResponse(t *testing.T) {
	message := &gossipmessages.HeaderAvailabilityResponseMessage{
		SignedBatchRange: (&gossipmessages.HeaderSyncRangeBuilder{
			HeaderType:                gossipmessages.HEADER_TYPE_RESULTS_BLOCK_HEADER_WITH_PROOF,
			FirstBlockHeight:         1001,
			LastBlockHeight:          2001,
			LastCommittedBlockHeight: 3001,
		}).Build(),
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderNodeAddress: []byte{0x01, 0x02, 0x03},
			Signature:         []byte{0x04, 0x05, 0x06},
		}).Build(),
	}

	payloads, err := EncodeHeaderAvailabilityResponse((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeHeaderAvailabilityResponse(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	test.RequireCmpEqual(t, message, decoded, "decoded encoded should equal to original")
}

func TestHeaderSync_EmptyHeaderAvailabilityResponse(t *testing.T) {
	_, err := DecodeHeaderAvailabilityResponse(builders.EmptyPayloads(2))
	require.Error(t, err, "decode should fail and return error")
}

func TestHeaderSync_HeaderAvailabilityResponseDoNotFailWhenSenderContainsNil(t *testing.T) {
	message := &gossipmessages.HeaderAvailabilityResponseMessage{
		SignedBatchRange: (&gossipmessages.HeaderSyncRangeBuilder{
			HeaderType:                gossipmessages.HEADER_TYPE_RESULTS_BLOCK_HEADER_WITH_PROOF,
			FirstBlockHeight:         1001,
			LastBlockHeight:          2001,
			LastCommittedBlockHeight: 3001,
		}).Build(),
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderNodeAddress: nil,
			Signature:         nil,
		}).Build(),
	}

	payloads, err := EncodeHeaderAvailabilityResponse((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeHeaderAvailabilityResponse(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	test.RequireCmpEqual(t, message, decoded, "decoded encoded should equal to original")
	test.RequireDoesNotContainNil(t, decoded)
}

func TestHeaderSync_HeaderSyncRequest(t *testing.T) {
	message := &gossipmessages.HeaderSyncRequestMessage{
		SignedChunkRange: (&gossipmessages.HeaderSyncRangeBuilder{
			HeaderType:                gossipmessages.HEADER_TYPE_RESULTS_BLOCK_HEADER_WITH_PROOF,
			FirstBlockHeight:         1001,
			LastBlockHeight:          2001,
			LastCommittedBlockHeight: 3001,
		}).Build(),
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderNodeAddress: []byte{0x01, 0x02, 0x03},
			Signature:         []byte{0x04, 0x05, 0x06},
		}).Build(),
	}

	payloads, err := EncodeHeaderSyncRequest((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeHeaderSyncRequest(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	test.RequireCmpEqual(t, message, decoded, "decoded encoded should equal to original")
}

func TestHeaderSync_EmptyHeaderSyncRequest(t *testing.T) {
	_, err := DecodeHeaderSyncRequest(builders.EmptyPayloads(2))
	require.Error(t, err, "decode should fail and return error")
}

func TestHeaderSync_HeaderSyncRequestDoNotFailWhenSenderContainsNil(t *testing.T) {
	message := &gossipmessages.HeaderSyncRequestMessage{
		SignedChunkRange: (&gossipmessages.HeaderSyncRangeBuilder{
			HeaderType:                gossipmessages.HEADER_TYPE_RESULTS_BLOCK_HEADER_WITH_PROOF,
			FirstBlockHeight:         1001,
			LastBlockHeight:          2001,
			LastCommittedBlockHeight: 3001,
		}).Build(),
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderNodeAddress: nil,
			Signature:         nil,
		}).Build(),
	}

	payloads, err := EncodeHeaderSyncRequest((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeHeaderSyncRequest(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	test.RequireCmpEqual(t, message, decoded, "decoded encoded should equal to original")
	test.RequireDoesNotContainNil(t, decoded)
}

func TestHeaderSync_HeaderSyncResponse(t *testing.T) {
	message := &gossipmessages.HeaderSyncResponseMessage{
		SignedChunkRange: (&gossipmessages.HeaderSyncRangeBuilder{
			HeaderType:               gossipmessages.HEADER_TYPE_RESULTS_BLOCK_HEADER_WITH_PROOF,
			FirstBlockHeight:         1001,
			LastBlockHeight:          2001,
			LastCommittedBlockHeight: 3001,
		}).Build(),
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderNodeAddress: []byte{0x01, 0x02, 0x03},
			Signature:         []byte{0x04, 0x05, 0x06},
		}).Build(),
		HeaderWithProof: []*gossipmessages.ResultsBlockHeaderWithProof{
			buildResultsBlockHeaderWithProof(3),
			buildResultsBlockHeaderWithProof(4),
		},
	}

	payloads, err := EncodeHeaderSyncResponse((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeHeaderSyncResponse(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	test.RequireCmpEqual(t, message, decoded, "decoded encoded should equal to original")
}


type corruptBlockPair struct {
	txContainer *protocol.TransactionsBlockContainer
	rxContainer *protocol.ResultsBlockContainer
}

func CorruptBlockPair() *corruptBlockPair {
	return &corruptBlockPair{}
}

func (c *corruptBlockPair) Build() *protocol.BlockPairContainer {
	return &protocol.BlockPairContainer{
		TransactionsBlock: c.txContainer,
		ResultsBlock:      c.rxContainer,
	}
}



func buildResultsBlockHeaderWithProof(height primitives.BlockHeight) *gossipmessages.ResultsBlockHeaderWithProof {
	empty32ByteHash := hash.Make32EmptyBytes()
	rxBlockHeader := (&protocol.ResultsBlockHeaderBuilder{
		ProtocolVersion:                 primitives.ProtocolVersion(1),
		VirtualChainId:                  primitives.VirtualChainId(42),
		BlockHeight:                     height,
		PrevBlockHashPtr:                empty32ByteHash,
		Timestamp:                       primitives.TimestampNano(time.Now().UnixNano()),
		ReceiptsMerkleRootHash:          empty32ByteHash,
		StateDiffHash:                   empty32ByteHash,
		TransactionsBlockHashPtr:        empty32ByteHash,
		PreExecutionStateMerkleRootHash: empty32ByteHash,
		NumContractStateDiffs:           1,
		NumTransactionReceipts:          1,
		BlockProposerAddress: 			 empty32ByteHash,
	}).Build()

	keyPair := keys.EcdsaSecp256K1KeyPairForTests(0)
	rxProof := (&protocol.ResultsBlockProofBuilder{
		Type: protocol.RESULTS_BLOCK_PROOF_TYPE_BENCHMARK_CONSENSUS,
		BenchmarkConsensus: &consensus.BenchmarkConsensusBlockProofBuilder{
			BlockRef: nil,
			Nodes: []*consensus.BenchmarkConsensusSenderSignatureBuilder{{
				SenderNodeAddress: keyPair.NodeAddress(),
				Signature:         nil,
			}},
			Placeholder: []byte{0x01, 0x02},
		},
	}).Build()

	headerProof := &gossipmessages.ResultsBlockHeaderWithProof{
		Header: 	rxBlockHeader,
		BlockProof: rxProof,
	}
	return headerProof
}

func buildCorruptResultsBlockHeaderWithProof(height primitives.BlockHeight) *gossipmessages.ResultsBlockHeaderWithProof {
	headerProof := &gossipmessages.ResultsBlockHeaderWithProof{}
	return headerProof
}

func TestHeaderSync_EmptyHeaderSyncResponse(t *testing.T) {
	_, err := DecodeHeaderSyncResponse(builders.EmptyPayloads(2 + NUM_HARDCODED_PAYLOADS_FOR_HEADER_WITH_PROOF))
	require.Error(t, err, "decode should fail and return error")
}

func TestHeaderSync_HeaderSyncResponseDoNotFailWhenSenderContainsNil(t *testing.T) {
	message := &gossipmessages.HeaderSyncResponseMessage{
		SignedChunkRange: (&gossipmessages.HeaderSyncRangeBuilder{
			HeaderType:                gossipmessages.HEADER_TYPE_RESULTS_BLOCK_HEADER_WITH_PROOF,
			FirstBlockHeight:         1001,
			LastBlockHeight:          2001,
			LastCommittedBlockHeight: 3001,
		}).Build(),
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderNodeAddress: nil,
			Signature:         nil,
		}).Build(),
		HeaderWithProof: []*gossipmessages.ResultsBlockHeaderWithProof{
			buildResultsBlockHeaderWithProof(3),
			buildResultsBlockHeaderWithProof(4),
		},
	}

	payloads, err := EncodeHeaderSyncResponse((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.NoError(t, err, "encode should not fail")
	decoded, err := DecodeHeaderSyncResponse(payloads[1:])
	require.NoError(t, err, "decode should not fail")
	test.RequireCmpEqual(t, message, decoded, "decoded encoded should equal to original")
	test.RequireDoesNotContainNil(t, decoded)
}


func TestHeaderSync_HeaderSyncResponseWithCorruptHeaderProof(t *testing.T) {
	message := &gossipmessages.HeaderSyncResponseMessage{
		SignedChunkRange: (&gossipmessages.HeaderSyncRangeBuilder{
			HeaderType:                gossipmessages.HEADER_TYPE_RESULTS_BLOCK_HEADER_WITH_PROOF,
			FirstBlockHeight:         1001,
			LastBlockHeight:          2001,
			LastCommittedBlockHeight: 3001,
		}).Build(),
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderNodeAddress: []byte{0x01, 0x02, 0x03},
			Signature:         []byte{0x04, 0x05, 0x06},
		}).Build(),
		HeaderWithProof: []*gossipmessages.ResultsBlockHeaderWithProof{
			buildCorruptResultsBlockHeaderWithProof(7),
			buildCorruptResultsBlockHeaderWithProof(8),
		},
	}

	_, err := EncodeHeaderSyncResponse((&gossipmessages.HeaderBuilder{}).Build(), message)
	require.Error(t, err, "encode should fail and return error")
}
