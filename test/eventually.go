// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

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
	var errExample error
	Eventually(timeout, func() bool {
		numVerified := 0
		errExample = nil
		for _, mock := range mocks {
			v, err := mock.Verify()
			if v {
				numVerified++
			} else {
				errExample = err
			}
		}
		return numVerified == len(mocks)
	})
	if errExample != nil {
		return errExample
	}
	return nil
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
