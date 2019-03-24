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

func EncodeBenchmarkConsensusCommitMessage(header *gossipmessages.Header, message *gossipmessages.BenchmarkConsensusCommitMessage) ([][]byte, error) {
	blockPairPayloads, err := EncodeBlockPair(message.BlockPair)
	if err != nil {
		return nil, err
	}

	return append([][]byte{header.Raw()}, blockPairPayloads...), nil
}

func DecodeBenchmarkConsensusCommitMessage(payloads [][]byte) (*gossipmessages.BenchmarkConsensusCommitMessage, error) {
	blockPair, err := DecodeBlockPair(payloads)
	if err != nil {
		return nil, err
	}

	return &gossipmessages.BenchmarkConsensusCommitMessage{
		BlockPair: blockPair,
	}, nil
}

func EncodeBenchmarkConsensusCommittedMessage(header *gossipmessages.Header, message *gossipmessages.BenchmarkConsensusCommittedMessage) ([][]byte, error) {
	if message.Status == nil {
		return nil, errors.New("missing Status")
	}
	if message.Sender == nil {
		return nil, errors.New("missing Sender")
	}
	return [][]byte{header.Raw(), message.Status.Raw(), message.Sender.Raw()}, nil
}

func DecodeBenchmarkConsensusCommittedMessage(payloads [][]byte) (*gossipmessages.BenchmarkConsensusCommittedMessage, error) {
	if len(payloads) != 2 {
		return nil, errors.New("wrong num of payloads")
	}

	status := gossipmessages.BenchmarkConsensusStatusReader(payloads[0])
	if !status.IsValid() {
		return nil, errors.New("Status is corrupted and cannot be decoded")
	}
	senderSignature := gossipmessages.SenderSignatureReader(payloads[1])
	if !senderSignature.IsValid() {
		return nil, errors.New("SenderSignature is corrupted and cannot be decoded")
	}

	return &gossipmessages.BenchmarkConsensusCommittedMessage{
		Status: status,
		Sender: senderSignature,
	}, nil
}
