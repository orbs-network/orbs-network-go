package adapter

import (
	"bytes"
	"context"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
	"math/big"
)

type ethereumAdapterConfig interface {
	EthereumEndpoint() string
}

type EthereumConnection interface {
	CallContract(ctx context.Context, contractAddress []byte, packedInput []byte, blockNumber *big.Int) (packedOutput []byte, err error)
	GetTransactionLogs(ctx context.Context, txHash primitives.Uint256, eventSignature []byte) ([]*TransactionLog, error)
}

type connectorCommon struct {
	logger            log.BasicLogger
	getContractCaller func() (EthereumCaller, error)
}

type EthereumCaller interface {
	bind.ContractBackend
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
}

func (c *connectorCommon) CallContract(ctx context.Context, contractAddress []byte, packedInput []byte, blockNumber *big.Int) (packedOutput []byte, err error) {
	client, err := c.getContractCaller()
	if err != nil {
		return nil, err
	}

	address := common.BytesToAddress(contractAddress)

	// we do not support pending calls, opts is always empty
	opts := new(bind.CallOpts)

	msg := ethereum.CallMsg{From: opts.From, To: &address, Data: packedInput}
	output, err := client.CallContract(ctx, msg, blockNumber)
	if err == nil && len(output) == 0 {
		// Make sure we have a contract to operate on, and bail out otherwise.
		if code, err := client.CodeAt(ctx, address, blockNumber); err != nil {
			return nil, err
		} else if len(code) == 0 {
			return nil, bind.ErrNoCode
		}
	}

	return output, err
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
