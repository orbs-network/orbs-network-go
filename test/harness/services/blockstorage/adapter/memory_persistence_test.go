package adapter

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestInMemoryBlockPersistence_GetBlocks(t *testing.T) {
	persistence := NewInMemoryBlockPersistence()

	var blocks []*protocol.BlockPairContainer

	for i := uint64(1); i <= 100; i++ {
		block := builders.BlockPair().WithHeight(primitives.BlockHeight(i)).Build()
		blocks = append(blocks, block)

		err := persistence.WriteBlock(block)
		require.NoError(t, err)
	}

	slicedBlocks, err := persistence.GetBlocks(3, 8)
	require.NoError(t, err)

	require.EqualValues(t, 6, len(slicedBlocks))
	require.EqualValues(t, 3, slicedBlocks[0].TransactionsBlock.Header.BlockHeight())
	require.EqualValues(t, 8, slicedBlocks[5].TransactionsBlock.Header.BlockHeight())
	require.EqualValues(t, blocks[2:8], slicedBlocks)

	shortSliceOfBlocks, err := persistence.GetBlocks(80, 200)
	require.NoError(t, err)

	for _, b := range blocks {
		fmt.Println(uint64(b.TransactionsBlock.Header.BlockHeight()))
	}

	require.EqualValues(t, 21, len(shortSliceOfBlocks))
	require.EqualValues(t, 80, shortSliceOfBlocks[0].TransactionsBlock.Header.BlockHeight())
	require.EqualValues(t, 100, shortSliceOfBlocks[20].TransactionsBlock.Header.BlockHeight())
}
