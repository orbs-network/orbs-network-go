package test

import (
	"github.com/orbs-network/go-mock"
	"time"
)

const iterationsEventually = 100
const iterationsConsistently = 100
const interval = 5 * time.Millisecond

func Eventually(f func() bool) bool {
	for i := 0; i < iterationsEventually; i++ {
		if f() {
			return true
		}
		time.Sleep(interval)
	}
	return false
}

func Consistently(f func() bool) bool {
	for i := 0; i < iterationsConsistently; i++ {
		if !f() {
			return false
		}
		time.Sleep(interval)
	}
	return true
}

func EventuallyVerify(mocks ...mock.HasVerify) error {
	verified := make([]bool, len(mocks))
	numVerified := 0
	var errExample error
	Eventually(func() bool {
		for i, mock := range mocks {
			if !verified[i] {
				ok, err := mock.Verify()
				if ok {
					verified[i] = true
					numVerified++
				} else {
					errExample = err
				}
			}
		}
		return numVerified == len(mocks)
	})
	if numVerified == len(mocks) {
		return nil
	} else {
		return errExample
	}
}

func ConsistentlyVerify(mocks ...mock.HasVerify) error {
	var errExample error
	Consistently(func() bool {
		for _, mock := range mocks {
			ok, err := mock.Verify()
			if !ok {
				errExample = err
				return false
			}
		}
		return true
	})
	return errExample
}
