// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

// +build memoryleak

package acceptance

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
)

// as we are using a build flag, and we want to avoid logging in the stress test
// as the harness will cache them because of t.Log, we have this conditional compilation for creating the harness
func getStressTestHarness() *networkHarnessBuilder {
	return newHarness().WithLogFilters(log.DiscardAll())
}
