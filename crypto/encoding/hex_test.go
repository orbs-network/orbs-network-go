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
}

func TestHexEncodeWithChecksum(t *testing.T) {
	for _, pair := range encodeStringTestTable {
		data, err := hex.DecodeString(pair.sourceHex)
		require.NoError(t, err, "failed to decode, human error most likely")
		encoded := EncodeHex(data)

		require.Equal(t, pair.checksumEncodedHex, encoded, "expected encoding with a specific result for each input")
	}
}
