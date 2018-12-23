package adapter

import (
	"bytes"
	"encoding/binary"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestEncodesAndDecodes(t *testing.T) {
	block := builders.BlockPair().WithHeight(1).Build()
	rw := new(bytes.Buffer)

	err := encode(block, rw)
	require.NoError(t, err)
	decodedBlock, err := decode(rw)

	require.NoError(t, err)
	test.RequireCmpEqual(t, block, decodedBlock)
}

func TestEncodeHeader(t *testing.T) {
	rw := new(bytes.Buffer)
	header := &blockHeader{
		FixedSize:    1,
		ReceiptsSize: 2,
		DiffsSize:    3,
		TxsSize:      4,
	}
	err := header.write(rw)
	require.NoError(t, err)

	bytes := rw.Bytes()
	require.Len(t, bytes, 4*4)

	decodedHeader := &blockHeader{}
	err = decodedHeader.read(rw)
	require.NoError(t, err)

	require.EqualValues(t, header, decodedHeader)

	// enforce header structure
	require.Equal(t, header.FixedSize, binary.LittleEndian.Uint32(bytes[0:4]))
	require.Equal(t, header.ReceiptsSize, binary.LittleEndian.Uint32(bytes[4:8]))
	require.Equal(t, header.DiffsSize, binary.LittleEndian.Uint32(bytes[8:12]))
	require.Equal(t, header.TxsSize, binary.LittleEndian.Uint32(bytes[12:16]))

}
