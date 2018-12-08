package test

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
	randPreference.mode = randPrefInvokeClock
	nlMock := NewNamedLoggerMock("MockName1")
	var loggedSeed int64
	nlMock.When("Log", mock.Any).Call(func(message string) {
		var err error
		tokens := strings.Split(message, " ")
		loggedSeed, err = strconv.ParseInt(tokens[2], 0, 64)
		require.NoError(t, err, "expected third word in log message to be an int64 random seed")
		require.Equal(t, "(MockName1)", tokens[3], "expected fourth word in log message to be the name of the test")
	}).Times(1)
	randUint1 := NewControlledRand(nlMock).Uint64()
	expectedRandUint1 := rand.New(rand.NewSource(loggedSeed)).Uint64()
	require.Equal(t, expectedRandUint1, randUint1, "expected NewControlledRand() to log the correct random seed")
}
func TestWithExplicitRand(t *testing.T) {
	randPreference.seed = 1
	randPreference.mode = randPrefExplicit
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
	randPreference.seed = launchClock
	randPreference.mode = randPrefLaunchClock
	nlMock1 := NewNamedLoggerMock("MockName")
	nlMock1.When("Log", mock.Any).Call(func(message string) {
		require.EqualValues(t, fmt.Sprintf("random seed %v (MockName)", randPreference.seed), message, "expected NewControlledRand to log the launch clock")
	}).Times(1)
	nlMock2 := NewNamedLoggerMock("MockName")
	nlMock2.When("Log", mock.Any).Call(func(message string) {
		require.EqualValues(t, fmt.Sprintf("random seed %v (MockName)", randPreference.seed), message, "expected NewControlledRand to log the launch clock")
	}).Times(1)
	randUint1 := NewControlledRand(nlMock1).Uint64()
	randUint2 := NewControlledRand(nlMock2).Uint64()
	expectedRand := rand.New(rand.NewSource(launchClock)).Uint64()
	require.Equal(t, expectedRand, randUint1, "expected launch-clock random seed to produce identical values on each invocation")
	require.Equal(t, expectedRand, randUint2, "expected launch-clock random seed to produce identical values on each invocation")
	require.Equal(t, launchClock, randPreference.seed, "expected seed to not change when calling NewControlledRand in LaunchClock mode")
	_, err := nlMock1.Verify()
	require.NoError(t, err)
}
func TestWithInvocationClock(t *testing.T) {
	randPreference.mode = randPrefInvokeClock
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
