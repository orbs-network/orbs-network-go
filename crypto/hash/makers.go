package hash

func MakeBytesWithFirstByte(size int, firstByteValue int) []byte {
	o := make([]byte, size)
	o[0] = byte(firstByteValue)
	return o
}

func Make32EmptyBytes() []byte {
	return MakeBytesWithFirstByte(SHA256_HASH_SIZE_BYTES, 0)
}

func Make32BytesWithFirstByte(firstByteValue int) []byte {
	return MakeBytesWithFirstByte(SHA256_HASH_SIZE_BYTES, firstByteValue)
}

func MakeEmptyLenBytes() []byte {
	return []byte{}
}
