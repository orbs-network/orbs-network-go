package encoding

import (
	"encoding/hex"
	"github.com/stretchr/testify/require"
	"testing"
)

type testStringPair struct {
	sourceHex          string
	checksumEncodedHex string
}

var encodeStringTestTable = []testStringPair{
	{"de709f2102306220921060314715629080e2fb77", "0xdE709f2102306220921060314715629080e2FB77"},
	{"dbF03B407c01E7cD3CBea99509d93f8DDDC8C6FB", "0xdBf03B407c01e7CD3CBea99509d93F8DDdc8c6Fb"},
	{"abcdef", "0xABcDEf"},
	{"dbF03B407c01E7cD3CBea99509d93f8DDDC8C6FBdbF03B407c01E7cD3CBea99509d93f8DDDC8C6FB", "0xDBF03b407C01E7CD3CBea99509D93F8DdDC8C6FBdbF03b407C01e7Cd3cbEa99509D93f8DdDc8C6FB"},
	{"dbF03B407c01E7cD3CBea99509d93f8DDDC8C6FBdbF03B407c01E7cD3CBea99509d93f8DDDC8C6FBde709f2102306220921060314715629080e2fb77dbF03B407c01E7cD3CBea99509d93f8DDDC8C6FBdbF03B407c01E7cD3CBea99509d93f8DDDC8C6FBde709f2102306220921060314715629080e2fb77", "0xDBf03b407C01E7Cd3CBeA99509d93F8DDdC8C6fBDBf03B407c01E7CD3cbEa99509d93f8dDDc8C6FbdE709f2102306220921060314715629080E2FB77dbf03b407C01E7Cd3CbEA99509D93f8Dddc8c6fBDbF03B407C01E7cd3cBeA99509d93f8DDDc8C6FbDE709F2102306220921060314715629080e2FB77"},
}

func TestHexEncodeWithChecksum(t *testing.T) {
	for _, pair := range encodeStringTestTable {
		data, err := hex.DecodeString(pair.sourceHex)
		require.NoError(t, err, "failed to decode, human error most likely")
		encoded := EncodeHex(data)

		require.Equal(t, pair.checksumEncodedHex, encoded, "expected encoding with a specific result for each input")
	}
}

func TestHexDecodeGoodChecksum(t *testing.T) {
	for _, pair := range encodeStringTestTable {
		rawData, err := hex.DecodeString(pair.sourceHex)
		require.NoError(t, err, "failed to decode, human error most likely")
		decoded, err := DecodeHex(pair.checksumEncodedHex)
		require.NoError(t, err, "checksum should be valid")
		require.Equal(t, rawData, decoded, "data should be decoded correctly")
	}
}

func TestHexDecodeBadChecksum(t *testing.T) {
	pair := encodeStringTestTable[0]
	rawData, err := hex.DecodeString(pair.sourceHex)
	require.NoError(t, err, "failed to decode, human error most likely")
	wrongCheckSum := "de" + pair.checksumEncodedHex[4:]
	decoded, err := DecodeHex(wrongCheckSum)
	require.EqualError(t, err, "invalid checksum", "checksum should be invalid")
	require.Equal(t, rawData, decoded, "data should be decoded correctly even though checksum is invalid")
}

func TestHexDecodeInvalidHex(t *testing.T) {
	decoded, err := DecodeHex("0")
	require.Error(t, err, "should not succeed with invalid hex")
	require.Nil(t, decoded, "result should be nil")
}

func BenchmarkHexEncodeWithChecksum(b *testing.B) {
	rawData, err := hex.DecodeString(encodeStringTestTable[0].sourceHex)
	require.NoError(b, err, "failed to decode, human error most likely")
	for i := 0; i < b.N; i++ {
		EncodeHex(rawData)
	}
}

func BenchmarkHexDecodeWithChecksum(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := DecodeHex(encodeStringTestTable[0].checksumEncodedHex)
		if err != nil { // require/testify is very slow
			b.Fail()
		}
	}
}
