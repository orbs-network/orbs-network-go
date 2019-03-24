// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package main

import (
	"flag"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter/test"
	testUtils "github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"time"
)

// tool to generate large blocks files. intended to be used internally for benchmark tests and performance/load
// the blocks generated here do not go through consensus. they simulate some state diffs, tx and receipts, but will
// may not be successfully validated as blocks under consensus
func main() {
	dir, virtualChain, targetHeight, randomizeEach := parseParams()

	logger := adHocLogger("")
	rand := testUtils.NewControlledRand(&logger)
	conf := &randomChainConfig{dir: dir, virtualChainId: virtualChain}

	start := time.Now()
	fmt.Printf("\nusing:\noutput directory: %s\nvirtual chain id: %d\n\nloading adapter and building index...\n", conf.BlockStorageFileSystemDataDir(), conf.VirtualChainId())

	adapter, release, err := test.NewFilesystemAdapterDriver(log.GetLogger(), conf)
	if err != nil {
		panic(err)
	}
	defer release()

	currentHeight, err := adapter.GetLastBlockHeight()
	if err != nil {
		panic(err)
	}
	prevBlock, err := adapter.GetLastBlock()
	if err != nil {
		panic(err)
	}

	fmt.Printf("indexed %d blocks in %v\n", currentHeight, time.Now().Sub(start))
	block := builders.RandomizedBlock(0, rand, prevBlock)

	for currentHeight < targetHeight {
		nextHeight := currentHeight + 1

		if randomizeEach {
			block = builders.RandomizedBlock(nextHeight, rand, prevBlock)
		} else {
			_ = block.ResultsBlock.Header.MutateBlockHeight(nextHeight)
			_ = block.TransactionsBlock.Header.MutateBlockHeight(nextHeight)
		}

		_, err := adapter.WriteNextBlock(block)
		if err != nil {
			logger.Log("error writing block to file at height %d. error %s", nextHeight, err)
			panic(err)
		}
		if nextHeight%1000 == 0 {
			logger.Log(fmt.Sprintf("wrote height %d", nextHeight))
		}

		currentHeight++
		prevBlock = block
	}

	fmt.Printf("\n\nblocks file in %s/ now has %d blocks\n\n", conf.BlockStorageFileSystemDataDir(), currentHeight)
}

func parseParams() (dir string, vchain primitives.VirtualChainId, height primitives.BlockHeight, randomEach bool) {
	intHeight := flag.Uint64("height", 100, "target height for blocks file")
	outputDir := flag.String("output", "./gen_data", "target directory for new block file")
	virtualChain := flag.Uint("vchain", 42, "blocks file virtual chain id")
	rand := flag.Bool("full_random", false, "generate a different random block for each block height")
	flag.Parse()
	fmt.Printf("usage: [-output output_folder_name] [-height target_block_height] [-vchain vchain_id] [-full_random]\n\n")
	targetHeight := primitives.BlockHeight(*intHeight)
	return *outputDir, primitives.VirtualChainId(*virtualChain), targetHeight, *rand
}

type randomChainConfig struct {
	dir            string
	virtualChainId primitives.VirtualChainId
}

func (l *randomChainConfig) VirtualChainId() primitives.VirtualChainId {
	return l.virtualChainId
}

func (l *randomChainConfig) BlockStorageFileSystemDataDir() string {
	return l.dir
}

func (l *randomChainConfig) BlockStorageFileSystemMaxBlockSizeInBytes() uint32 {
	return 1000000000
}

type adHocLogger string

func (l *adHocLogger) Log(args ...interface{}) {
	fmt.Println(args...)
}
func (l *adHocLogger) Name() string {
	return string(*l)
}
