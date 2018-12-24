package hash

import (
	"github.com/stretchr/testify/require"
	"testing"
)

var someData = []byte("testing")

const (
	ExpectedSha256         = "cf80cd8aed482d5d1527d7dc72fceff84e6326592848447d2dc0b0e87dfc9a90"
	ExpectedSha256Ripmd160 = "1acb19a469206161ed7e5ed9feb996a6e24be441"
	ExpectedKeccak256      = "5f16f4c7f149ac4f9510d9cf8cf384038ad348b3bcdc01915f95de12df9d1b02"
)

func TestCalcSha256(t *testing.T) {
	h := CalcSha256(someData)
	require.Equal(t, SHA256_HASH_SIZE_BYTES, len(h))
	require.Equal(t, ExpectedSha256, h.String(), "result should match")
}

func TestCalcSha256_MultipleChunks(t *testing.T) {
	h := CalcSha256(someData[:3], someData[3:])
	require.Equal(t, SHA256_HASH_SIZE_BYTES, len(h))
	require.Equal(t, ExpectedSha256, h.String(), "result should match")
}

func TestCalcRipemd160Sha256(t *testing.T) {
	h := CalcRipemd160Sha256(someData)
	require.Equal(t, RIPEMD160_HASH_SIZE_BYTES, len(h))
	require.Equal(t, ExpectedSha256Ripmd160, h.String(), "result should match")
}

func TestCalcKeccak256(t *testing.T) {
	h := CalcKeccak256(someData)
	require.Equal(t, KECCAK256_HASH_SIZE_BYTES, len(h))
	require.Equal(t, ExpectedKeccak256, h.String(), "result should match")
}

func BenchmarkCalcSha256(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CalcSha256(someData)
	}
}

func BenchmarkCalcRipemd160Sha256(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CalcRipemd160Sha256(someData)
	}
}
