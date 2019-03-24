// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package config

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"time"
)

type NodeConfig interface {
	// shared
	ProtocolVersion() primitives.ProtocolVersion
	VirtualChainId() primitives.VirtualChainId
	NetworkType() protocol.SignerNetworkType
	NodeAddress() primitives.NodeAddress
	NodePrivateKey() primitives.EcdsaSecp256K1PrivateKey
	GenesisValidatorNodes() map[string]ValidatorNode
	GossipPeers() map[string]GossipPeer
	TransactionExpirationWindow() time.Duration

	// consensus
	ActiveConsensusAlgo() consensus.ConsensusAlgoType

	// Lean Helix consensus
	LeanHelixConsensusRoundTimeoutInterval() time.Duration
	LeanHelixConsensusMinimumCommitteeSize() uint32
	LeanHelixConsensusMaximumCommitteeSize() uint32
	LeanHelixShowDebug() bool

	// benchmark consensus
	BenchmarkConsensusRetryInterval() time.Duration
	BenchmarkConsensusRequiredQuorumPercentage() uint32
	BenchmarkConsensusConstantLeader() primitives.NodeAddress

	// block storage
	BlockSyncNumBlocksInBatch() uint32
	BlockSyncNoCommitInterval() time.Duration
	BlockSyncCollectResponseTimeout() time.Duration
	BlockSyncCollectChunksTimeout() time.Duration
	BlockStorageTransactionReceiptQueryTimestampGrace() time.Duration
	BlockStorageFileSystemDataDir() string
	BlockStorageFileSystemMaxBlockSizeInBytes() uint32

	// state storage
	StateStorageHistorySnapshotNum() uint32

	// block tracker
	BlockTrackerGraceDistance() uint32
	BlockTrackerGraceTimeout() time.Duration

	// consensus context
	ConsensusContextMaximumTransactionsInBlock() uint32
	ConsensusContextSystemTimestampAllowedJitter() time.Duration

	// transaction pool
	TransactionPoolPendingPoolSizeInBytes() uint32
	TransactionPoolFutureTimestampGraceTimeout() time.Duration
	TransactionPoolPendingPoolClearExpiredInterval() time.Duration
	TransactionPoolCommittedPoolClearExpiredInterval() time.Duration
	TransactionPoolPropagationBatchSize() uint16
	TransactionPoolPropagationBatchingTimeout() time.Duration
	TransactionPoolTimeBetweenEmptyBlocks() time.Duration
	TransactionPoolNodeSyncRejectTime() time.Duration

	// gossip
	GossipListenPort() uint16
	GossipConnectionKeepAliveInterval() time.Duration
	GossipNetworkTimeout() time.Duration

	// public api
	PublicApiSendTransactionTimeout() time.Duration
	PublicApiNodeSyncWarningTime() time.Duration

	// processor
	ProcessorArtifactPath() string
	ProcessorSanitizeDeployedContracts() bool

	// ethereum connector (crosschain)
	EthereumEndpoint() string
	EthereumFinalityTimeComponent() time.Duration
	EthereumFinalityBlocksComponent() uint32

	// logger
	LoggerHttpEndpoint() string
	LoggerBulkSize() uint32
	LoggerFileTruncationInterval() time.Duration
	LoggerFullLog() bool

	// http server
	HttpAddress() string

	// profiling
	Profiling() bool
}

type OverridableConfig interface {
	NodeConfig
	OverrideNodeSpecificValues(httpAddress string, gossipListenPort int, nodeAddress primitives.NodeAddress, nodePrivateKey primitives.EcdsaSecp256K1PrivateKey, blockStorageDataDirPrefix string) NodeConfig
	ForNode(nodeAddress primitives.NodeAddress, privateKey primitives.EcdsaSecp256K1PrivateKey) NodeConfig
}

type mutableNodeConfig interface {
	OverridableConfig
	Set(key string, value NodeConfigValue) mutableNodeConfig
	SetDuration(key string, value time.Duration) mutableNodeConfig
	SetUint32(key string, value uint32) mutableNodeConfig
	SetString(key string, value string) mutableNodeConfig
	SetBool(key string, value bool) mutableNodeConfig
	SetGenesisValidatorNodes(nodes map[string]ValidatorNode) mutableNodeConfig
	SetGossipPeers(peers map[string]GossipPeer) mutableNodeConfig
	SetNodeAddress(key primitives.NodeAddress) mutableNodeConfig
	SetNodePrivateKey(key primitives.EcdsaSecp256K1PrivateKey) mutableNodeConfig
	SetBenchmarkConsensusConstantLeader(key primitives.NodeAddress) mutableNodeConfig
	SetActiveConsensusAlgo(algoType consensus.ConsensusAlgoType) mutableNodeConfig
	MergeWithFileConfig(source string) (mutableNodeConfig, error)
	Clone() mutableNodeConfig
}

type BlockStorageConfig interface {
	NodeAddress() primitives.NodeAddress
	BlockSyncNumBlocksInBatch() uint32
	BlockSyncNoCommitInterval() time.Duration
	BlockSyncCollectResponseTimeout() time.Duration
	BlockSyncCollectChunksTimeout() time.Duration
	BlockStorageTransactionReceiptQueryTimestampGrace() time.Duration
	TransactionExpirationWindow() time.Duration
	BlockTrackerGraceTimeout() time.Duration
}

type FilesystemBlockPersistenceConfig interface {
	BlockStorageFileSystemDataDir() string
	BlockStorageFileSystemMaxBlockSizeInBytes() uint32
	VirtualChainId() primitives.VirtualChainId
}

type GossipTransportConfig interface {
	NodeAddress() primitives.NodeAddress
	GossipPeers() map[string]GossipPeer
	GossipListenPort() uint16
	GossipConnectionKeepAliveInterval() time.Duration
	GossipNetworkTimeout() time.Duration
}

// Config based on https://github.com/orbs-network/orbs-spec/blob/master/behaviors/config/services.md#consensus-context
type ConsensusContextConfig interface {
	ProtocolVersion() primitives.ProtocolVersion
	VirtualChainId() primitives.VirtualChainId
	ConsensusContextMaximumTransactionsInBlock() uint32
	GenesisValidatorNodes() map[string]ValidatorNode
	LeanHelixConsensusMinimumCommitteeSize() uint32
	ConsensusContextSystemTimestampAllowedJitter() time.Duration
}

type PublicApiConfig interface {
	PublicApiSendTransactionTimeout() time.Duration
	PublicApiNodeSyncWarningTime() time.Duration
	VirtualChainId() primitives.VirtualChainId
}

type StateStorageConfig interface {
	StateStorageHistorySnapshotNum() uint32
	BlockTrackerGraceDistance() uint32
	BlockTrackerGraceTimeout() time.Duration
}

type TransactionPoolConfig interface {
	NodeAddress() primitives.NodeAddress
	NodePrivateKey() primitives.EcdsaSecp256K1PrivateKey
	VirtualChainId() primitives.VirtualChainId
	BlockTrackerGraceDistance() uint32
	BlockTrackerGraceTimeout() time.Duration
	TransactionPoolPendingPoolSizeInBytes() uint32
	TransactionExpirationWindow() time.Duration
	TransactionPoolFutureTimestampGraceTimeout() time.Duration
	TransactionPoolPendingPoolClearExpiredInterval() time.Duration
	TransactionPoolCommittedPoolClearExpiredInterval() time.Duration
	TransactionPoolPropagationBatchSize() uint16
	TransactionPoolPropagationBatchingTimeout() time.Duration
	TransactionPoolTimeBetweenEmptyBlocks() time.Duration
	TransactionPoolNodeSyncRejectTime() time.Duration
}

type EthereumCrosschainConnectorConfig interface {
	EthereumFinalityTimeComponent() time.Duration
	EthereumFinalityBlocksComponent() uint32
}

type NativeProcessorConfig interface {
	ProcessorSanitizeDeployedContracts() bool
	VirtualChainId() primitives.VirtualChainId
}

type LeanHelixConsensusConfig interface {
	NodeAddress() primitives.NodeAddress
	NodePrivateKey() primitives.EcdsaSecp256K1PrivateKey
	LeanHelixConsensusRoundTimeoutInterval() time.Duration
	LeanHelixConsensusMaximumCommitteeSize() uint32
	LeanHelixShowDebug() bool
	ActiveConsensusAlgo() consensus.ConsensusAlgoType
	VirtualChainId() primitives.VirtualChainId
	NetworkType() protocol.SignerNetworkType
}

type ValidatorNode interface {
	NodeAddress() primitives.NodeAddress
}

type GossipPeer interface {
	GossipPort() int
	GossipEndpoint() string
}

type HttpServerConfig interface {
	HttpAddress() string
	Profiling() bool
}
