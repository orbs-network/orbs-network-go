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

	for _, arg := range eventABI.Inputs {
		if arg.Indexed {

			if curTopicIndex >= len(log.PackedTopics) {
				return nil, errors.Errorf("num topics %d does not match total inputs %d minus %d non indexed", len(log.PackedTopics), len(eventABI.Inputs), eventABI.Inputs.LengthNonIndexed())
			}
			res = append(res, log.PackedTopics[curTopicIndex]...)
			curTopicIndex++

		} else {

			b, err := log.PackedDataArgumentAt(curDataIndex)
			if err != nil {
				return nil, err
			}
			res = append(res, b...)
			curDataIndex++

		}
	}

	return
}
