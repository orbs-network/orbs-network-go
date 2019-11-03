// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package e2e

import (
	"fmt"
	"golang.org/x/net/context"
	"os"
	"testing"
	"time"
)

const TIMES_TO_RUN_EACH_TEST = 2

func TestMain(m *testing.M) {
	exitCode := 0

	config := getConfig()
	if config.bootstrap {
		if isRunningInCI() {
			fmt.Println("Skipping in-process E2E in CI (Docker-based E2Es are running in a separate step)")
		} else {
			tl := NewLoggerRandomer()

			mgmtNetwork := NewInProcessE2EMgmtNetwork(config.mgmtVcid, tl)
			appNetwork := NewInProcessE2EAppNetwork(config.appVcid, tl)

			exitCode = m.Run()
			appNetwork.GracefulShutdownAndWipeDisk()
			mgmtNetwork.GracefulShutdownAndWipeDisk()

			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			appNetwork.WaitUntilShutdown(shutdownCtx)
			mgmtNetwork.WaitUntilShutdown(shutdownCtx)
		}
	} else {
		exitCode = m.Run()
	}

	os.Exit(exitCode)
}

func isRunningInCI() bool {
	return os.Getenv("CI") != ""
}

func runMultipleTimes(t *testing.T, f func(t *testing.T)) {
	for i := 0; i < TIMES_TO_RUN_EACH_TEST; i++ {
		name := fmt.Sprintf("%s_#%d", t.Name(), i+1)
		t.Run(name, f)
		time.Sleep(100 * time.Millisecond) // give async processes time to separate between iterations
	}
}
