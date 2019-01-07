package adapter

import (
	"bytes"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"io"
	"testing"
)

func TestConstructIndexFromReader(t *testing.T) {
	numBlocks := int32(17)
	ctrlRand := test.NewControlledRand(t)
	blocks := builders.RandomizedBlockChain(numBlocks, ctrlRand)

	rw := new(bytes.Buffer)
	codec := &mockCodec{}

	// init simulated block sizes
	blocksSizes := make([]byte, len(blocks))
	_, _ = ctrlRand.Read(blocksSizes)
	totalSize := 0
	for _, s := range blocksSizes {
		totalSize += int(s)
	}

	currentBlockIdx := 0
	codec.When("decode", mock.Any).Call(func(r io.Reader) (*protocol.BlockPairContainer, int, error) {
		if currentBlockIdx > len(blocks)-1 {
			return nil, 0, io.EOF
		}
		block, bytes := blocks[currentBlockIdx], int(blocksSizes[currentBlockIdx])
		currentBlockIdx++
		return block, bytes, nil
	})

	blockHeightIndex, err := constructIndexFromReader(rw, log.GetLogger(), codec)

	require.NoError(t, err)
	require.EqualValues(t, numBlocks, blockHeightIndex.topBlockHeight)
	test.RequireCmpEqual(t, blockHeightIndex.topBlock, blocks[len(blocks)-1])
	require.EqualValues(t, totalSize, blockHeightIndex.heightOffset[primitives.BlockHeight(len(blocks))+1])

}

type mockCodec struct {
	mock.Mock
}

func (mc *mockCodec) encode(block *protocol.BlockPairContainer, w io.Writer) error {
	return mc.Called(block, w).Error(0)
}

func (mc *mockCodec) decode(r io.Reader) (*protocol.BlockPairContainer, int, error) {
	ret := mc.Called(r)
	return ret.Get(0).(*protocol.BlockPairContainer), ret.Int(1), ret.Error(2)
}
