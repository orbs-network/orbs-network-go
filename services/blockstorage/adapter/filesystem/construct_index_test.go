// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package filesystem

import (
	"bytes"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/rand"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"io"
	"testing"
)

func TestConstructIndexFromReader(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		numBlocks := int32(17)
		ctrlRand := rand.NewControlledRand(t)
		blocksQueue := builders.RandomizedBlockChain(numBlocks, ctrlRand)
		lastBlockInChain := blocksQueue[len(blocksQueue)-1]

		rw := new(bytes.Buffer)
		codec := &mockCodec{}

		totalBytesRead := 0
		codec.When("decode", mock.Any).Call(func(r io.Reader) (*protocol.BlockPairContainer, int, error) {
			if len(blocksQueue) == 0 {
				return nil, 0, io.EOF
			}
			randBlockSize := ctrlRand.Intn(500) + 1
			totalBytesRead += randBlockSize
			block, bytes := blocksQueue[0], randBlockSize
			blocksQueue = blocksQueue[1:]
			return block, bytes, nil
		})

		metrics := generateBlockStorageMetrics(metric.NewRegistry())
		blockHeightIndex, err := buildIndex(rw, 0, harness.Logger, codec, metrics)

		require.NoError(t, err, "expected index to construct with no error")
		require.EqualValues(t, numBlocks, blockHeightIndex.getLastBlockHeight(), "expected index to reach topHeight block height")
		test.RequireCmpEqual(t, blockHeightIndex.getLastBlock(), lastBlockInChain, "expected index to cache last block")
		require.EqualValues(t, totalBytesRead, blockHeightIndex.fetchNextOffset(), "expected next block offset to be the buffer size")
	})

}

// constructing index from block stream with mixed blockHeight order: 1-10, 20-11, 21-40, 100-41
func TestConstructIndexFromReaderInterleavedOrder(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		numBlocks := int32(100)
		ctrlRand := rand.NewControlledRand(t)
		const maxTransactions = 20
		const maxStateDiffs = 20
		blocksQueue := make([]*protocol.BlockPairContainer, 0, numBlocks)
		blocks := builders.RandomizedBlockChain(numBlocks, ctrlRand)

		blocksQueue = append(blocksQueue, blocks[0:10]...)
		for i := 19; i >= 10; i-- {
			blocksQueue = append(blocksQueue, blocks[i])
		}
		blocksQueue = append(blocksQueue, blocks[20:40]...)
		for i := 99; i >= 40; i-- {
			blocksQueue = append(blocksQueue, blocks[i])
		}

		lastBlockInChain := blocks[len(blocks)-1]

		rw := new(bytes.Buffer)
		codec := &mockCodec{}

		totalBytesRead := 0
		codec.When("decode", mock.Any).Call(func(r io.Reader) (*protocol.BlockPairContainer, int, error) {
			if len(blocksQueue) == 0 {
				return nil, 0, io.EOF
			}
			randBlockSize := ctrlRand.Intn(500) + 1
			totalBytesRead += randBlockSize
			block, bytes := blocksQueue[0], randBlockSize
			blocksQueue = blocksQueue[1:]
			return block, bytes, nil
		})

		metrics := generateBlockStorageMetrics(metric.NewRegistry())
		blockHeightIndex, err := buildIndex(rw, 0, harness.Logger, codec, metrics)

		require.NoError(t, err, "expected index to construct with no error")
		require.EqualValues(t, numBlocks, blockHeightIndex.getLastBlockHeight(), "expected index to reach topHeight block height")
		test.RequireCmpEqual(t, blockHeightIndex.getLastBlock(), lastBlockInChain, "expected index to cache last block")
		test.RequireCmpEqual(t, blockHeightIndex.topBlock, blockHeightIndex.sequentialTopBlock, "expected index topBlock to match sequentialTopBlock")
		test.RequireCmpEqual(t, blockHeightIndex.topBlock, blockHeightIndex.lastWrittenBlock, "expected index topBlock to match lastWrittenBlock")
		require.EqualValues(t, totalBytesRead, blockHeightIndex.fetchNextOffset(), "expected next block offset to be the buffer size")
	})

}

// constructing index from block stream with mixed blockHeight order and a gap: 1-10, 100-90
// blockHeightIndex indices should depict the following state: {top:100, lastWritten:90, sequentialTop: 10}
func TestConstructIndexFromReaderInterleavedOrderWithGap(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		numBlocks := int32(100)
		ctrlRand := rand.NewControlledRand(t)
		const maxTransactions = 20
		const maxStateDiffs = 20
		blocksQueue := make([]*protocol.BlockPairContainer, 0, numBlocks)
		blocks := builders.RandomizedBlockChain(numBlocks, ctrlRand)

		blocksQueue = append(blocksQueue, blocks[0:10]...)
		for i := 99; i >= 89; i-- {
			blocksQueue = append(blocksQueue, blocks[i])
		}

		sequentialTopBlock := blocks[9]
		topBlock := blocks[99]
		lastWrittenBlock := blocks[89]

		rw := new(bytes.Buffer)
		codec := &mockCodec{}

		totalBytesRead := 0
		codec.When("decode", mock.Any).Call(func(r io.Reader) (*protocol.BlockPairContainer, int, error) {
			if len(blocksQueue) == 0 {
				return nil, 0, io.EOF
			}
			randBlockSize := ctrlRand.Intn(500) + 1
			totalBytesRead += randBlockSize
			block, bytes := blocksQueue[0], randBlockSize
			blocksQueue = blocksQueue[1:]
			return block, bytes, nil
		})

		metrics := generateBlockStorageMetrics(metric.NewRegistry())
		blockHeightIndex, err := buildIndex(rw, 0, harness.Logger, codec, metrics)

		require.NoError(t, err, "expected index to construct with no error")
		require.EqualValues(t, getBlockHeight(sequentialTopBlock), blockHeightIndex.getLastBlockHeight(), "expected index to reach sequential top height")
		require.EqualValues(t, getBlockHeight(blockHeightIndex.sequentialTopBlock), primitives.BlockHeight(10), "expected index sequentialTopBlock height to be 10")
		test.RequireCmpEqual(t, blockHeightIndex.getLastBlock(), sequentialTopBlock, "expected index to cache sequentialTopBlock")
		require.EqualValues(t, getBlockHeight(blockHeightIndex.topBlock), primitives.BlockHeight(100), "expected index topBlock height to be 100")
		test.RequireCmpEqual(t, blockHeightIndex.topBlock, topBlock, "expected index topBlock to match topBlock")
		require.EqualValues(t, getBlockHeight(blockHeightIndex.lastWrittenBlock), primitives.BlockHeight(90), "expected index lastWrittenBlock height to be 90")
		test.RequireCmpEqual(t, blockHeightIndex.lastWrittenBlock, lastWrittenBlock, "expected index lastWrittenBlock to match lastWrittenBlock")
		require.EqualValues(t, totalBytesRead, blockHeightIndex.fetchNextOffset(), "expected next block offset to be the buffer size")
	})
}

func newBlockFileReadStream(t *testing.T, ctrlRand *rand.ControlledRand, numBlocks int32, maxTransactions uint32, maxStateDiffs uint32, codec *codec) (io.Reader, chan int) {
	blocksQueue := builders.RandomizedBlockChainWithLimit(numBlocks, ctrlRand, maxTransactions, maxStateDiffs)

	pr, pw := io.Pipe()

	done := make(chan int)
	go func() {
		for _, blockPair := range blocksQueue {
			_, err := codec.encode(blockPair, pw)
			require.NoError(t, err, "expected codec to successfully encode a block pair")
		}
		_ = pw.Close()
		close(done)
	}()

	return pr, done
}

func OneByteAtATimeReader(t *testing.T, r io.Reader) (io.Reader, chan int) {
	pr, pw := io.Pipe()

	done := make(chan int)
	go func() {
		defer pw.Close()
		defer close(done)

		b := make([]byte, 1)

		var err error
		var n int
		for err == nil {
			n, err = io.ReadFull(r, b)
			if err == io.EOF {
				break
			}
			require.NoError(t, err, "expected a byte from the stream or EOF")
			require.Equal(t, 1, n, "expected exactly one byte from the stream")

			n, err = pw.Write(b)
			require.NoError(t, err, "Expected a successful write")
			require.Equal(t, 1, n, "expected exactly one byte to be written")
		}
	}()

	return pr, done
}

func TestBuildIndexSucceedsIndexingFromReader(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		const numBlocks = 17
		const maxTransactions = 200
		const maxStateDiffs = 200

		ctrlRand := rand.NewControlledRand(t)
		codec := newCodec(1024 * 1024)
		r, done := newBlockFileReadStream(t, ctrlRand, numBlocks, maxTransactions, maxStateDiffs, codec)

		metrics := generateBlockStorageMetrics(metric.NewRegistry())
		bhIndex, err := buildIndex(r, 0, harness.Logger, codec, metrics)

		require.NoError(t, err, "expected buildIndex to succeed")
		require.Equal(t, bhIndex.getLastBlockHeight(), primitives.BlockHeight(numBlocks), "expected block height to match the encoded block count")

		<-done
	})
}

// The purpose of this test is to assure that buildIndex handles the case where a reader returns less bytes than
// requested, even when more will be available in a subsequent read.
// (this is the behaviour of the buffered reader we use for reading the block file)
// To test this, we wrap the file reader with a reader that only returns one byte at a time.
func TestBuildIndexHandlesPartialReads(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		// This test reads the file one byte at a time which is slow, so we use a small chain
		const numBlocks = 2
		const maxTransactions = 30
		const maxStateDiffs = 30

		ctrlRand := rand.NewControlledRand(t)
		codec := newCodec(1024 * 1024)
		r, done := newBlockFileReadStream(t, ctrlRand, numBlocks, maxTransactions, maxStateDiffs, codec)

		rBuffered, done2 := OneByteAtATimeReader(t, r)
		metrics := generateBlockStorageMetrics(metric.NewRegistry())
		bhIndex, err := buildIndex(rBuffered, 0, harness.Logger, codec, metrics)

		require.NoError(t, err, "expected buildIndex to succeed with a buffered reader")
		require.Equal(t, bhIndex.getLastBlockHeight(), primitives.BlockHeight(numBlocks), "expected block height to match the encoded block count")

		<-done
		<-done2
	})
}

func TestBuildIndexHandlesEmptyFile(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		codec := newCodec(1024 * 1024)

		r := bytes.NewReader(make([]byte, 0, 0))
		metrics := generateBlockStorageMetrics(metric.NewRegistry())
		bhIndex, err := buildIndex(r, 0, harness.Logger, codec, metrics)

		require.NoError(t, err, "expected buildIndex to succeed")
		require.Equal(t, bhIndex.getLastBlockHeight(), primitives.BlockHeight(0), "expected block height to be zero")
	})
}

type mockCodec struct {
	mock.Mock
}

func (mc *mockCodec) encode(block *protocol.BlockPairContainer, w io.Writer) (int, error) {
	ret := mc.Called(block, w)
	return ret.Int(0), ret.Error(1)
}

func (mc *mockCodec) decode(r io.Reader) (*protocol.BlockPairContainer, int, error) {
	ret := mc.Called(r)
	return ret.Get(0).(*protocol.BlockPairContainer), ret.Int(1), ret.Error(2)
}
