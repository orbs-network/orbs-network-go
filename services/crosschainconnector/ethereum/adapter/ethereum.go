package adapter

import (
	"bytes"
	"context"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"math/big"
)

type TransactionLog struct {
	ContractAddress []byte
	PackedTopics [][]byte // indexed fields
	Data         []byte   // non-indexed fields
	BlockNumber  uint64
}

type ethereumAdapterConfig interface {
	EthereumEndpoint() string
}

type EthereumConnection interface {
	CallContract(ctx context.Context, address []byte, packedInput []byte, blockNumber *big.Int) (packedOutput []byte, err error)
	GetLogs(ctx context.Context, txHash primitives.Uint256, contractAddress []byte, eventSignature []byte) ([]*TransactionLog, error)
}

type connectorCommon struct {
	getContractCaller func() (EthereumCaller, error)
}

type EthereumCaller interface {
	bind.ContractBackend
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
}

func (c *connectorCommon) CallContract(ctx context.Context, address []byte, packedInput []byte, blockNumber *big.Int) (packedOutput []byte, err error) {
	client, err := c.getContractCaller()
	if err != nil {
		return nil, err
	}

	contractAddress := common.BytesToAddress(address)

	// we do not support pending calls, opts is always empty
	opts := new(bind.CallOpts)

	msg := ethereum.CallMsg{From: opts.From, To: &contractAddress, Data: packedInput}
	output, err := client.CallContract(ctx, msg, blockNumber)
	if err == nil && len(output) == 0 {
		// Make sure we have a contract to operate on, and bail out otherwise.
		if code, err := client.CodeAt(ctx, contractAddress, blockNumber); err != nil {
			return nil, err
		} else if len(code) == 0 {
			return nil, bind.ErrNoCode
		}
	}

	return output, err
}

func (c *connectorCommon) GetLogs(ctx context.Context, txHash primitives.Uint256, contractAddress []byte, eventSignature []byte) ([]*TransactionLog, error) {
	client, err := c.getContractCaller()
	if err != nil {
		return nil, err
	}

	receipt, err := client.TransactionReceipt(ctx, common.BytesToHash(txHash))
	if err != nil {
		return nil, err
	}

	var eventLogs []*TransactionLog
	for _, log := range receipt.Logs {
		if matchesEvent(log, eventSignature) {
			var topics [][]byte
			for _, topic := range log.Topics {
				topics = append(topics, topic.Bytes())
			}
			transactionLog := &TransactionLog{
				PackedTopics: topics,
				Data:         log.Data,
				BlockNumber:  log.BlockNumber,
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
