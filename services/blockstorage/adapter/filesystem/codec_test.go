// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package filesystem

import (
	"bytes"
	"encoding/binary"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/rand"
	"github.com/stretchr/testify/require"
	"hash/crc32"
	"testing"
)

func TestCodec_EnforcesBlockSizeLimit(t *testing.T) {
	largeBlock := builders.BlockPair().WithHeight(1).WithTransactions(6).Build()
	c := newCodec(5)
	_, err := c.encode(largeBlock, new(bytes.Buffer))

	require.Error(t, err, "expected to fail encoding a block larger than maxBlockSize")
}

func TestCodec_EncodesAndDecodes(t *testing.T) {
	ctrlRand := rand.NewControlledRand(t)
	block := builders.RandomizedBlock(1, ctrlRand, nil)
	rw := new(bytes.Buffer)
	c := newCodec(1024 * 1024)

	bytesWritten, err := c.encode(block, rw)
	require.NoError(t, err)

	blockLen := rw.Len()

	decodedBlock, readSize, err := c.decode(rw)
	require.NoError(t, err, "expected to decode block record successfully")
	require.EqualValues(t, bytesWritten, readSize, "expected to read same number of bytes as written")
	require.EqualValues(t, blockLen, readSize, "expected to read entire buffer")
	test.RequireCmpEqual(t, block, decodedBlock, "expected to decode an identical block as encoded")
}

func TestCodec_DetectsDataCorruption(t *testing.T) {
	ctrlRand := rand.NewControlledRand(t)

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

func TestBlockHeaderCodec_EncodeAndDecode(t *testing.T) {
	rw := new(bytes.Buffer)
	ctrlRand := rand.NewControlledRand(t)
	header := newBlockHeader()
	header.FixedSize = ctrlRand.Uint32()
	header.ReceiptsSize = ctrlRand.Uint32()
	header.DiffsSize = ctrlRand.Uint32()
	header.TxsSize = ctrlRand.Uint32()

	err := header.write(rw)
	require.NoError(t, err, "expected to encode header successfully")

	decodedHeader := &blockHeader{}
	err = decodedHeader.read(rw)
	require.NoError(t, err, "expected to decode header successfully")

	require.EqualValues(t, header, decodedHeader, "expected decoded header to match original")
}

func TestBlockHeaderCodec_Magic(t *testing.T) {
	rw := new(bytes.Buffer)
	header := newBlockHeader()

	err := header.write(rw)
	require.NoError(t, err, "expected to encode header successfully")

	require.EqualValues(t, "BLOk", rw.Bytes()[:4], "expected header to begin with `BLOk`")
}

func TestBlockHeaderCodec_RejectDecodingWrongMagic(t *testing.T) {
	header := newBlockHeader()

	header.Magic++ // fake wrong magic

	rw := new(bytes.Buffer)
	err := header.write(rw)
	require.NoError(t, err, "expected to encode header successfully")

	decodedHeader := &blockHeader{}
	err = decodedHeader.read(rw)
	require.Error(t, err, "expected to fail parsing block header with wrong magic")
}

func TestBlockHeaderCodec_RejectDecodingWrongVersion(t *testing.T) {
	header := newBlockHeader()

	header.Version++ // fake wrong version

	rw := new(bytes.Buffer)
	err := header.write(rw)
	require.NoError(t, err, "expected to encode header successfully")

	decodedHeader := &blockHeader{}
	err = decodedHeader.read(rw)
	require.Error(t, err, "expected to fail parsing block header with wrong version")
}

func TestFileHeaderCodec_Magic(t *testing.T) {
	header := newBlocksFileHeader(0, 0)

	rw := new(bytes.Buffer)
	err := header.write(rw)
	require.NoError(t, err, "expected to encode header successfully")

	require.EqualValues(t, "ORBS", rw.Bytes()[:4], "expected header to begin with `ORBS`")
}

func TestFileHeaderCodec_EncodesAndDecodesHeader(t *testing.T) {
	ctrlRand := rand.NewControlledRand(t)
	header := newBlocksFileHeader(ctrlRand.Uint32(), ctrlRand.Uint32())

	rw := new(bytes.Buffer)
	err := header.write(rw)
	require.NoError(t, err, "expected to encode header successfully")

	decodedHeader := &blocksFileHeader{}
	err = decodedHeader.read(rw)
	require.NoError(t, err, "expected to decode header successfully")

	test.RequireCmpEqual(t, header, decodedHeader, "expected to decode identical header")
}

func TestFileHeaderCodec_RejectDecodingWrongVersion(t *testing.T) {
	header := newBlocksFileHeader(0, 0)

	header.FileVersion++ // fake wrong version

	rw := new(bytes.Buffer)
	err := header.write(rw)
	require.NoError(t, err, "expected to encode header successfully")

	decodedHeader := &blocksFileHeader{}
	err = decodedHeader.read(rw)
	require.Error(t, err, "expected to fail parsing a header with wrong version")
}

func TestFileHeaderCodec_RejectDecodingWrongMagic(t *testing.T) {
	header := newBlocksFileHeader(0, 0)

	header.Magic++ // fake wrong magic

	rw := new(bytes.Buffer)
	err := header.write(rw)
	require.NoError(t, err, "expected to encode header successfully")

	decodedHeader := &blocksFileHeader{}
	err = decodedHeader.read(rw)
	require.Error(t, err, "expected to fail parsing file header with wrong magic")
}

func TestFileHeaderCodec_RejectDecodingBadChecksum(t *testing.T) {
	header := newBlocksFileHeader(0, 0)

	rw := new(bytes.Buffer)
	err := header.write(rw)
	require.NoError(t, err, "expected to encode header successfully")

	ctrlRand := rand.NewControlledRand(t)
	rw.Bytes()[ctrlRand.Intn(rw.Len())]++ // increment a random byte

	decodedHeader := &blocksFileHeader{}
	err = decodedHeader.read(rw)
	require.Error(t, err, "expected to fail parsing a header with corrupt data")
}

func TestDynamicSectionChecksum(t *testing.T) {
	ctrlRand := rand.NewControlledRand(t)

	rw := new(bytes.Buffer)
	codec := newCodec(10000000)

	block := builders.RandomizedBlock(1, ctrlRand, nil)
	err := codec.writeDynamicBlockSectionWithChecksum(rw, transactionReceiptsToMessages(block.ResultsBlock.TransactionReceipts))
	require.NoError(t, err, "expected to encode receipts successfully")

	encodedChecksum := binary.LittleEndian.Uint32(rw.Bytes()[rw.Len()-4:])

	calcChecksum := crc32.New(crc32.MakeTable(crc32.Castagnoli))
	_, _ = calcChecksum.Write(rw.Bytes()[:rw.Len()-4])

	require.EqualValues(t, calcChecksum.Sum32(), encodedChecksum, "expected encoded section to end correct checksum")

	_, readChecksum, err := codec.readDynamicBlockSection(rw, &readingBudget{limit: 10000000}, uint32(len(block.ResultsBlock.TransactionReceipts)))

	require.NoError(t, err, "expected to decode successfully")
	require.EqualValues(t, readChecksum, encodedChecksum, "expected read method to return encoded checksum")
}

func TestFixedSectionChecksum(t *testing.T) {
	ctrlRand := rand.NewControlledRand(t)

	rw := new(bytes.Buffer)
	codec := newCodec(10000)

	block := builders.RandomizedBlock(1, ctrlRand, nil)
	err := codec.writeFixedBlockSectionWithChecksum(rw, block)
	require.NoError(t, err, "expected to encode fixed section successfully")

	encodedChecksum := binary.LittleEndian.Uint32(rw.Bytes()[rw.Len()-4:])

	calcChecksum := crc32.New(crc32.MakeTable(crc32.Castagnoli))
	_, _ = calcChecksum.Write(rw.Bytes()[:rw.Len()-4])

	require.EqualValues(t, calcChecksum.Sum32(), encodedChecksum, "expected encoded section to end correct checksum")

	_, readChecksum, err := codec.readFixedSection(rw, &readingBudget{limit: 10000})

	require.NoError(t, err, "expected to decode successfully")
	require.EqualValues(t, readChecksum, encodedChecksum, "expected read method to return encoded checksum")

}
