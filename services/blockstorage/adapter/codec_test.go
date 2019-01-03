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
	ctrlRand := test.NewControlledRand(t)
	block := builders.RandomizedBlock(1, ctrlRand, nil)
	rw := new(bytes.Buffer)

	err := encode(block, rw)
	require.NoError(t, err)

	decodedBlock, _, err := decode(rw)
	require.NoError(t, err)
	test.RequireCmpEqual(t, block, decodedBlock)
}

func TestDetectsCorruption(t *testing.T) {
	ctrlRand := test.NewControlledRand(t)

	block := builders.RandomizedBlock(1, ctrlRand, nil)

	encodedBlock := new(bytes.Buffer)
	err := encode(block, encodedBlock)
	blockBytes := encodedBlock.Bytes()
	require.NoError(t, err)

	_, _, err = decode(encodedBlock)
	require.NoError(t, err, "expected uncorrupted bytes to decode successfully")

	// corrupt random bits
	corruptBlock := new(bytes.Buffer)
	for i := 0; i < len(blockBytes); i += ctrlRand.Intn(len(blockBytes) / 20) {
		corruptBlock.Reset()
		corruptBlock.Write(blockBytes)
		raw := corruptBlock.Bytes()

		// flip one bit
		bitFlip := byte(1) << uintptr(ctrlRand.Intn(8))
		raw[i] = raw[i] ^ bitFlip

		_, _, err = decode(corruptBlock)
		require.Error(t, err, "expected codec to detect data corruption when flipping bit %08b in byte %v/%v", bitFlip, i, len(blockBytes))
		t.Logf("flipping bit %08b in byte %v/%v", bitFlip, i, len(blockBytes))
	}
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
	require.EqualValues(t, header.FixedSize, binary.LittleEndian.Uint32(bytes[0:4]))
	require.EqualValues(t, header.ReceiptsSize, binary.LittleEndian.Uint32(bytes[4:8]))
	require.EqualValues(t, header.DiffsSize, binary.LittleEndian.Uint32(bytes[8:12]))
	require.EqualValues(t, header.TxsSize, binary.LittleEndian.Uint32(bytes[12:16]))

}
