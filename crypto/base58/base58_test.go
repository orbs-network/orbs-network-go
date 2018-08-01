package base58_test

import (
	"bytes"
	"github.com/orbs-network/orbs-network-go/crypto/base58"
	"testing"
)

type testStringPair struct {
	decoded string
	encoded string
}

type testBytePair struct {
	decoded []byte
	encoded []byte
}

var encodeStringTestTable = []testStringPair{
	{"", ""},
	{"0", "q"},
	{"9", "z"},
	{":", "21"},
	{"0001", "2ESbJx"},
	{"helloworldandstuff1124z24", "j1Q1Y54mCcVfR5jVAQMMJEy6VbZEtYeM3R"},
	{"abcde37f332f68db77bd9d7edd4969571ad671cf9dd3babcd", "GmWWBp6bqF7cUpZTyfRgPKvrkSPJJSoJWapvpPSTH2Cnf67dCEm1nfbnh5vfKVV4tXu"},
}

var encodeByteTestTable = []testBytePair{
	{[]byte{}, []byte{}},
	{[]byte{0}, []byte("1")},
	{[]byte{0, 0, 0, 1}, []byte("1112")},
}

var invalidBase58StringDecodeTestList = []string{
	"0",
	"_",
	"xxIxx",
	"dj#x4Z",
}

func TestBase58Encode(t *testing.T) {
	for _, pair := range encodeStringTestTable {
		out := base58.Encode([]byte(pair.decoded))
		if string(out) != pair.encoded {
			t.Errorf("Base58 encode mismatch, decoded: %s, encoded: %v(%s), expected: %s", pair.decoded, out, out, pair.encoded)
		}
	}

	for _, pair := range encodeByteTestTable {
		out := base58.Encode(pair.decoded)
		if !bytes.Equal(out, pair.encoded) {
			t.Errorf("Base58 encode mismatch, decoded: %v, encoded: %v(%s), expected: %s", pair.decoded, out, out, pair.encoded)
		}
	}
}

func TestBase58Decode(t *testing.T) {
	for _, pair := range encodeStringTestTable {
		out, err := base58.Decode([]byte(pair.encoded))
		if err != nil {
			t.Errorf("Base58 failed to decode %s, error is %s", pair.decoded, err)
		}
		if string(out) != pair.decoded {
			t.Errorf("Base58 decode mismatch, encoded: %s, decoded: %s, expected: %s", pair.encoded, string(out), pair.decoded)
		}
	}

	for _, pair := range encodeByteTestTable {
		out, err := base58.Decode([]byte(pair.encoded))
		if err != nil {
			t.Errorf("Base58 failed to decode %s, error is %s", pair.decoded, err)
		}
		if !bytes.Equal(out, pair.decoded) {
			t.Errorf("Base58 decode mismatch, encoded: %v, decoded: %s, expected: %v", pair.encoded, out, pair.decoded)
		}
	}
}

func TestBase58InvalidDecode(t *testing.T) {
	for _, value := range invalidBase58StringDecodeTestList {
		out, err := base58.Decode([]byte(value))
		if err == nil {
			t.Errorf("Base58 decoded an invalid string %s, output is %v", value, out)
		}
	}
}

func BenchmarkBase58Encode(b *testing.B) {
	rawAddress := []byte("X0abcde37f332f68db77bd9d7edd4969571ad671cf9dd3babcd")
	for i := 0; i < b.N; i++ {
		base58.Encode(rawAddress)
	}
}

func BenchmarkBase58Decode(b *testing.B) {
	b.StopTimer()
	rawAddressEncoded := base58.Encode([]byte("X0abcde37f332f68db77bd9d7edd4969571ad671cf9dd3babcd"))
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		base58.Decode(rawAddressEncoded)
	}
}
