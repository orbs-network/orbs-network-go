// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"fmt"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/scribe/log"
	"time"
)

const EVENTUALLY_ACCEPTANCE_TIMEOUT = 100 * time.Millisecond
const EVENTUALLY_ADAPTER_TIMEOUT = 200 * time.Millisecond
const EVENTUALLY_LOCAL_E2E_TIMEOUT = 400 * time.Millisecond
const EVENTUALLY_DOCKER_E2E_TIMEOUT = 1000 * time.Millisecond

const CONSISTENTLY_ACCEPTANCE_TIMEOUT = 100 * time.Millisecond
const CONSISTENTLY_ADAPTER_TIMEOUT = 100 * time.Millisecond
const CONSISTENTLY_LOCAL_E2E_TIMEOUT = 200 * time.Millisecond
const CONSISTENTLY_DOCKER_E2E_TIMEOUT = 500 * time.Millisecond

const eventuallyIterations = 20
const consistentlyIterations = 10

func Eventually(timeout time.Duration, f func() bool) bool {
	for i := 0; i < eventuallyIterations; i++ {
		if testButDontPanic(f) {
			return true
		}
		time.Sleep(timeout / eventuallyIterations)
	}
	return false
}

func RetryAndLog(timeout time.Duration, logger log.Logger, f func() error) error {
	var err error
	for i := 0; i < eventuallyIterations; i++ {
		if err = tryButDontPanic(f); err == nil {
			return nil
		}
		logger.Info(fmt.Sprintf("attempt %d out of %d failed with error", i, eventuallyIterations+1), log.Error(err))
		time.Sleep(timeout / eventuallyIterations)
	}
	if err = tryButDontPanic(f); err == nil {
		return nil
	}
	logger.Info(fmt.Sprintf("attempt %d out of %d failed with error", eventuallyIterations, eventuallyIterations+1), log.Error(err))
	return err
}

func Consistently(timeout time.Duration, f func() bool) bool {
	for i := 0; i < consistentlyIterations; i++ {
		if !testButDontPanic(f) {
			return false
		}
		time.Sleep(timeout / consistentlyIterations)
	}
	if !testButDontPanic(f) {
		return false
	}
	return true
}

func testButDontPanic(f func() bool) bool {
	defer func() { recover() }()
	return f()
}

func tryButDontPanic(f func() error) error {
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
