package adapter

import (
	"context"
	"errors"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"math/big"
	"sync"
	"time"
)

type EthereumRpcConnection struct {
	connectorCommon

	config ethereumAdapterConfig

	mu struct {
		sync.Mutex
		client     EthereumCaller
		fullClient *ethclient.Client
	}
}

func NewEthereumRpcConnection(config ethereumAdapterConfig, logger log.BasicLogger) *EthereumRpcConnection {
	rpc := &EthereumRpcConnection{
		config: config,
	}
	rpc.logger = logger.WithTags(log.String("adapter", "ethereum"))
	rpc.getContractCaller = rpc.dialIfNeededAndReturnClient
	rpc.getBlockByTimestamp = rpc.getEthBlockByTimestamp
	return rpc
}

func (rpc *EthereumRpcConnection) dial() error {
	rpc.mu.Lock()
	defer rpc.mu.Unlock()
	if client, err := ethclient.Dial(rpc.config.EthereumEndpoint()); err != nil {
		return err
	} else {
		rpc.mu.client = client
		rpc.mu.fullClient = client
	}
	return nil
}

func (rpc *EthereumRpcConnection) dialIfNeededAndReturnClient() (EthereumCaller, error) {
	if rpc.mu.client == nil {
		if err := rpc.dial(); err != nil {
			return nil, err
		}
	}
	return rpc.mu.client, nil
}

func (rpc *EthereumRpcConnection) getFullClient() (*ethclient.Client, error) {
	if rpc.mu.fullClient == nil {
		if _, err := rpc.dialIfNeededAndReturnClient(); err != nil {
			return nil, err
		}
	}
	return rpc.mu.fullClient, nil
}

func (rpc *EthereumRpcConnection) getEthBlockByTimestamp(ctx context.Context, nano primitives.TimestampNano) (*big.Int, error) {
	client, err := rpc.getFullClient()
	if err != nil {
		return nil, err
	}

	timestampInSeconds := int64(nano) / int64(time.Second)

	latest, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, err
	}

	latestTimestamp := latest.Time.Int64()
	if latest.Time.Int64() < timestampInSeconds {
		return nil, errors.New("invalid request to get block, trying to get a block in the future (sync issues?)")
	}

	latestNumber := latest.Number.Int64()
	// a possible improvement can be instead of going back 10k blocks, assume secs/block to begin with and guess the block ts/number, but that may cause invalid calculation for older blocks
	tenKblocksAgoNumber := big.NewInt(latestNumber - 10000)
	older, err := client.HeaderByNumber(ctx, tenKblocksAgoNumber)
	if err != nil {
		return nil, err
	}

	theBlock, err := rpc.findBlockByTimeStamp(ctx, client, timestampInSeconds, latestNumber, latestTimestamp, older.Number.Int64(), older.Time.Int64())
	return theBlock, err
}
