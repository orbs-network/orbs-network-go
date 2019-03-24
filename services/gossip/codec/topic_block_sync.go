// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package codec

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/pkg/errors"
)

func EncodeBlockAvailabilityRequest(header *gossipmessages.Header, message *gossipmessages.BlockAvailabilityRequestMessage) ([][]byte, error) {
	if message.SignedBatchRange == nil {
		return nil, errors.New("missing SignedBatchRange")
	}
	if message.Sender == nil {
		return nil, errors.New("missing Sender")
	}
	return [][]byte{header.Raw(), message.SignedBatchRange.Raw(), message.Sender.Raw()}, nil
}

func DecodeBlockAvailabilityRequest(payloads [][]byte) (*gossipmessages.BlockAvailabilityRequestMessage, error) {
	if len(payloads) != 2 {
		return nil, errors.New("wrong num of payloads")
	}

	batchRange := gossipmessages.BlockSyncRangeReader(payloads[0])
	if !batchRange.IsValid() {
		return nil, errors.New("SignedBatchRange is corrupted and cannot be decoded")
	}
	senderSignature := gossipmessages.SenderSignatureReader(payloads[1])
	if !senderSignature.IsValid() {
		return nil, errors.New("SenderSignature is corrupted and cannot be decoded")
	}

	return &gossipmessages.BlockAvailabilityRequestMessage{
		SignedBatchRange: batchRange,
		Sender:           senderSignature,
	}, nil
}

func EncodeBlockAvailabilityResponse(header *gossipmessages.Header, message *gossipmessages.BlockAvailabilityResponseMessage) ([][]byte, error) {
	if message.SignedBatchRange == nil {
		return nil, errors.New("missing SignedBatchRange")
	}
	if message.Sender == nil {
		return nil, errors.New("missing Sender")
	}
	return [][]byte{header.Raw(), message.SignedBatchRange.Raw(), message.Sender.Raw()}, nil
}

func DecodeBlockAvailabilityResponse(payloads [][]byte) (*gossipmessages.BlockAvailabilityResponseMessage, error) {
	if len(payloads) != 2 {
		return nil, errors.New("wrong num of payloads")
	}

	batchRange := gossipmessages.BlockSyncRangeReader(payloads[0])
	if !batchRange.IsValid() {
		return nil, errors.New("SignedBatchRange is corrupted and cannot be decoded")
	}
	senderSignature := gossipmessages.SenderSignatureReader(payloads[1])
	if !senderSignature.IsValid() {
		return nil, errors.New("SenderSignature is corrupted and cannot be decoded")
	}

	return &gossipmessages.BlockAvailabilityResponseMessage{
		SignedBatchRange: batchRange,
		Sender:           senderSignature,
	}, nil
}

func EncodeBlockSyncRequest(header *gossipmessages.Header, message *gossipmessages.BlockSyncRequestMessage) ([][]byte, error) {
	if message.SignedChunkRange == nil {
		return nil, errors.New("missing SignedChunkRange")
	}
	if message.Sender == nil {
		return nil, errors.New("missing Sender")
	}
	return [][]byte{header.Raw(), message.SignedChunkRange.Raw(), message.Sender.Raw()}, nil
}

func DecodeBlockSyncRequest(payloads [][]byte) (*gossipmessages.BlockSyncRequestMessage, error) {
	if len(payloads) != 2 {
		return nil, errors.New("wrong num of payloads")
	}

	chunkRange := gossipmessages.BlockSyncRangeReader(payloads[0])
	if !chunkRange.IsValid() {
		return nil, errors.New("SignedChunkRange is corrupted and cannot be decoded")
	}
	senderSignature := gossipmessages.SenderSignatureReader(payloads[1])
	if !senderSignature.IsValid() {
		return nil, errors.New("SenderSignature is corrupted and cannot be decoded")
	}

	return &gossipmessages.BlockSyncRequestMessage{
		SignedChunkRange: chunkRange,
		Sender:           senderSignature,
	}, nil
}

func EncodeBlockSyncResponse(header *gossipmessages.Header, message *gossipmessages.BlockSyncResponseMessage) ([][]byte, error) {
	if message.SignedChunkRange == nil {
		return nil, errors.New("missing SignedChunkRange")
	}
	if len(message.BlockPairs) == 0 {
		return nil, errors.New("missing BlockPairs")
	}
	if message.Sender == nil {
		return nil, errors.New("missing Sender")
	}

	payloads := [][]byte{header.Raw(), message.SignedChunkRange.Raw(), message.Sender.Raw()}

	blockPairPayloads, err := EncodeBlockPairs(message.BlockPairs)
	if err != nil {
		return nil, err
	}
	return append(payloads, blockPairPayloads...), nil
}

func DecodeBlockSyncResponse(payloads [][]byte) (*gossipmessages.BlockSyncResponseMessage, error) {
	if len(payloads) < 2+NUM_HARDCODED_PAYLOADS_FOR_BLOCK_PAIR {
		return nil, errors.New("wrong num of payloads")
	}
	chunkRange := gossipmessages.BlockSyncRangeReader(payloads[0])
	if !chunkRange.IsValid() {
		return nil, errors.New("SignedChunkRange is corrupted and cannot be decoded")
	}
	senderSignature := gossipmessages.SenderSignatureReader(payloads[1])
	if !senderSignature.IsValid() {
		return nil, errors.New("SenderSignature is corrupted and cannot be decoded")
	}

	blocks, err := DecodeBlockPairs(payloads[2:])
	if err != nil {
		return nil, err
	}

	return &gossipmessages.BlockSyncResponseMessage{
		SignedChunkRange: chunkRange,
		Sender:           senderSignature,
		BlockPairs:       blocks,
	}, nil
}
