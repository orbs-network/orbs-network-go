package test

import (
	"math/rand"
	"sync"
)

type syncRand struct {
	lk sync.Mutex
	*rand.Rand
}

func newSyncRand(seed int64) *syncRand {
	return &syncRand{Rand: rand.New(rand.NewSource(seed))}
}
func (r *syncRand) Read(p []byte) (n int, err error) {
	r.lk.Lock()
	defer r.lk.Unlock()
	return r.Rand.Read(p)
}
func (r *syncRand) Seed(seed int64) {
	r.lk.Lock()
	defer r.lk.Unlock()
	r.Rand.Seed(seed)
}
func (r *syncRand) Int63() int64 {
	r.lk.Lock()
	defer r.lk.Unlock()
	return r.Rand.Int63()
}
func (r *syncRand) Uint32() uint32 {
	r.lk.Lock()
	defer r.lk.Unlock()
	return r.Rand.Uint32()
}
func (r *syncRand) Uint64() uint64 {
	r.lk.Lock()
	defer r.lk.Unlock()
	return r.Rand.Uint64()
}
func (r *syncRand) Int31() int32 {
	r.lk.Lock()
	defer r.lk.Unlock()
	return r.Rand.Int31()
}
func (r *syncRand) Int() int {
	r.lk.Lock()
	defer r.lk.Unlock()
	return r.Rand.Int()
}
func (r *syncRand) Int63n(n int64) int64 {
	r.lk.Lock()
	defer r.lk.Unlock()
	return r.Rand.Int63n(n)
}
func (r *syncRand) Int31n(n int32) int32 {
	r.lk.Lock()
	defer r.lk.Unlock()
	return r.Rand.Int31n(n)
}
func (r *syncRand) Intn(n int) int {
	r.lk.Lock()
	defer r.lk.Unlock()
	return r.Rand.Intn(n)
}
func (r *syncRand) Float64() float64 {
	r.lk.Lock()
	defer r.lk.Unlock()
	return r.Rand.Float64()
}
func (r *syncRand) Float32() float32 {
	r.lk.Lock()
	defer r.lk.Unlock()
	return r.Rand.Float32()
}
func (r *syncRand) Perm(n int) []int {
	r.lk.Lock()
	defer r.lk.Unlock()
	return r.Rand.Perm(n)
}
func (r *syncRand) Shuffle(n int, swap func(i, j int)) {
	r.lk.Lock()
	defer r.lk.Unlock()
	r.Rand.Shuffle(n, swap)
}
