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

type EthBlock struct {
	number    int64
	timestamp int64
}

type EthBlocksSearchCache struct {
	latest  EthBlock
	back10k EthBlock
}

type EthereumRpcConnection struct {
	connectorCommon

	config ethereumAdapterConfig

	mu struct {
		sync.Mutex
		client      EthereumCaller
		fullClient  *ethclient.Client
		blocksCache *EthBlocksSearchCache
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

func (rpc *EthereumRpcConnection) refreshBlocksCache(ctx context.Context) error {
	rpc.logger.Info("refreshing eth blocks cache")

	client, err := rpc.getFullClient()
	if err != nil {
		return err
	}

	rpc.mu.Lock()
	defer rpc.mu.Unlock()

	if rpc.mu.blocksCache == nil {
		rpc.mu.blocksCache = &EthBlocksSearchCache{}
	}

	latest, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return err
	}

	rpc.mu.blocksCache.latest.timestamp = latest.Time.Int64()
	rpc.mu.blocksCache.latest.number = latest.Number.Int64()

	older, err := client.HeaderByNumber(ctx, latest.Number.Sub(latest.Number, big.NewInt(10000)))
	if err != nil {
		return err
	}

	rpc.mu.blocksCache.back10k.timestamp = older.Time.Int64()
	rpc.mu.blocksCache.back10k.number = older.Number.Int64()

	return nil
}

func (rpc *EthereumRpcConnection) getBlocksCacheAndRefreshIfNeeded(ctx context.Context, unixEpoch int64) (*EthBlocksSearchCache, error) {
	// TODO: (v1) move cache refresh timeout to config
	if rpc.mu.blocksCache == nil || rpc.mu.blocksCache.latest.timestamp < unixEpoch-1000 {
		if err := rpc.refreshBlocksCache(ctx); err != nil {
			return nil, err
		}
	}

	return rpc.mu.blocksCache, nil
}

func (rpc *EthereumRpcConnection) getEthBlockByTimestamp(ctx context.Context, nano primitives.TimestampNano) (*big.Int, error) {
	timestampInSeconds := int64(nano) / int64(time.Second)
	// ethereum started around 2015/07/31
	if timestampInSeconds < 1438300800 {
		return nil, errors.New("cannot query before ethereum genesis")
	}

	client, err := rpc.getFullClient()
	if err != nil {
		return nil, err
	}

	cache, err := rpc.getBlocksCacheAndRefreshIfNeeded(ctx, timestampInSeconds)
	if err != nil {
		return nil, err
	}

	// TODO: (v1) move cache refresh timeout to config
	if cache.latest.timestamp+1000 < timestampInSeconds {
		return nil, errors.New("invalid request to get block, trying to get a block in the future (sync issues?)")
	}

	theBlock, err := rpc.findBlockByTimeStamp(ctx, client, timestampInSeconds, cache.latest.number, cache.latest.timestamp, cache.back10k.number, cache.back10k.timestamp)
	return theBlock, err
}
