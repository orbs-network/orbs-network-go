// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.
// +build javascript

package e2e

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/services/processor/javascript/test"
	"github.com/orbs-network/orbs-network-go/test/e2e"
	"golang.org/x/net/context"
	"os"
	"testing"
	"time"
)

const TIMES_TO_RUN_EACH_TEST = 2
const DUMMY_PLUGIN_SOURCE = "services/processor/plugins/dummy/"
const DUMMY_PLUGIN_BINARY = "services/processor/javascript/test/e2e/dummy_plugin.bin"

func TestMain(m *testing.M) {
	exitCode := 0

	test.BuildDummyPlugin(DUMMY_PLUGIN_SOURCE, DUMMY_PLUGIN_BINARY)
	defer test.RemoveDummyPlugin(DUMMY_PLUGIN_BINARY)
	pluginPath := test.DummyPluginPath(DUMMY_PLUGIN_BINARY)

	config := e2e.GetConfig()
	if config.Bootstrap {
		tl := e2e.NewLoggerRandomer()

		//mgmtNetwork := e2e.NewInProcessE2EMgmtNetwork(config.MgmtVcid, tl, pluginPath)
		appNetwork := e2e.NewInProcessE2EAppNetwork(config.AppVcid, tl, pluginPath)

		exitCode = m.Run()
		appNetwork.GracefulShutdownAndWipeDisk()
		//mgmtNetwork.GracefulShutdownAndWipeDisk()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		appNetwork.WaitUntilShutdown(shutdownCtx)
		//mgmtNetwork.WaitUntilShutdown(shutdownCtx)

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

func jsEnabled() bool {
	return os.Getenv("JS_ENABLED") == "true"
}
