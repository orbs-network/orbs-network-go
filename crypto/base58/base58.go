package base58

import (
	"fmt"
)

const (
	base58Alphabet  = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	base58ZeroValue = '1'
)

var base58AlphabetDecodeMap = [128]byte{
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 0, 1, 2, 3, 4, 5, 6,
	7, 8, 255, 255, 255, 255, 255, 255,
	255, 9, 10, 11, 12, 13, 14, 15,
	16, 255, 17, 18, 19, 20, 21, 255,
	22, 23, 24, 25, 26, 27, 28, 29,
	30, 31, 32, 255, 255, 255, 255, 255,
	255, 33, 34, 35, 36, 37, 38, 39,
	40, 41, 42, 43, 255, 44, 45, 46,
	47, 48, 49, 50, 51, 52, 53, 54,
	55, 56, 57, 255, 255, 255, 255, 255,
}

func Encode(source []byte) []byte {
	sourceSize := len(source)

	zeroCount := 0
	for zeroCount < sourceSize && source[zeroCount] == 0 {
		zeroCount++
	}

	encodedSize := (sourceSize-zeroCount)*138/100 + 1
	var encodedBuffer = make([]byte, encodedSize)

	msbIndex := encodedSize - 1
	for i := zeroCount; i < sourceSize; i++ {
		currentByteIndex := encodedSize - 1
		for carry := uint32(source[i]); currentByteIndex > msbIndex || carry != 0; currentByteIndex-- {
			carry += uint32(encodedBuffer[currentByteIndex]) << 8
			encodedBuffer[currentByteIndex] = byte(carry % 58)
			carry /= 58
		}
		msbIndex = currentByteIndex
	}

	var encodeIndex int
	for encodeIndex = 0; encodeIndex < encodedSize && encodedBuffer[encodeIndex] == 0; encodeIndex++ {
	}

	var b58 = make([]byte, encodedSize-encodeIndex+zeroCount)
	for i := 0; i < zeroCount; i++ {
		b58[i] = base58ZeroValue
	}

	for i := zeroCount; encodeIndex < encodedSize; i++ {
		b58[i] = base58Alphabet[encodedBuffer[encodeIndex]]
		encodeIndex++
	}

	return b58
}

func Decode(source []byte) ([]byte, error) {
	sourceLength := len(source)

	var (
		zeroMask  uint32
		zeroCount int

		bytesLeft = sourceLength % 4
	)

	if bytesLeft > 0 {
		zeroMask = 0xffffffff << uint32(bytesLeft*8)
	} else {
		bytesLeft = 4
	}

	decodedIntegersSize := (sourceLength + 3) / 4
	var decodedIntegers = make([]uint32, decodedIntegersSize)

	for i := 0; i < sourceLength && source[i] == base58ZeroValue; i++ {
		zeroCount++
	}

	for _, currentSourceChar := range source {
		if currentSourceChar > 127 || base58AlphabetDecodeMap[currentSourceChar] == 255 {
			return nil, fmt.Errorf("invalid base58 digit (%q) in input %s", currentSourceChar, source)
		}

		c := uint32(base58AlphabetDecodeMap[currentSourceChar])

		for j := decodedIntegersSize - 1; j >= 0; j-- {
			t := uint64(decodedIntegers[j])*58 + uint64(c)
			c = uint32(t>>32) & 0x3f
			decodedIntegers[j] = uint32(t & 0xffffffff)
		}

		if c > 0 {
			return nil, fmt.Errorf("output number too big (carry to the next int32)")
		}

		if decodedIntegers[0]&zeroMask != 0 {
			return nil, fmt.Errorf("output number too big (last int32 filled too far)")
		}
	}

	// we may be using more memory than needed, but the optimization is not required (imo)
	decodedBytes := make([]byte, (sourceLength+3)*2)

	totalBytesUsed := 0
	currentDecodedIndex := 0
	// first time around we may not have a full uint32 to process
	if bytesLeft < 4 {
		for conversionMask := uint32(bytesLeft-1) * 8; conversionMask <= 0x18; conversionMask -= 8 {
			decodedBytes[totalBytesUsed] = byte(decodedIntegers[currentDecodedIndex] >> conversionMask)
			totalBytesUsed++
		}
		bytesLeft = 4
		currentDecodedIndex++
	}

	for ; currentDecodedIndex < decodedIntegersSize; currentDecodedIndex++ {
		for conversionMask := uint32(0x18); conversionMask <= 0x18; conversionMask -= 8 {
			decodedBytes[totalBytesUsed] = byte(decodedIntegers[currentDecodedIndex] >> conversionMask)
			totalBytesUsed++
		}
	}

	// this can probably be made more efficient as well by optimizing memory consumption, but its minor
	start := 0
	for n, v := range decodedBytes {
		if v > 0 {
			if n > zeroCount {
				start = n - zeroCount
			}
			break
		}
	}
	return decodedBytes[start:totalBytesUsed], nil
}
