package adapter

import (
	"strconv"
	"fmt"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"bytes"
	"encoding/binary"
)

func copyByteArray(data []byte) []byte {
	result := make([]byte, len(data))
	copy(result, data)

	return result
}

func copyArrayOfByteArrays(data [][]byte) [][]byte {
	result := make([][]byte, len(data))
	for i := range data {
		result[i] = make([]byte, len(data[i]))
		copy(result[i], data[i])
	}

	return result
}

func formatInt(i int) string {
	return strconv.FormatInt(int64(i), 10)
}

func anyErrors(errors []error) bool {
	for _, error := range errors {
		if error != nil {
			fmt.Println("Found error", errors)
			return true
		}
	}

	return false
}

func anyConditions(bools []bool) bool {
	for _, val := range bools {
		if val == false {
			return false
		}
	}

	return true
}

func bufferPutKeyValue(buffer *bytes.Buffer, key string, value []byte) {
	keyLength := uint64(len(key))
	valueLength := uint64(len(value))

	keyLengthAsBytes := make([]byte, binary.MaxVarintLen64)
	binary.PutUvarint(keyLengthAsBytes, keyLength)

	buffer.Write(keyLengthAsBytes)
	buffer.Write([]byte(key))

	valueLengthAsBytes := make([]byte, binary.MaxVarintLen64)
	binary.PutUvarint(valueLengthAsBytes, valueLength)

	buffer.Write(valueLengthAsBytes)
	buffer.Write(value)
}

func bufferReadPair(data []byte, offset int) (value []byte, newOffset int) {
	keyLengthStart := offset
	keyLengthEnd := offset+binary.MaxVarintLen64

	keyLength, _ := binary.ReadUvarint(bytes.NewReader(data[keyLengthStart:keyLengthEnd]))

	keyStart := keyLengthEnd
	keyEnd := keyStart +int(keyLength)

	key := data[keyStart:keyEnd]

	return key, keyEnd
}

func iterateOverKeyValueBuffer(buffer *bytes.Buffer, parseValue func(key string, value []byte)) {
	data := buffer.Bytes()
	offset := 0

	var key []byte
	var value []byte

	for offset < len(data)  {
		key, offset = bufferReadPair(buffer.Bytes(), offset)
		value, offset = bufferReadPair(buffer.Bytes(), offset)

		parseValue(string(key), value)
	}
}

func blockAsByteArray(container *protocol.BlockPairContainer) (result []byte) {
	buffer := bytes.NewBuffer([]byte{})


	return buffer.Bytes()
}

func byteArrayAsBlock(data []byte) *protocol.BlockPairContainer {
	return nil
}