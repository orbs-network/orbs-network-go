package adapter

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/pkg/errors"
	"math/big"
)

type LogTopics [][]byte

func (topics LogTopics) BigIntAt(index int) (*big.Int, error) {
	if len(topics)-1 < index {
		return nil, errors.Errorf("request index %d is out of range, got %d topics", index, len(topics))
	}
	var out big.Int
	out.SetBytes(topics[index])
	return &out, nil
}

func (topics LogTopics) BytesAt(index int, size int) ([]byte, error) {
	if len(topics)-1 < index {
		return nil, errors.Errorf("request index %d is out of range, got %d topics", index, len(topics))
	}
	to := 32
	from := to - size
	return topics[index][from:to], nil
}

type TransactionLog struct {
	ContractAddress []byte
	PackedTopics    LogTopics // indexed fields
	Data            []byte    // non-indexed fields
	BlockNumber     uint64
}

func (log *TransactionLog) UnpackDataUsing(eventABI abi.Event) ([]interface{}, error) {
	return eventABI.Inputs.UnpackValues(log.Data)
}
