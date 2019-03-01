package adapter

import (
	"bytes"
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
)

type TransactionLog struct {
	ContractAddress []byte
	PackedTopics    [][]byte // indexed fields
	Data            []byte   // non-indexed fields
	RepackedData    []byte
	BlockNumber     uint64
	TxIndex         uint32
}

// TODO(v1): this assumes that in events Data every input is 32 bytes (eg. no tuples), is this always the case? [OdedW]
func (l *TransactionLog) PackedDataArgumentAt(index int, arg abi.Argument) ([]byte, error) {
	from := index * 32
	if from+32 > len(l.Data) {
		return nil, errors.Errorf("request index %d is out of bounds, got %d bytes", index, len(l.Data))
	}
	return l.Data[from : from+32], nil
}

func (c *connectorCommon) GetTransactionLogs(ctx context.Context, txHash primitives.Uint256, eventSignature []byte) ([]*TransactionLog, error) {
	client, err := c.getContractCaller()
	if err != nil {
		return nil, err
	}

	receipt, err := client.TransactionReceipt(ctx, common.BytesToHash(txHash))
	if err != nil {
		return nil, errors.Wrapf(err, "error getting receipt for transaction with hash %s", txHash)
	}
	if receipt == nil {
		return nil, errors.Wrapf(err, "got no logs for transaction with hash %s", txHash)
	}

	var eventLogs []*TransactionLog
	for _, log := range receipt.Logs {
		if matchesEvent(log, eventSignature) {
			var topics [][]byte
			for _, topic := range log.Topics {
				topics = append(topics, topic.Bytes())
			}
			transactionLog := &TransactionLog{
				PackedTopics:    topics,
				Data:            log.Data,
				BlockNumber:     log.BlockNumber,
				TxIndex:         uint32(log.TxIndex),
				ContractAddress: log.Address.Bytes(),
			}
			eventLogs = append(eventLogs, transactionLog)
		}
	}

	return eventLogs, nil
}

func matchesEvent(log *types.Log, eventSignature []byte) bool {
	for _, topic := range log.Topics {
		if bytes.Equal(topic.Bytes(), eventSignature) {
			return true
		}
	}

	return false
}
