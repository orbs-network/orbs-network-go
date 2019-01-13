package e2e

import (
	"fmt"
	"os"
	"testing"
	"time"
)

const TIMES_TO_RUN_EACH_TEST = 2

func TestMain(m *testing.M) {
	exitCode := 0

	bootstrap := getConfig().bootstrap

	if bootstrap {
		n := newInProcessE2ENetwork()

		exitCode = m.Run()
		n.gracefulShutdownAndWipeDisk()

	} else {
		exitCode = m.Run()
	}

	os.Exit(exitCode)
}

func runMultipleTimes(t *testing.T, f func(t *testing.T)) {
	for i := 0; i < TIMES_TO_RUN_EACH_TEST; i++ {
		name := fmt.Sprintf("%s_#%d", t.Name(), i+1)
		t.Run(name, f)
		time.Sleep(100 * time.Millisecond) // give async processes time to separate between iterations
	}
}
