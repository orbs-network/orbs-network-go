// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package bloom_test

import (
	"bytes"
	"github.com/orbs-network/orbs-network-go/crypto/bloom"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"testing"
	"time"
)

type tsbfNewTrio struct {
	input    int
	size     uint32
	bitCount uint8
}

var newTestTable = []tsbfNewTrio{
	{input: 15, size: 16, bitCount: 4},
	{input: 20, size: 32, bitCount: 5},
	{input: 200, size: 256, bitCount: 8},
	{input: 1024, size: 1024, bitCount: 10},
}

var nanoForRaw = []primitives.TimestampNano{
	1533731643509419667,
	1533731643509435135,
	1533731643509465891,
	1533731643509489416,
	1533731643509515475,
	1533731643509519636,
	1533731643511038190,
	1533731643511049004,
}

func TestNew(t *testing.T) {
	for _, test := range newTestTable {
		x := bloom.New(test.input)
		if x.Size() != test.size {
			t.Errorf("size should be %d, but is %d", test.size, x.Size())
		}

		if x.BitCount() != test.bitCount {
			t.Errorf("bitcount should be %d, but is %d", test.bitCount, x.BitCount())
		}
	}
}

func TestTimestampBloomFilter_Add(t *testing.T) {
	x := bloom.New(16)
	t1 := primitives.TimestampNano(time.Now().UnixNano())
	x.Add(t1)
	if !x.Test(t1) {
		t.Errorf("bloom filter failed, value should have been in the filter")
	}
}

func TestTimestampBloomFilter_AddAndTestInvalid(t *testing.T) {
	x := bloom.New(16)
	t1 := primitives.TimestampNano(time.Now().UnixNano())
	x.Add(t1)
	if !x.Test(t1) {
		t.Errorf("bloom filter failed, value should have been in the filter")
	}
	t1++
	// this may be flaky, but at a low probability (if it happens a lot or even at all then we have a problem with the hash function)
	if x.Test(t1) {
		t.Errorf("bloom filter failed, value should not have been in the filter")
	}
}

func TestTimestampBloomFilter_Equals(t *testing.T) {
	x := bloom.New(16)
	for _, ts := range nanoForRaw {
		x.Add(ts)
	}

	other := bloom.New(16)
	for _, ts := range nanoForRaw {
		other.Add(ts)
	}

	if !x.Equals(other) {
		t.Errorf("expected both bloom filters with same data to be equivalent")
	}
}

func TestTimestampBloomFilter_Raw(t *testing.T) {
	x := bloom.New(16)
	for _, ts := range nanoForRaw {
		x.Add(ts)
	}

	expected := []byte{209, 24}

	if !bytes.Equal(expected, x.Raw()) {
		t.Errorf("raw did not output the expected byte state")
	}
}

func TestTimestampBloomFilter_Raw_Small(t *testing.T) {
	x := bloom.New(1)
	x.Add(nanoForRaw[0])

	expected := []byte{1}

	if !bytes.Equal(expected, x.Raw()) {
		t.Errorf("raw did not output the expected byte state")
	}
}

func TestNewFromRaw(t *testing.T) {
	x := bloom.New(16)
	for _, ts := range nanoForRaw {
		x.Add(ts)
	}

	fromRaw := bloom.NewFromRaw(x.Raw())

	if !x.Equals(fromRaw) {
		t.Error("serialization from raw failed")
	}
}

func BenchmarkFillTSBloom(b *testing.B) {
	b.StopTimer()
	x := bloom.New(16)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		x.Add(nanoForRaw[0])
	}
}

func BenchmarkTestTSBloom(b *testing.B) {
	b.StopTimer()
	x := bloom.New(16)
	for _, ts := range nanoForRaw {
		x.Add(ts)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		r := x.Test(nanoForRaw[3])
		if !r {
			b.Error("bloom filter failed, value should have been in the filter")
		}
	}
}
