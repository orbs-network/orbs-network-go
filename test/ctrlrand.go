package test

import (
	"flag"
	"fmt"
	"strconv"
	"sync"
	"time"
)

func init() {
	flag.Var(&randPreference, "test.randSeed",
		"Specify a random seed for tests, or 'launchClock' to use"+
			" the same arbitrary value in each test invocation")
}

const singleRandSafetyBufferSize = 1000

var singleRandPerTestSafety = newBufferedSingleRandSafety(singleRandSafetyBufferSize)
var randPreference randomPreference // initialized in init()

type NamedLogger interface {
	Log(args ...interface{})
	Name() string
}

type controlledRandMode int

const (
	randPrefInvokeClock controlledRandMode = iota
	randPrefLaunchClock
	randPrefExplicit
)

type ControlledRand struct {
	*syncRand
}

func NewControlledRand(t NamedLogger) *ControlledRand {
	singleRandPerTestSafety.assertFirstRand(t)

	var newSeed int64
	if randPreference.mode == randPrefInvokeClock {
		newSeed = time.Now().UTC().UnixNano()
	} else {
		newSeed = randPreference.seed
	}
	t.Log(fmt.Sprintf("random seed %v (%s)", newSeed, t.Name()))

	return &ControlledRand{newSyncRand(newSeed)}
}

type randomPreference struct {
	mode controlledRandMode // default value is randPrefInvokeClock
	seed int64              // applicable only in mode != randPrefInvokeClock
}

func (i *randomPreference) String() string {
	var preference string
	switch i.mode {
	case randPrefInvokeClock:
		preference = "clock at invocation (default)"
	case randPrefLaunchClock:
		preference = fmt.Sprintf("launchClock: %v", i.seed)
	case randPrefExplicit:
		preference = fmt.Sprintf("explicit seed: %v", i.seed)
	}
	return preference
}

func (i *randomPreference) Set(value string) error {
	if value == "launchClock" {
		i.mode = randPrefLaunchClock
		i.seed = time.Now().UTC().UnixNano()
		return nil
	}
	i.mode = randPrefExplicit
	v, err := strconv.ParseInt(value, 0, 64)
	i.seed = v
	return err
}

type bufferedSingleRandSafety struct {
	sync.Mutex
	loggerLookup   map[NamedLogger]bool
	loggerBuffer   []NamedLogger
	nextWriteIndex int
}

func newBufferedSingleRandSafety(bufferSize int) *bufferedSingleRandSafety {
	return &bufferedSingleRandSafety{
		loggerLookup:   make(map[NamedLogger]bool),
		loggerBuffer:   make([]NamedLogger, bufferSize),
		nextWriteIndex: 0,
	}
}

func (ris *bufferedSingleRandSafety) assertFirstRand(t NamedLogger) {
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
