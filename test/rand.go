package test

import (
	"flag"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"
)

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

func init() {
	flag.Var(&randPreference, "test.randSeed",
		"Specify a random seed for tests, or 'launchClock' to use"+
			" the same arbitrary value in each test invocation")
}

var duplicateRandInitSafety = struct {
	sync.Mutex
	ts map[NamedLogger]bool
}{ts: make(map[NamedLogger]bool)}

func NewControlledRand(t NamedLogger) *ControlledRand {
	duplicateRandInitSafety.Lock()
	defer duplicateRandInitSafety.Unlock()
	if duplicateRandInitSafety.ts[t] {
		panic("ControlledRand should be instantiated at most once in each test")
	}

	var newSeed int64
	if randPreference.mode == randPrefInvokeClock {
		newSeed = time.Now().UTC().UnixNano()
	} else {
		newSeed = randPreference.seed
	}
	t.Log(fmt.Sprintf("random seed %v (%s)", newSeed, t.Name()))

	duplicateRandInitSafety.ts[t] = true
	return &ControlledRand{Rand: rand.New(rand.NewSource(newSeed))}
}

type ControlledRand struct { //TODO make this type thread safe... wrap all public calls with a lock
	*rand.Rand
}
