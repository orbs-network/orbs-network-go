// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package adapter

import (
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

// A Callback function provided by consumers of blocks from storage. Each invocation receives a single blocks page
// of the requested size. Methods receiving this type will call this function
// repeatedly until it returns false to signal no more pages are required or until there are no more blocks to fetch.
type CursorFunc func(first primitives.BlockHeight, page []*protocol.BlockPairContainer) (wantsMore bool)

type BlockPersistence interface {
	WriteNextBlock(blockPair *protocol.BlockPairContainer) (bool, error)

	ScanBlocks(from primitives.BlockHeight, pageSize uint8, f CursorFunc) error

	GetLastBlockHeight() (primitives.BlockHeight, error)
	GetLastBlock() (*protocol.BlockPairContainer, error)

	GetTransactionsBlock(height primitives.BlockHeight) (*protocol.TransactionsBlockContainer, error)
	GetResultsBlock(height primitives.BlockHeight) (*protocol.ResultsBlockContainer, error)
	GetBlockByTx(txHash primitives.Sha256, minBlockTs primitives.TimestampNano, maxBlockTs primitives.TimestampNano) (block *protocol.BlockPairContainer, txIndexInBlock int, err error)

	GetBlockTracker() *synchronization.BlockTracker
}
