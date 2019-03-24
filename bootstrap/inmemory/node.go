// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package inmemory

import (
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"

	blockStorageAdapter "github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	ethereumAdapter "github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	nativeProcessorAdapter "github.com/orbs-network/orbs-network-go/services/processor/native/adapter"
	stateStorageAdapter "github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
)

// Represents an in-memory Orbs node, that uses in-memory storage and communicates with its peers via in-memory gossip
// Useful for in-process tests and simulating Orbs chains during development
type Node struct {
	index                       int
	name                        string
	config                      config.NodeConfig
	blockPersistence            blockStorageAdapter.BlockPersistence
	statePersistence            stateStorageAdapter.StatePersistence
	stateBlockHeightReporter    stateStorageAdapter.BlockHeightReporter
	transactionPoolBlockTracker *synchronization.BlockTracker // Wait() used in Network.CreateAndStartNodes()
	nativeCompiler              nativeProcessorAdapter.Compiler
	ethereumConnection          ethereumAdapter.EthereumConnection
	nodeLogic                   bootstrap.NodeLogic
	metricRegistry              metric.Registry
}

func (n *Node) GetPublicApi() services.PublicApi {
	return n.nodeLogic.PublicApi()
}

func (n *Node) Started() bool {
	return n.nodeLogic != nil
}

func (n *Node) Destroy() {
	n.nodeLogic = nil
}

func (n *Node) ExtractBlocks() ([]*protocol.BlockPairContainer, error) {

	lastBlock, err := n.blockPersistence.GetLastBlock()
	if err != nil {
		return nil, errors.Wrapf(err, "failed reading block height")
	}
	var blockPairs []*protocol.BlockPairContainer
	pageSize := uint8(lastBlock.ResultsBlock.Header.BlockHeight())
	err = n.blockPersistence.ScanBlocks(1, pageSize, func(first primitives.BlockHeight, page []*protocol.BlockPairContainer) bool {
		blockPairs = page // TODO should we copy the slice here to make sure both networks are isolated?
		return false
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed extract blocks")
	}
	return blockPairs, nil
}
