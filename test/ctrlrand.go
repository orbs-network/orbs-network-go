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

var singleRandSafety = newBufferedSingleRandSafety(singleRandSafetyBufferSize)

type NamedLogger interface {
	Log(args ...interface{})
	Name() string
}

type randMode int

const (
	randPrefInvokeClock randMode = iota
	randPrefLaunchClock
	randPrefExplicit
)

type ControlledRand struct {
	*syncRand
}

func NewControlledRand(t NamedLogger) *ControlledRand {
	singleRandSafety.assertFirstRand(t)

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
	mode randMode // default value is randPrefInvokeClock
	seed int64    // applicable only in mode != randPrefInvokeClock
}

var randPreference randomPreference

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
	loggerSet      map[NamedLogger]bool
	loggerBuffer   []NamedLogger
	bufferSize     int
	nextWriteIndex int
}

func newBufferedSingleRandSafety(bufferSize int) *bufferedSingleRandSafety {
	return &bufferedSingleRandSafety{
		loggerSet:    make(map[NamedLogger]bool),
		loggerBuffer: make([]NamedLogger, bufferSize),
		bufferSize:   bufferSize,
	}
}

func (ris *bufferedSingleRandSafety) assertFirstRand(t NamedLogger) {
	ris.Lock()
	defer ris.Unlock()

	if ris.loggerSet[t] {
		panic("ControlledRand should be instantiated at most once in each test")
	}

	loggerToEvict := ris.loggerBuffer[ris.nextWriteIndex]
	if loggerToEvict != nil {
		delete(ris.loggerSet, loggerToEvict)
	}
	ris.loggerBuffer[ris.nextWriteIndex] = t
	ris.loggerSet[t] = true
	ris.nextWriteIndex = (ris.nextWriteIndex + 1) % ris.bufferSize
}
