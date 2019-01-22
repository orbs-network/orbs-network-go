package test

import (
	"github.com/orbs-network/go-mock"
	"time"
)

const EVENTUALLY_ACCEPTANCE_TIMEOUT = 40 * time.Millisecond
const EVENTUALLY_ADAPTER_TIMEOUT = 100 * time.Millisecond
const EVENTUALLY_LOCAL_E2E_TIMEOUT = 400 * time.Millisecond
const EVENTUALLY_DOCKER_E2E_TIMEOUT = 1000 * time.Millisecond

const CONSISTENTLY_ACCEPTANCE_TIMEOUT = 20 * time.Millisecond
const CONSISTENTLY_ADAPTER_TIMEOUT = 50 * time.Millisecond
const CONSISTENTLY_LOCAL_E2E_TIMEOUT = 200 * time.Millisecond
const CONSISTENTLY_DOCKER_E2E_TIMEOUT = 500 * time.Millisecond

const eventuallyIterations = 50
const consistentlyIterations = 25

func Eventually(timeout time.Duration, f func() bool) bool {
	for i := 0; i < eventuallyIterations; i++ {
		if testButDontPanic(f) {
			return true
		}
		time.Sleep(timeout / eventuallyIterations)
	}
	return false
}

func Consistently(timeout time.Duration, f func() bool) bool {
	for i := 0; i < consistentlyIterations; i++ {
		if !testButDontPanic(f) {
			return false
		}
		time.Sleep(timeout / consistentlyIterations)
	}
	return true
}

func testButDontPanic(f func() bool) bool {
	defer func() { recover() }()
	return f()
}

func EventuallyVerify(timeout time.Duration, mocks ...mock.HasVerify) error {
	verified := make([]bool, len(mocks))
	numVerified := 0
	var errExample error
	Eventually(timeout, func() bool {
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

func ConsistentlyVerify(timeout time.Duration, mocks ...mock.HasVerify) error {
	var errExample error
	Consistently(timeout, func() bool {
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
