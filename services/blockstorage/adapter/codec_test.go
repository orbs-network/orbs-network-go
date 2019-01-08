package adapter

import (
	"bytes"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCodec_EnforcesBlockSizeLimit(t *testing.T) {
	largeBlock := builders.BlockPair().WithHeight(1).WithTransactions(6).Build()
	c := newCodec(5)
	_, err := c.encode(largeBlock, new(bytes.Buffer))

	require.Error(t, err, "expected to fail encoding a block larger than maxBlockSize")
}

func TestCodec_EncodesAndDecodes(t *testing.T) {
	ctrlRand := test.NewControlledRand(t)
	block := builders.RandomizedBlock(1, ctrlRand, nil)
	rw := new(bytes.Buffer)
	c := newCodec(1024 * 1024)

	writeSize, err := c.encode(block, rw)
	require.NoError(t, err)

	blockLen := rw.Len()

	decodedBlock, readSize, err := c.decode(rw)
	require.NoError(t, err, "expected to decode block record successfully")
	require.EqualValues(t, writeSize, readSize, "expected to read same number of bytes as written")
	require.EqualValues(t, blockLen, readSize, "expected to read entire buffer")
	test.RequireCmpEqual(t, block, decodedBlock, "expected to decode an identical block as encoded")
}

func TestCodec_DetectsDataCorruption(t *testing.T) {
	ctrlRand := test.NewControlledRand(t)

	block := builders.RandomizedBlock(1, ctrlRand, nil)

	// serialize
	c := newCodec(1024 * 1024)
	encodedBlock := new(bytes.Buffer)
	_, err := c.encode(block, encodedBlock)
	blockBytes := encodedBlock.Bytes()
	require.NoError(t, err, "expected to encode block successfully")

	// decode ok
	_, _, err = c.decode(encodedBlock)
	require.NoError(t, err, "expected uncorrupted bytes to decode successfully")

	// decode fail with random bit flips
	corruptBlock := new(bytes.Buffer)
	for ri := 0; ri < len(blockBytes); ri += ctrlRand.Intn(len(blockBytes) / 20) {
		// clone block reader
		corruptBlock.Reset()
		corruptBlock.Write(blockBytes)

		// flip one bit
		bitFlip := byte(1) << uintptr(ctrlRand.Intn(8))
		raw := corruptBlock.Bytes()
		raw[ri] = raw[ri] ^ bitFlip

		_, _, err = c.decode(corruptBlock)
		require.Error(t, err, "expected codec to detect data corruption when flipping bit %08b in byte %v/%v", bitFlip, ri, len(blockBytes))
		t.Logf("flipping bit %08b in byte %v/%v", bitFlip, ri, len(blockBytes))
	}
}

func TestEncodeHeader(t *testing.T) {
	rw := new(bytes.Buffer)
	header := &blockHeader{
		Version:      0,
		FixedSize:    1,
		ReceiptsSize: 2,
		DiffsSize:    3,
		TxsSize:      4,
	}
	err := header.write(rw)
	require.NoError(t, err, "expected to encode header successfully")

	bytes := rw.Bytes()
	require.Len(t, bytes, 5*4, "expected header size to be 4 bytes per header field")

	decodedHeader := &blockHeader{}
	err = decodedHeader.read(rw)
	require.NoError(t, err, "expected to decode header successfully")

	require.EqualValues(t, header, decodedHeader, "expected decoded header to match original")
}
