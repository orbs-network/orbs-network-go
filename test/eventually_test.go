// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"github.com/orbs-network/go-mock"
	"testing"
	"time"
)

func TestEventually(t *testing.T) {
	t.Parallel()

	sem := newSemaphore(4)
	num := 1
	go func() {
		sem.waitUntilZero()
		num = 2
	}()
	ok := Eventually(EVENTUALLY_ACCEPTANCE_TIMEOUT, func() bool {
		sem.dec()
		return num == 2
	})
	if !ok {
		t.Fatal("Eventually did not discover change to 2")
	}
}

func TestConsistently(t *testing.T) {
	t.Parallel()

	sem := newSemaphore(4)
	num := 1
	go func() {
		sem.waitUntilZero()
		num = 2
	}()
	ok := Consistently(CONSISTENTLY_ACCEPTANCE_TIMEOUT, func() bool {
		sem.dec()
		return num == 1
	})
	if ok {
		t.Fatal("Consistently did not discover change to 2")
	}
}

type personMock struct {
	mock.Mock
}

func (p *personMock) GetName() string {
	return p.Called().String(0)
}

func TestEventuallyVerifySuccess(t *testing.T) {
	t.Parallel()

	p := &personMock{}
	p.When("GetName").Return("john").Times(1)
	go func() {
		time.Sleep(1 * time.Millisecond)
		p.GetName()
	}()
	err := EventuallyVerify(EVENTUALLY_ACCEPTANCE_TIMEOUT, p)
	if err != nil {
		t.Fatal("EventuallyVerify did not discover mock was eventually called")
	}
}

func TestEventuallyVerifyFailure(t *testing.T) {
	t.Parallel()

	p := &personMock{}
	p.When("GetName").Return("john").Times(1)
	err := EventuallyVerify(EVENTUALLY_ACCEPTANCE_TIMEOUT, p)
	if err == nil {
		t.Fatal("EventuallyVerify did not discover mock was not eventually called")
	}
}

func TestEventuallyVerifySuccessWithTwoMocks(t *testing.T) {
	t.Parallel()

	p1 := &personMock{}
	p1.When("GetName").Return("john").Times(1)
	p2 := &personMock{}
	p2.When("GetName").Return("smith").Times(1)
	go func() {
		time.Sleep(1 * time.Millisecond)
		p1.GetName()
		time.Sleep(15 * time.Millisecond)
		p2.GetName()
	}()
	err := EventuallyVerify(EVENTUALLY_ACCEPTANCE_TIMEOUT, p1, p2)
	if err != nil {
		t.Fatal("EventuallyVerify did not discover both mocks were eventually called")
	}
}

func TestEventuallyVerifyFailureWithTwoMocks(t *testing.T) {
	t.Parallel()

	p1 := &personMock{}
	p1.When("GetName").Return("john").Times(1)
	p2 := &personMock{}
	p2.When("GetName").Return("smith").Times(1)
	go func() {
		time.Sleep(1 * time.Millisecond)
		p1.GetName()
	}()
	err := EventuallyVerify(EVENTUALLY_ACCEPTANCE_TIMEOUT, p1, p2)
	if err == nil {
		t.Fatal("EventuallyVerify did not discover mock was not eventually called")
	}
}

func TestConsistentlyVerifySuccess(t *testing.T) {
	t.Parallel()

	p := &personMock{}
	p.When("GetName").Return("john").Times(0)
	err := ConsistentlyVerify(CONSISTENTLY_ACCEPTANCE_TIMEOUT, p)
	if err != nil {
		t.Fatal("ConsistentlyVerify discovered incorrectly that mock was called")
	}
}

func TestConsistentlyVerifyFailure(t *testing.T) {
	t.Parallel()

	p := &personMock{}
	p.When("GetName").Return("john").Times(0)
	go func() {
		time.Sleep(1 * time.Millisecond)
		p.GetName()
	}()
	err := ConsistentlyVerify(CONSISTENTLY_ACCEPTANCE_TIMEOUT, p)
	if err == nil {
		t.Fatal("ConsistentlyVerify did not discover mock was called")
	}
}

func TestConsistentlyVerifySuccessWithTwoMocks(t *testing.T) {
	t.Parallel()

	p1 := &personMock{}
	p1.When("GetName").Return("john").Times(0)
	p2 := &personMock{}
	p2.When("GetName").Return("smith").Times(0)
	err := ConsistentlyVerify(CONSISTENTLY_ACCEPTANCE_TIMEOUT, p1, p2)
	if err != nil {
		t.Fatal("ConsistentlyVerify discovered incorrectly that mocks were called")
	}
}

func TestConsistentlyVerifyFailureWithTwoMocks(t *testing.T) {
	t.Parallel()

	p1 := &personMock{}
	p1.When("GetName").Return("john").Times(0)
	p2 := &personMock{}
	p2.When("GetName").Return("smith").Times(0)
	go func() {
		time.Sleep(15 * time.Millisecond)
		p2.GetName()
	}()
	err := ConsistentlyVerify(CONSISTENTLY_ACCEPTANCE_TIMEOUT, p1, p2)
	if err == nil {
		t.Fatal("ConsistentlyVerify did not discover mock was called")
	}
}

type semaphore struct {
	c     chan bool
	value int
}

func newSemaphore(initialValue int) *semaphore {
	return &semaphore{make(chan bool), initialValue}
}

func (s *semaphore) dec() {
	if s.value > 0 {
		s.value--
		if s.value == 0 {
			close(s.c)
		}
	}
}

func (s *semaphore) waitUntilZero() {
	for range s.c {
	}
}
