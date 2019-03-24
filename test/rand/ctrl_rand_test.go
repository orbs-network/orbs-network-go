// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package rand

import (
	"fmt"
	"github.com/orbs-network/go-mock"
	"github.com/stretchr/testify/require"
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestWithRandLogsCorrectSeedAndTestName(t *testing.T) {
	seedPreference.mode = randModeTestClockSeed
	nlMock := NewNamedLoggerMock("MockName")
	var loggedSeed int64
	nlMock.When("Log", mock.Any).Call(func(message string) {
		var err error
		tokens := strings.Split(message, " ")
		loggedSeed, err = strconv.ParseInt(tokens[2], 0, 64)
		require.NoError(t, err, "expected third word in log message to be an int64 random seed")
		require.Equal(t, "(MockName)", tokens[3], "expected fourth word in log message to be the name of the test")
	}).Times(1)
	randUint := NewControlledRand(nlMock).Uint64()
	expectedRandUint := rand.New(rand.NewSource(loggedSeed)).Uint64()
	require.Equal(t, expectedRandUint, randUint, "expected ControlledRand to log the random seed used for random source")
}

func TestWithExplicitRand(t *testing.T) {
	seedPreference.seed = 1
	seedPreference.mode = randModeExplicitSeed
	nlMock1 := NewNamedLoggerMock("MockName1")
	nlMock1.When("Log", mock.Any)
	nlMock2 := NewNamedLoggerMock("MockName2")
	randUint1 := NewControlledRand(nlMock1).Uint64()
	randUint2 := NewControlledRand(nlMock2).Uint64()
	expectedRand := rand.New(rand.NewSource(1)).Uint64()
	require.Equal(t, expectedRand, randUint1, "expected explicit random seed to produce identical random values")
	require.Equal(t, expectedRand, randUint2, "expected explicit random seed to produce identical random values")
}

func TestWithLaunchClock(t *testing.T) {
	launchClock := time.Now().UTC().UnixNano()
	seedPreference.seed = launchClock
	seedPreference.mode = randModeProcessClockSeed
	nlMock1 := NewNamedLoggerMock("MockName")
	nlMock1.When("Log", mock.Any).Call(func(message string) {
		require.EqualValues(t, fmt.Sprintf("random seed %v (MockName)", seedPreference.seed), message, "expected NewControlledRand to log the launch clock")
	}).Times(1)
	nlMock2 := NewNamedLoggerMock("MockName")
	nlMock2.When("Log", mock.Any).Call(func(message string) {
		require.EqualValues(t, fmt.Sprintf("random seed %v (MockName)", seedPreference.seed), message, "expected NewControlledRand to log the launch clock")
	}).Times(1)
	randUint1 := NewControlledRand(nlMock1).Uint64()
	randUint2 := NewControlledRand(nlMock2).Uint64()
	expectedRand := rand.New(rand.NewSource(launchClock)).Uint64()
	require.Equal(t, expectedRand, randUint1, "expected launch-clock random seed to produce identical values on each invocation")
	require.Equal(t, expectedRand, randUint2, "expected launch-clock random seed to produce identical values on each invocation")
	require.Equal(t, launchClock, seedPreference.seed, "expected seed to not change when calling NewControlledRand in LaunchClock mode")
	_, err := nlMock1.Verify()
	require.NoError(t, err)
}

func TestWithInvocationClock(t *testing.T) {
	seedPreference.mode = randModeTestClockSeed
	var loggedSeeds []int64
	nlMock1 := NewNamedLoggerMock("MockName")
	nlMock1.When("Log", mock.Any).Call(func(message string) {
		seed, _ := strconv.ParseInt(strings.Split(message, " ")[2], 0, 64)
		loggedSeeds = append(loggedSeeds, seed)
	}).Times(1)
	nlMock2 := NewNamedLoggerMock("MockName")
	nlMock2.When("Log", mock.Any).Call(func(message string) {
		seed, _ := strconv.ParseInt(strings.Split(message, " ")[2], 0, 64)
		loggedSeeds = append(loggedSeeds, seed)
	}).Times(1)
	randUint1 := NewControlledRand(nlMock1).Uint64()
	randUint2 := NewControlledRand(nlMock2).Uint64()
	require.Equal(t, rand.New(rand.NewSource(loggedSeeds[0])).Uint64(), randUint1)
	require.Equal(t, rand.New(rand.NewSource(loggedSeeds[1])).Uint64(), randUint2)
	require.NotEqual(t, loggedSeeds[0], loggedSeeds[1], "expected seed values to be different on two NewControlledRand invocations")
	require.True(t, time.Now().UTC().UnixNano()-loggedSeeds[0] < int64(1*time.Millisecond))
	_, err := nlMock1.Verify()
	require.NoError(t, err)
	_, err = nlMock2.Verify()
	require.NoError(t, err)
}

func TestNewControlledRand_AtMostOncePerTest(t *testing.T) {
	NewControlledRand(t)
	require.Panics(t, func() {
		NewControlledRand(t)
	})
}

func TestBufferedSingleRandSafety_assertFirstRand(t *testing.T) {
	randInitSafety := newBufferedSingleRandSafety(1)

	// check and store t in the buffer
	randInitSafety.assertFirstRand(t)

	require.Panics(t, func() {
		randInitSafety.assertFirstRand(t)
	}, "expected init safety to protect against duplicate")

	// check and store t1 in the buffer, causing t to be evicted since buffer size is 1
	t.Run(t.Name(), func(t1 *testing.T) {
		randInitSafety.assertFirstRand(t1)
	})

	require.NotPanics(t, func() {
		randInitSafety.assertFirstRand(t)
	}, "expected no duplicate to be detected after t has been evicted from buffer")
}

type namedLoggerMock struct {
	mock.Mock
	name string
}

func NewNamedLoggerMock(name string) *namedLoggerMock {
	res := &namedLoggerMock{name: name}
	res.When("Log", mock.Any)
	return res
}

func (t *namedLoggerMock) Log(args ...interface{}) {
	t.Mock.Called(args...)
}

func (t *namedLoggerMock) Name() string {
	return t.name
}
