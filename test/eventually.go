package test

import (
	"github.com/orbs-network/go-mock"
	"time"
)

const iterations = 10
const interval = 5 * time.Millisecond

func Eventually(f func() bool) bool {
	for i := 0; i < iterations; i++ {
		if f() {
			return true
		}
		time.Sleep(interval)
	}
	return false
}

func Consistently(f func() bool) bool {
	for i := 0; i < iterations; i++ {
		if !f() {
			return false
		}
		time.Sleep(interval)
	}
	return true
}

func EventuallyVerify(mock mock.HasVerify) (err error) {
	var ok bool
	Eventually(func() bool {
		ok, err = mock.Verify()
		return ok
	})
	return
}

func ConsistentlyVerify(mock mock.HasVerify) (err error) {
	var ok bool
	Consistently(func() bool {
		ok, err = mock.Verify()
		return ok
	})
	return
}
