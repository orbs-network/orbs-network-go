package adapter

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/pkg/errors"
	"math/big"
)

type TransactionLog struct {
	ContractAddress []byte
	PackedTopics    [][]byte // indexed fields
	Data            []byte   // non-indexed fields
	BlockNumber     uint64
}

func (l *TransactionLog) UnpackTopicBigIntAt(index int) (*big.Int, error) {
	if len(l.PackedTopics)-1 < index {
		return nil, errors.Errorf("request index %d is out of range, got %d topics", index, len(l.PackedTopics))
	}
	var out big.Int
	out.SetBytes(l.PackedTopics[index])
	return &out, nil
}

func (l *TransactionLog) UnpackTopicBytesAt(index int, size int) ([]byte, error) {
	if len(l.PackedTopics)-1 < index {
		return nil, errors.Errorf("request index %d is out of range, got %d topics", index, len(l.PackedTopics))
	}
	if size > 32 {
		return nil, errors.Errorf("request size %d is out of bounds", size)
	}
	to := 32
	from := to - size
	return l.PackedTopics[index][from:to], nil
}

func (l *TransactionLog) UnpackDataUsing(eventABI abi.Event) ([]interface{}, error) {
	return eventABI.Inputs.UnpackValues(l.Data)
}

// TODO(v1): this assumes that in events Data every input is 32 bytes (eg. no tuples), is this always the case? [OdedW]
func (l *TransactionLog) PackedDataArgumentAt(index int) ([]byte, error) {
	from := index * 32
	if from+32 > len(l.Data) {
		return nil, errors.Errorf("request index %d is out of bounds, got %d bytes", index, len(l.Data))
	}
	return l.Data[from : from+32], nil
}
