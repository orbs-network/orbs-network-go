package codec

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/pkg/errors"
)

func EncodeBenchmarkConsensusCommitted(header *gossipmessages.Header, message *gossipmessages.BenchmarkConsensusCommittedMessage) ([][]byte, error) {
	if message.Status == nil {
		return nil, errors.New("missing Status")
	}
	return [][]byte{header.Raw(), message.Status.Raw(), message.Sender.Raw()}, nil
}

func DecodeBenchmarkConsensusCommitted(payloads [][]byte) (*gossipmessages.BenchmarkConsensusCommittedMessage, error) {
	if len(payloads) != 2 {
		return nil, errors.New("wrong num of payloads")
	}
	status := gossipmessages.BenchmarkConsensusStatusReader(payloads[0])
	senderSignature := gossipmessages.SenderSignatureReader(payloads[1])
	return &gossipmessages.BenchmarkConsensusCommittedMessage{
		Status: status,
		Sender: senderSignature,
	}, nil
}
