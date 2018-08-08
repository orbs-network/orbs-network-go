package bloom

import "github.com/orbs-network/orbs-spec/types/go/primitives"

// this bf works on time stamps, and more specifically, the time stamps intervals - we know that the set of numbers will be relatively small - the difference between blocks
// so if bA has closed at 20180101-12:00:00 and bB closed at 20180101-12:00:30 (30 seconds later) the bf can only contain the time intervals in between the values,
// ensuring small entropy. usually blocks will be closed in less than a second, a full 32bit uint will hold 4.29 seconds of nanosecond interval (ts is UnixNano()),
// so the performance here should be very high with very little false positive
type TimestampBloomFilter struct {
	bitset   []bool
	size     uint32 // size of bf
	bitCount uint8
}

const (
	FNVMagicNumber = 0xAF63BD4C8601B7DF
	//FNVOffset = 0xcbf29ce484222325
	//FNVPrime  = 0x100000001b3
)

func nextHighPowerOfTwo(n uint32) uint32 {
	// bit twiddling
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n++
	return n
}

func countPowerOfTwoBit(n uint32) uint8 {
	// a bit quicker than log()
	c := uint8(0)
	for ; n > 1; n >>= 1 {
		c++
	}
	return c
}

func New(size int32) *TimestampBloomFilter {
	roundedSize := nextHighPowerOfTwo(uint32(size))
	ts := &TimestampBloomFilter{
		bitset:   make([]bool, roundedSize),
		size:     roundedSize,
		bitCount: countPowerOfTwoBit(roundedSize),
	}

	return ts
}

func NewFromRaw(raw primitives.BloomFilter) *TimestampBloomFilter {
	size := uint32(len(raw) * 8)
	ts := &TimestampBloomFilter{
		bitset:   make([]bool, size),
		size:     size,
		bitCount: countPowerOfTwoBit(size),
	}

	for i, b := range raw {
		for j := 7; j >= 0; j-- {
			bit := b & (1 << uint(j))
			ts.bitset[(7-j)+(i*8)] = bit != 0
		}
	}

	return ts
}

func (bf *TimestampBloomFilter) Size() uint32 {
	return bf.size
}

func (bf *TimestampBloomFilter) BitCount() uint8 {
	return bf.bitCount
}

func (bf *TimestampBloomFilter) hash(v uint64) uint64 {
	// using fnv-1 and fnv-1a, we hash and dump the leftmost bits (in theory more false positives, but we are in a low entropy, can probably come up with something quicker even
	hash := FNVMagicNumber ^ v
	hash <<= 64 - bf.bitCount
	loc := hash >> (64 - bf.bitCount)
	return loc
}

func (bf *TimestampBloomFilter) Add(timeStamp uint64) {
	loc := bf.hash(timeStamp)
	bf.bitset[loc] = true
}

func (bf *TimestampBloomFilter) Test(timeStamp uint64) bool {
	loc := bf.hash(timeStamp)
	return bf.bitset[loc]
}

func boolSliceToByte(slice []bool) byte {
	if len(slice) > 8 {
		return 0
	}

	l := len(slice) - 1
	r := byte(0)

	for i, b := range slice {
		if b {
			mask := byte(1) << uint(l-i)
			r |= mask
		}
	}
	return r
}

func (bf *TimestampBloomFilter) Raw() primitives.BloomFilter {
	byteCount := int(bf.size / 8)
	output := make([]byte, 0, byteCount)
	for i := 0; i < byteCount; i++ {
		boolSlice := bf.bitset[i*8 : (i+1)*8]
		b := boolSliceToByte(boolSlice)
		output = append(output, b)
	}

	return output
}

func (bf *TimestampBloomFilter) Equals(other *TimestampBloomFilter) bool {
	if bf.size != other.size {
		return false
	}

	for i, b := range bf.bitset {
		if b != other.bitset[i] {
			return false
		}
	}

	return true
}
