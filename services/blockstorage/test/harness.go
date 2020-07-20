// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"fmt"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/blockstorage"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter/testkit"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/internodesync"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

type configForBlockStorageTests struct {
	nodeAddress              primitives.NodeAddress
	syncBatchSize            uint32
	syncNoCommit             time.Duration
	syncCollectResponses     time.Duration
	syncCollectChunks        time.Duration
	committeeValidityTimeout time.Duration
	syncBlocksOrder          gossipmessages.SyncBlocksOrder
	descendingEnabled        bool
	queryGrace               time.Duration
	queryExpirationWindow    time.Duration
	blockTrackerGrace        time.Duration
}

func (c *configForBlockStorageTests) NodeAddress() primitives.NodeAddress {
	return c.nodeAddress
}

func (c *configForBlockStorageTests) BlockSyncNumBlocksInBatch() uint32 {
	return c.syncBatchSize
}

func (c *configForBlockStorageTests) BlockSyncNoCommitInterval() time.Duration {
	return c.syncNoCommit
}

func (c *configForBlockStorageTests) BlockSyncCollectResponseTimeout() time.Duration {
	return c.syncCollectResponses
}

func (c *configForBlockStorageTests) BlockSyncCollectChunksTimeout() time.Duration {
	return c.syncCollectChunks
}

func (c *configForBlockStorageTests) CommitteeValidityTimeout() time.Duration {
	return c.committeeValidityTimeout
}

func (c *configForBlockStorageTests) BlockSyncDescendingEnabled() bool {
	return c.descendingEnabled
}

func (c *configForBlockStorageTests) BlockSyncBlocksOrder() gossipmessages.SyncBlocksOrder {
	return c.syncBlocksOrder
}

func (c *configForBlockStorageTests) BlockStorageTransactionReceiptQueryTimestampGrace() time.Duration {
	return c.queryGrace
}

func (c *configForBlockStorageTests) TransactionExpirationWindow() time.Duration {
	return c.queryExpirationWindow
}

func (c *configForBlockStorageTests) BlockTrackerGraceTimeout() time.Duration {
	return c.blockTrackerGrace
}

type harness struct {
	*with.ConcurrencyHarness

	sync.Mutex
	stateStorage   *services.MockStateStorage
	storageAdapter testkit.TamperingInMemoryBlockPersistence
	blockStorage   *blockstorage.Service
	consensus      *handlers.MockConsensusBlocksHandler
	gossip         *gossiptopics.MockBlockSync
	txPool         *services.MockTransactionPool
	config         *configForBlockStorageTests
}

func (d *harness) withSyncBroadcast(times int) *harness {
	d.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any, mock.Any).Return(nil, nil).Times(times)
	return d
}

func (d *harness) withCommitStateDiff(times int) *harness {
	d.stateStorage.When("CommitStateDiff", mock.Any, mock.Any).Call(func(ctx context.Context, input *services.CommitStateDiffInput) (*services.CommitStateDiffOutput, error) {
		return &services.CommitStateDiffOutput{
			NextDesiredBlockHeight: input.ResultsBlockHeader.BlockHeight() + 1,
		}, nil
	}).Times(times)
	return d
}

func (d *harness) withValidateConsensusAlgos(times int) *harness {
	out := &handlers.HandleBlockConsensusOutput{}

	d.consensus.When("HandleBlockConsensus", mock.Any, mock.Any).Return(out, nil).Times(times)
	return d
}

func (d *harness) expectValidateConsensusAlgos() *harness {
	out := &handlers.HandleBlockConsensusOutput{}

	d.consensus.When("HandleBlockConsensus", mock.Any, mock.Any).Return(out, nil).AtLeast(0)
	return d
}

func (d *harness) expectCommitStateDiffTimes(times int) {
	csdOut := &services.CommitStateDiffOutput{}

	d.stateStorage.When("CommitStateDiff", mock.Any, mock.Any).Return(csdOut, nil).Times(times)
}

func (d *harness) verifyMocksConsistently(t *testing.T, times int) {
	err := test.ConsistentlyVerify(test.EVENTUALLY_ACCEPTANCE_TIMEOUT*time.Duration(times), d.gossip, d.stateStorage, d.consensus)
	require.NoError(t, err)
}

func (d *harness) verifyMocks(t *testing.T, times int) {
	err := test.EventuallyVerify(test.EVENTUALLY_ACCEPTANCE_TIMEOUT*time.Duration(times), d.gossip, d.stateStorage, d.consensus)
	require.NoError(t, err)
}

func (d *harness) commitBlock(ctx context.Context, blockPairContainer *protocol.BlockPairContainer) (*services.CommitBlockOutput, error) {
	return d.blockStorage.CommitBlock(ctx, &services.CommitBlockInput{
		BlockPair: blockPairContainer,
	})
}

func (d *harness) numOfWrittenBlocks() int {
	numBlocks, err := d.storageAdapter.GetLastBlockHeight()
	if err != nil {
		panic(fmt.Sprintf("failed getting last block height, err=%s", err.Error()))
	}
	return int(numBlocks)
}

func (d *harness) getLastBlockHeight(ctx context.Context, t *testing.T) *services.GetLastCommittedBlockHeightOutput {
	out, err := d.blockStorage.GetLastCommittedBlockHeight(ctx, &services.GetLastCommittedBlockHeightInput{})

	require.NoError(t, err)
	return out
}

func (d *harness) getBlock(height int) *protocol.BlockPairContainer {
	txBlock, err := d.storageAdapter.GetTransactionsBlock(primitives.BlockHeight(height))
	if err != nil {
		panic(fmt.Sprintf("failed getting tx block, err=%s", err.Error()))
	}

	rxBlock, err := d.storageAdapter.GetResultsBlock(primitives.BlockHeight(height))
	if err != nil {
		panic(fmt.Sprintf("failed getting results block, err=%s", err.Error()))
	}

	return &protocol.BlockPairContainer{
		TransactionsBlock: txBlock,
		ResultsBlock:      rxBlock,
	}
}

func (d *harness) withSyncNoCommitTimeout(duration time.Duration) *harness {
	d.config.syncNoCommit = duration
	return d
}

func (d *harness) withSyncCollectResponsesTimeout(duration time.Duration) *harness {
	d.config.syncCollectResponses = duration
	return d
}

func (d *harness) withSyncCollectChunksTimeout(duration time.Duration) *harness {
	d.config.syncCollectChunks = duration
	return d
}

func (d *harness) withBlockSyncDescendingEnabled(isEnabled bool) *harness {
	d.config.descendingEnabled = isEnabled
	return d
}

func (d *harness) withBatchSize(size uint32) *harness {
	d.config.syncBatchSize = size
	return d
}

func (d *harness) withBlockStorageTransactionReceiptQueryTimestampGrace(value time.Duration) *harness {
	d.config.queryGrace = value
	return d
}

func (d *harness) withTransactionExpirationWindow(value time.Duration) *harness {
	d.config.queryExpirationWindow = value
	return d
}

func (d *harness) withNodeAddress(address primitives.NodeAddress) *harness {
	d.config.nodeAddress = address
	return d
}

func (d *harness) failNextBlocks() {
	d.storageAdapter.TamperWithBlockWrites(nil)
}

func (d *harness) commitSomeBlocks(ctx context.Context, count int) {
	for i := 1; i <= count; i++ {
		_, _ = d.commitBlock(ctx, builders.BlockPair().WithHeight(primitives.BlockHeight(i)).WithBlockCreated(time.Now()).Build())
	}
}

func (d *harness) setupCustomBlocksForInit() time.Time {
	now := time.Now()
	for i := 1; i <= 10; i++ {
		now = now.Add(1 * time.Millisecond)
		_, _, _ = d.storageAdapter.WriteNextBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(i)).WithBlockCreated(now).Build())
	}

	return now
}

func createConfig(nodeAddress primitives.NodeAddress) *configForBlockStorageTests {
	cfg := &configForBlockStorageTests{}
	cfg.nodeAddress = nodeAddress
	cfg.syncBatchSize = 2
	cfg.syncNoCommit = 30 * time.Second // setting a long time here so sync never starts during the tests
	cfg.syncCollectResponses = 5 * time.Millisecond
	cfg.syncCollectChunks = 20 * time.Millisecond
	cfg.descendingEnabled = true
	cfg.committeeValidityTimeout = 1 * time.Minute

	cfg.queryGrace = 5 * time.Second
	cfg.queryExpirationWindow = 30 * time.Minute
	cfg.blockTrackerGrace = 1 * time.Hour

	return cfg
}

func newBlockStorageHarness(parentHarness *with.ConcurrencyHarness) *harness {
	keyPair := keys.EcdsaSecp256K1KeyPairForTests(0)
	cfg := createConfig(keyPair.NodeAddress())

	registry := metric.NewRegistry()
	d := &harness{config: cfg, ConcurrencyHarness: parentHarness}
	d.stateStorage = &services.MockStateStorage{}
	d.storageAdapter = testkit.NewBlockPersistence(d.Logger, nil, registry)

	d.consensus = &handlers.MockConsensusBlocksHandler{}

	d.gossip = &gossiptopics.MockBlockSync{}
	d.gossip.When("RegisterBlockSyncHandler", mock.Any).Return().Times(1)

	d.txPool = &services.MockTransactionPool{}
	d.txPool.When("CommitTransactionReceipts", mock.Any, mock.Any).Call(func(ctx context.Context, input *services.CommitTransactionReceiptsInput) (*services.CommitTransactionReceiptsOutput, error) {
		return &services.CommitTransactionReceiptsOutput{
			NextDesiredBlockHeight: input.ResultsBlockHeader.BlockHeight() + 1,
		}, nil
	}).AtLeast(0)

	return d
}

func (d *harness) allowingErrorsMatching(pattern string) *harness {
	d.AllowErrorsMatching(pattern)
	return d
}

func (d *harness) start(ctx context.Context) *harness {
	d.Lock()
	defer d.Unlock()
	registry := metric.NewRegistry()

	d.blockStorage = blockstorage.NewBlockStorage(ctx, d.config, d.storageAdapter, d.gossip, d.Logger, registry, nil)
	d.blockStorage.RegisterConsensusBlocksHandler(d.consensus)

	d.Supervise(d.blockStorage)

	return d
}

func respondToBroadcastAvailabilityRequest(ctx context.Context, harness *harness, requestInput *gossiptopics.BlockAvailabilityRequestInput, availableBlocks primitives.BlockHeight, sources ...int) {
	harness.Lock()
	defer harness.Unlock()

	if harness.blockStorage == nil {
		return // protect against edge condition where harness did not finish initializing and sync has started
	}

	blocksOrder := requestInput.Message.SignedBatchRange.BlocksOrder()
	fromBlock := requestInput.Message.SignedBatchRange.FirstBlockHeight()
	toBlock := requestInput.Message.SignedBatchRange.LastBlockHeight()

	if blocksOrder == gossipmessages.SYNC_BLOCKS_ORDER_DESCENDING {
		if toBlock > availableBlocks {
			return
		}
		if fromBlock == internodesync.UNKNOWN_BLOCK_HEIGHT {
			fromBlock = availableBlocks
		}
	} else {
		if fromBlock > availableBlocks {
			return
		}
		toBlock = availableBlocks
	}

	for _, sourceAddressIndex := range sources {
		response := builders.BlockAvailabilityResponseInput().
			WithLastCommittedBlockHeight(availableBlocks).
			WithFirstBlockHeight(fromBlock).
			WithLastBlockHeight(toBlock).
			WithSenderNodeAddress(keys.EcdsaSecp256K1KeyPairForTests(sourceAddressIndex).NodeAddress()).
			WithBlocksOrder(blocksOrder).Build()
		go harness.blockStorage.HandleBlockAvailabilityResponse(ctx, response)
	}

}

func reverse(arr []*protocol.BlockPairContainer) {
	for i, j := 0, len(arr)-1; i < j; i, j = i+1, j-1 {
		arr[i], arr[j] = arr[j], arr[i]
	}
}

func createBlockSyncResponse(input *gossiptopics.BlockSyncRequestInput, blockChain []*protocol.BlockPairContainer, batchSize uint32) *gossiptopics.BlockSyncResponseInput {
	blocksOrder := input.Message.SignedChunkRange.BlocksOrder()
	fromBlock := input.Message.SignedChunkRange.FirstBlockHeight()
	toBlock := input.Message.SignedChunkRange.LastBlockHeight()
	availableBlocks := len(blockChain)
	blockChainCopy := make([]*protocol.BlockPairContainer, availableBlocks)
	copy(blockChainCopy, blockChain)
	var blocks []*protocol.BlockPairContainer

	if blocksOrder == gossipmessages.SYNC_BLOCKS_ORDER_DESCENDING {
		if fromBlock == internodesync.UNKNOWN_BLOCK_HEIGHT {
			fromBlock = primitives.BlockHeight(availableBlocks)
		}
		if toBlock > primitives.BlockHeight(availableBlocks) {
			return nil
		}
		// limit batch size server
		if (fromBlock+1 > primitives.BlockHeight(batchSize)) && (fromBlock+1-primitives.BlockHeight(batchSize) > toBlock) {
			toBlock = fromBlock + 1 - primitives.BlockHeight(batchSize)
		}
		blocks = blockChainCopy[toBlock-1 : fromBlock]
		reverse(blocks)

	} else {
		blocks = blockChain[fromBlock-1 : toBlock]
	}
	response := builders.BlockSyncResponseInput().
		WithFirstBlockHeight(fromBlock).
		WithLastBlockHeight(toBlock).
		WithLastCommittedBlockHeight(primitives.BlockHeight(availableBlocks)).
		WithBlocksOrder(input.Message.SignedChunkRange.BlocksOrder()).
		WithSenderNodeAddress(input.RecipientNodeAddress).
		WithBlocks(blocks).Build()

	return response
}

func respondToBlockSyncRequest(ctx context.Context, harness *harness, input *gossiptopics.BlockSyncRequestInput, blockChain []*protocol.BlockPairContainer, batchSize uint32) {
	response := createBlockSyncResponse(input, blockChain, batchSize)
	if response != nil {
		go harness.blockStorage.HandleBlockSyncResponse(ctx, response)
	}
}
