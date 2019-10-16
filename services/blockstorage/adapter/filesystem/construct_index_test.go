// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package filesystem

import (
	"bufio"
	"bytes"
	"github.com/orbs-network/go-mock"
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

		blockHeightIndex, err := buildIndex(rw, 0, harness.Logger, codec)

		require.NoError(t, err, "expected index to construct with no error")
		require.EqualValues(t, numBlocks, blockHeightIndex.topBlockHeight, "expected index to reach top block height")
		test.RequireCmpEqual(t, blockHeightIndex.topBlock, lastBlockInChain, "expected index to cache last block")
		require.EqualValues(t, totalBytesRead, blockHeightIndex.heightOffset[primitives.BlockHeight(numBlocks)+1], "expected next block offset to be the buffer size")
	})

}

func newBlockFileReadStream(t *testing.T, ctrlRand *rand.ControlledRand, numBlocks int32, codec *codec) (io.Reader, chan int) {
	blocksQueue := builders.RandomizedBlockChain(numBlocks, ctrlRand)

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

func TestBuildIndexSucceedsIndexingFromReader(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		const numBlocks = 17

		ctrlRand := rand.NewControlledRand(t)
		codec := newCodec(1024 * 1024)
		r, done := newBlockFileReadStream(t, ctrlRand, numBlocks, codec)

		bhIndex, err := buildIndex(r, 0, harness.Logger, codec)

		require.NoError(t, err, "expected buildIndex to succeed")
		require.Equal(t, bhIndex.topBlockHeight, primitives.BlockHeight(numBlocks), "expected block height to match the encoded block count")

		<-done
	})
}

func TestBuildIndexHandlesSmallBufferedReader(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		const numBlocks = 17

		ctrlRand := rand.NewControlledRand(t)
		codec := newCodec(1024 * 1024)
		r, done := newBlockFileReadStream(t, ctrlRand, numBlocks, codec)

		rBuffered := bufio.NewReaderSize(r, 1)
		bhIndex, err := buildIndex(rBuffered, 0, harness.Logger, codec)

		require.NoError(t, err, "expected buildIndex to succeed with a buffered reader")
		require.Equal(t, bhIndex.topBlockHeight, primitives.BlockHeight(numBlocks), "expected block height to match the encoded block count")

		<-done
	})
}

func TestBuildIndexHandlesEmptyFile(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		codec := newCodec(1024 * 1024)

		r := bytes.NewReader(make([]byte, 0, 0))
		bhIndex, err := buildIndex(r, 0, harness.Logger, codec)

		require.NoError(t, err, "expected buildIndex to succeed")
		require.Equal(t, bhIndex.topBlockHeight, primitives.BlockHeight(0), "expected block height to be zero")
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
