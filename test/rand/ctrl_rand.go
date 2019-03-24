// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package rand

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// seedPreference is meant to be specified at command line, parse test.randSeed flag
func init() {
	envRandSeedPref := strings.Trim(os.Getenv("TEST_RAND_SEED"), " \t")
	if envRandSeedPref != "" {
		seedPreference.Set(envRandSeedPref)
	}
}

const singleRandSafetyBufferSize = 1000

var singleRandPerTestSafety = newBufferedSingleRandSafety(singleRandSafetyBufferSize)
var seedPreference randomSeedPref // initialized in init()

type testLogger interface {
	Log(args ...interface{})
	Name() string
}

// randomMode represents the global policy mode for selecting test random seeds
type randomMode int

const (
	randModeTestClockSeed    randomMode = iota // default. seed is the clock at each test instance
	randModeProcessClockSeed                   // seed is the clock at process launch. same for every test instance
	randModeExplicitSeed                       // seed is specified at command line, and reset in each test instance
)

// tests should use this random object when generating random values
type ControlledRand struct {
	*syncRand
}

func NewControlledRand(t testLogger) *ControlledRand {
	singleRandPerTestSafety.assertFirstRand(t)

	var newSeed int64
	if seedPreference.mode == randModeTestClockSeed {
		newSeed = time.Now().UTC().UnixNano()
	} else {
		newSeed = seedPreference.seed
	}
	t.Log(fmt.Sprintf("random seed %v (%s)", newSeed, t.Name()))

	return &ControlledRand{newSyncRand(newSeed)}
}

// randomSeedPref implements flag.Value to parse the desired random seed preference
type randomSeedPref struct {
	mode randomMode
	seed int64 // applicable only if mode != randModeTestClockSeed
}

func (i *randomSeedPref) String() string {
	var preference string
	switch i.mode {
	case randModeTestClockSeed:
		preference = "clock at invocation (default)"
	case randModeProcessClockSeed:
		preference = fmt.Sprintf("launchClock: %v", i.seed)
	case randModeExplicitSeed:
		preference = fmt.Sprintf("explicit seed: %v", i.seed)
	}
	return preference
}

func (i *randomSeedPref) Set(value string) error {
	if value == "launchClock" {
		i.mode = randModeProcessClockSeed
		i.seed = time.Now().UTC().UnixNano()
		return nil
	}
	i.mode = randModeExplicitSeed
	v, err := strconv.ParseInt(value, 0, 64)
	i.seed = v
	return err
}

// bufferedSingleRandSafety.assertFirstRand() will panic
// when the same test object initializes two random objects
// each test instance should instantiate at most one ControlledRand
type bufferedSingleRandSafety struct {
	sync.Mutex
	loggerLookup   map[testLogger]bool
	loggerBuffer   []testLogger
	nextWriteIndex int
}

func newBufferedSingleRandSafety(bufferSize int) *bufferedSingleRandSafety {
	return &bufferedSingleRandSafety{
		loggerLookup:   make(map[testLogger]bool),
		loggerBuffer:   make([]testLogger, bufferSize),
		nextWriteIndex: 0,
	}
}

func (ris *bufferedSingleRandSafety) assertFirstRand(t testLogger) {
	ris.Lock()
	defer ris.Unlock()

	if ris.loggerLookup[t] {
		panic("ControlledRand should be instantiated at most once in each test")
	}

	// update buffer
	loggerToEvict := ris.loggerBuffer[ris.nextWriteIndex]
	ris.loggerBuffer[ris.nextWriteIndex] = t

	// update lookup
	if loggerToEvict != nil {
		delete(ris.loggerLookup, loggerToEvict)
	}
	ris.loggerLookup[t] = true

	ris.nextWriteIndex = (ris.nextWriteIndex + 1) % len(ris.loggerBuffer)
}
