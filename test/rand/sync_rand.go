// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package rand

import (
	"math/rand"
	"sync"
)

// syncRand is a locking wrapper around rand.Rand object
// unlike math/rand.lockedSource this object does not distinguish
// between single and array value random methods.
type syncRand struct {
	lk sync.Mutex
	r  *rand.Rand // avoid using no identifier name to block access to unwrapped methods in Rand
}

func newSyncRand(seed int64) *syncRand {
	return &syncRand{r: rand.New(rand.NewSource(seed))}
}

func (sr *syncRand) Read(p []byte) (n int, err error) {
	sr.lk.Lock()
	defer sr.lk.Unlock()
	return sr.r.Read(p)
}

func (sr *syncRand) Seed(seed int64) {
	sr.lk.Lock()
	defer sr.lk.Unlock()
	sr.r.Seed(seed)
}

func (sr *syncRand) Int63() int64 {
	sr.lk.Lock()
	defer sr.lk.Unlock()
	return sr.r.Int63()
}

func (sr *syncRand) Uint32() uint32 {
	sr.lk.Lock()
	defer sr.lk.Unlock()
	return sr.r.Uint32()
}

func (sr *syncRand) Uint64() uint64 {
	sr.lk.Lock()
	defer sr.lk.Unlock()
	return sr.r.Uint64()
}

func (sr *syncRand) Int31() int32 {
	sr.lk.Lock()
	defer sr.lk.Unlock()
	return sr.r.Int31()
}

func (sr *syncRand) Int() int {
	sr.lk.Lock()
	defer sr.lk.Unlock()
	return sr.r.Int()
}

func (sr *syncRand) Int63n(n int64) int64 {
	sr.lk.Lock()
	defer sr.lk.Unlock()
	return sr.r.Int63n(n)
}

func (sr *syncRand) Int31n(n int32) int32 {
	sr.lk.Lock()
	defer sr.lk.Unlock()
	return sr.r.Int31n(n)
}

func (sr *syncRand) Intn(n int) int {
	sr.lk.Lock()
	defer sr.lk.Unlock()
	return sr.r.Intn(n)
}

func (sr *syncRand) Float64() float64 {
	sr.lk.Lock()
	defer sr.lk.Unlock()
	return sr.r.Float64()
}

func (sr *syncRand) Float32() float32 {
	sr.lk.Lock()
	defer sr.lk.Unlock()
	return sr.r.Float32()
}

func (sr *syncRand) Perm(n int) []int {
	sr.lk.Lock()
	defer sr.lk.Unlock()
	return sr.r.Perm(n)
}

func (sr *syncRand) Shuffle(n int, swap func(i, j int)) {
	sr.lk.Lock()
	defer sr.lk.Unlock()
	sr.r.Shuffle(n, swap)
}
