//+build !race

package e2e

import (
	"github.com/orbs-network/orbs-network-go/services/processor/javascript/test"
)

const (
	DUMMY_PLUGIN_SOURCE = "services/processor/plugins/dummy/"
	DUMMY_PLUGIN_BINARY = "test/e2e/dummy_plugin.bin"
)

func buildDummyPlugin() {
	test.BuildDummyPlugin(DUMMY_PLUGIN_SOURCE, DUMMY_PLUGIN_BINARY)
}

func removeDummyPlugin() {
	test.RemoveDummyPlugin(DUMMY_PLUGIN_BINARY)
}

func dummyPluginPath() string {
	return test.DummyPluginPath(DUMMY_PLUGIN_BINARY)
}
