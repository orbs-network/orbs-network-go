package ethereum

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	"github.com/pkg/errors"
)

func repackEventABIWithTopics(eventABI abi.Event, log *adapter.TransactionLog) (res []byte, err error) {
	topicIndex := 0
	if !eventABI.Anonymous {
		topicIndex = 1
	}

	nonIndexedData, err := eventABI.Inputs.NonIndexed().UnpackValues(log.Data)
	if err != nil {
		return nil, errors.Wrapf(err, "failed unpacking non-indexed values: %v", nonIndexedData)
	}

	var unpacked []interface{}

	for _, arg := range eventABI.Inputs {
		if arg.Indexed {
			arg.Indexed = false
			v, err := abi.Arguments{arg}.UnpackValues(log.PackedTopics[topicIndex])
			if err != nil {
				return nil, errors.Wrapf(err, "failed unpacking indexed value: %v", log.PackedTopics[topicIndex])
			}
			unpacked = append(unpacked, v[0])

		}
	}

	for _, data := range nonIndexedData {
		unpacked = append(unpacked, data)
	}

	return eventABI.Inputs.Pack(unpacked...)
}
