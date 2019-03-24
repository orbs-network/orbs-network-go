// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package codec

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/pkg/errors"
)

func EncodeLeanHelixMessage(header *gossipmessages.Header, message *gossipmessages.LeanHelixMessage) ([][]byte, error) {
	if message.Content == nil {
		return nil, errors.New("missing Content")
	}

	var blockPairPayloads [][]byte
	if message.BlockPair != nil {
		var err error
		blockPairPayloads, err = EncodeBlockPair(message.BlockPair)
		if err != nil {
			return nil, err
		}
	}

	payloads := [][]byte{header.Raw(), message.Content}
	if len(blockPairPayloads) > 0 {
		payloads = append(payloads, blockPairPayloads...)
	}

	return payloads, nil
}

func DecodeLeanHelixMessage(header *gossipmessages.Header, payloads [][]byte) (*gossipmessages.LeanHelixMessage, error) {
	if len(payloads) < 1 {
		return nil, errors.New("wrong num of payloads")
	}

	content := payloads[0]

	var blockPair *protocol.BlockPairContainer
	if len(payloads) > 1 {
		var err error
		blockPair, err = DecodeBlockPair(payloads[1:])
		if err != nil {
			return nil, err
		}
	}

	return &gossipmessages.LeanHelixMessage{
		Content:   content,
		BlockPair: blockPair,
	}, nil
}
