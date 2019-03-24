// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package adapter

import (
	"bytes"
	"context"
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
