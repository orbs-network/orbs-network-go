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

func EncodeHeaderAvailabilityRequest(header *gossipmessages.Header, message *gossipmessages.HeaderAvailabilityRequestMessage) ([][]byte, error) {
	if message.SignedBatchRange == nil {
		return nil, errors.New("missing SignedBatchRange")
	}
	if message.Sender == nil {
		return nil, errors.New("missing Sender")
	}
	return [][]byte{header.Raw(), message.SignedBatchRange.Raw(), message.Sender.Raw()}, nil
}

func DecodeHeaderAvailabilityRequest(payloads [][]byte) (*gossipmessages.HeaderAvailabilityRequestMessage, error) {
	if len(payloads) != NUM_HARDCODED_PAYLOADS_FOR_SIGNED_RANGE_AND_SENDER {
		return nil, errors.New("wrong num of payloads")
	}

	batchRange := gossipmessages.HeaderSyncRangeReader(payloads[0])
	if !batchRange.IsValid() {
		return nil, errors.New("SignedBatchRange is corrupted and cannot be decoded")
	}
	senderSignature := gossipmessages.SenderSignatureReader(payloads[1])
	if !senderSignature.IsValid() {
		return nil, errors.New("SenderSignature is corrupted and cannot be decoded")
	}

	return &gossipmessages.HeaderAvailabilityRequestMessage{
		SignedBatchRange: batchRange,
		Sender:           senderSignature,
	}, nil
}

func EncodeHeaderAvailabilityResponse(header *gossipmessages.Header, message *gossipmessages.HeaderAvailabilityResponseMessage) ([][]byte, error) {
	if message.SignedBatchRange == nil {
		return nil, errors.New("missing SignedBatchRange")
	}
	if message.Sender == nil {
		return nil, errors.New("missing Sender")
	}
	return [][]byte{header.Raw(), message.SignedBatchRange.Raw(), message.Sender.Raw()}, nil
}

func DecodeHeaderAvailabilityResponse(payloads [][]byte) (*gossipmessages.HeaderAvailabilityResponseMessage, error) {
	if len(payloads) != NUM_HARDCODED_PAYLOADS_FOR_SIGNED_RANGE_AND_SENDER {
		return nil, errors.New("wrong num of payloads")
	}

	batchRange := gossipmessages.HeaderSyncRangeReader(payloads[0])
	if !batchRange.IsValid() {
		return nil, errors.New("SignedBatchRange is corrupted and cannot be decoded")
	}
	senderSignature := gossipmessages.SenderSignatureReader(payloads[1])
	if !senderSignature.IsValid() {
		return nil, errors.New("SenderSignature is corrupted and cannot be decoded")
	}

	return &gossipmessages.HeaderAvailabilityResponseMessage{
		SignedBatchRange: batchRange,
		Sender:           senderSignature,
	}, nil
}

func EncodeHeaderSyncRequest(header *gossipmessages.Header, message *gossipmessages.HeaderSyncRequestMessage) ([][]byte, error) {
	if message.SignedChunkRange == nil {
		return nil, errors.New("missing SignedChunkRange")
	}
	if message.Sender == nil {
		return nil, errors.New("missing Sender")
	}
	return [][]byte{header.Raw(), message.SignedChunkRange.Raw(), message.Sender.Raw()}, nil
}

func DecodeHeaderSyncRequest(payloads [][]byte) (*gossipmessages.HeaderSyncRequestMessage, error) {
	if len(payloads) != NUM_HARDCODED_PAYLOADS_FOR_SIGNED_RANGE_AND_SENDER {
		return nil, errors.New("wrong num of payloads")
	}

	chunkRange := gossipmessages.HeaderSyncRangeReader(payloads[0])
	if !chunkRange.IsValid() {
		return nil, errors.New("SignedChunkRange is corrupted and cannot be decoded")
	}
	senderSignature := gossipmessages.SenderSignatureReader(payloads[1])
	if !senderSignature.IsValid() {
		return nil, errors.New("SenderSignature is corrupted and cannot be decoded")
	}

	return &gossipmessages.HeaderSyncRequestMessage{
		SignedChunkRange: chunkRange,
		Sender:           senderSignature,
	}, nil
}

func EncodeHeaderSyncResponse(header *gossipmessages.Header, message *gossipmessages.HeaderSyncResponseMessage) ([][]byte, error) {
	if message.SignedChunkRange == nil {
		return nil, errors.New("missing SignedChunkRange")
	}
	if len(message.HeaderWithProof) == 0 {
		return nil, errors.New("missing HeaderWithProof")
	}
	if message.Sender == nil {
		return nil, errors.New("missing Sender")
	}

	payloads := [][]byte{header.Raw(), message.SignedChunkRange.Raw(), message.Sender.Raw()}

	headerProofPayloads, err := EncodeHeadersWithProofs(message.HeaderWithProof)
	if err != nil {
		return nil, err
	}
	return append(payloads, headerProofPayloads...), nil
}

func DecodeHeaderSyncResponse(payloads [][]byte) (*gossipmessages.HeaderSyncResponseMessage, error) {
	if len(payloads) < NUM_HARDCODED_PAYLOADS_FOR_SIGNED_RANGE_AND_SENDER + NUM_HARDCODED_PAYLOADS_FOR_HEADER_WITH_PROOF {
		return nil, errors.New("wrong num of payloads")
	}
	chunkRange := gossipmessages.HeaderSyncRangeReader(payloads[0])
	if !chunkRange.IsValid() {
		return nil, errors.New("SignedChunkRange is corrupted and cannot be decoded")
	}
	senderSignature := gossipmessages.SenderSignatureReader(payloads[1])
	if !senderSignature.IsValid() {
		return nil, errors.New("SenderSignature is corrupted and cannot be decoded")
	}

	headersProofs, err := DecodeHeadersWithProofs(payloads[2:])
	if err != nil {
		return nil, err
	}

	return &gossipmessages.HeaderSyncResponseMessage{
		SignedChunkRange: chunkRange,
		Sender:           senderSignature,
		HeaderWithProof:  headersProofs,
	}, nil
}
