package codec

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/pkg/errors"
)

func EncodeLeanHelixMessage(header *gossipmessages.Header, message *gossipmessages.LeanHelixMessage) ([][]byte, error) {
	if message.Content == nil {
		return nil, errors.New("missing Content")
	}

	blockPairPayloads, err := EncodeBlockPair(message.BlockPair)
	if err != nil {
		return nil, err
	}

	return append([][]byte{header.Raw(), message.Content}, blockPairPayloads...), nil
}

func DecodeLeanHelixMessage(header *gossipmessages.Header, payloads [][]byte) (*gossipmessages.LeanHelixMessage, error) {
	if len(payloads) < 3 {
		return nil, errors.New("wrong num of payloads")
	}

	messageType := header.LeanHelix()
	content := payloads[1]
	blockPair, err := DecodeBlockPair(payloads[2:])
	if err != nil {
		return nil, err
	}

	return &gossipmessages.LeanHelixMessage{
		MessageType: messageType,
		Content:     content,
		BlockPair:   blockPair,
	}, nil
}
