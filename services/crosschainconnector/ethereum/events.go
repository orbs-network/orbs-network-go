package ethereum

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	"github.com/pkg/errors"
)

func repackEventABIWithTopics(eventABI abi.Event, log *adapter.TransactionLog) (res []byte, err error) {
	curDataIndex := 0
	curTopicIndex := 0
	if !eventABI.Anonymous {
		curTopicIndex = 1
	}

	nonIndexedData, err := eventABI.Inputs.NonIndexed().UnpackValues(log.Data)

	if err != nil {
		return nil, errors.Wrapf(err, "failed unpacking non-indexed values: %v", nonIndexedData)
	}

	for _, arg := range eventABI.Inputs {
		if arg.Indexed {

			if curTopicIndex >= len(log.PackedTopics) {
				return nil, errors.Errorf("num topics %d does not match total inputs %d minus %d non indexed", len(log.PackedTopics), len(eventABI.Inputs), eventABI.Inputs.LengthNonIndexed())
			}
			res = append(res, log.PackedTopics[curTopicIndex]...)
			curTopicIndex++

		} else {
			repacked, err := abi.Arguments{arg}.Pack(nonIndexedData[curDataIndex])
			if err != nil {
				return nil, errors.Wrapf(err, "failed repacking non-indexed field %v with value %v", arg, nonIndexedData[curDataIndex])
			}
			res = append(res, repacked...)
			curDataIndex++

		}
	}

	return
}
