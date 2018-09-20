package hash_test

import (
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"testing"
)

var someData = []byte("testing")

const (
	ExpectedSha256         = "cf80cd8aed482d5d1527d7dc72fceff84e6326592848447d2dc0b0e87dfc9a90"
	ExpectedSha256Ripmd160 = "1acb19a469206161ed7e5ed9feb996a6e24be441"
)

func TestCalcSha256(t *testing.T) {
	h := hash.CalcSha256(someData)
	if h.String() != ExpectedSha256 {
		t.Errorf("sha256 failed expected %s got %s", ExpectedSha256, h)
	}
}

func TestCalcRipmd160Sha256(t *testing.T) {
	h := hash.CalcRipmd160Sha256(someData)
	if h.String() != ExpectedSha256Ripmd160 {
		t.Errorf("sha256ripmd160 failed expected %s got %s", ExpectedSha256Ripmd160, h)
	}
}

func BenchmarkCalcSha256(b *testing.B) {
	for i := 0; i < b.N; i++ {
		hash.CalcSha256(someData)
	}
}

func BenchmarkCalcRipmd160Sha256(b *testing.B) {
	for i := 0; i < b.N; i++ {
		hash.CalcRipmd160Sha256(someData)
	}
}
