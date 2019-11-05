// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.
//
// +build !race
// +build javascript

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
