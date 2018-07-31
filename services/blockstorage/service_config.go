package blockstorage

import "time"

type BlockStorageConfig interface {
	BlockSyncCommitTimeout() time.Duration
	TransactionSearchGrace() time.Duration
	QuerySyncGraceBlockNum() int
	MaxBlocksPerSyncBatch() int
}

type config struct {
	blockSyncCommitTimeout time.Duration
	transactionSearchGrace time.Duration
	querySyncGraceBlockNum int
	maxBlocksPerSyncBatch  int
}

func (c *config) BlockSyncCommitTimeout() time.Duration {
	return c.blockSyncCommitTimeout
}

func (c *config) TransactionSearchGrace() time.Duration {
	return c.transactionSearchGrace
}

func (c *config) QuerySyncGraceBlockNum() int {
	return c.querySyncGraceBlockNum
}

func (c *config) MaxBlocksPerSyncBatch() int {
	return c.maxBlocksPerSyncBatch
}

func DefaultBlockStorageConfig() BlockStorageConfig {
	return &config{
		blockSyncCommitTimeout: 8 * time.Second,
		transactionSearchGrace: 5 * time.Second,
		querySyncGraceBlockNum: 0,
		maxBlocksPerSyncBatch:  10000,
	}
}

func NewBlockStorageConfig(blockSyncCommitTimeout time.Duration, transactionSearchGrace time.Duration,
	querySyncGraceBlockNum int, maxBlocksPerSyncBatch int) BlockStorageConfig {
	return &config{
		blockSyncCommitTimeout,
		transactionSearchGrace,
		querySyncGraceBlockNum,
		maxBlocksPerSyncBatch,
	}
}
