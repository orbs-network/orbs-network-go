// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test/rand"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPersistenceAdapter_DetectsVirtualChainMismatch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Integration tests in short mode")
	}

	conf := newTempFileConfig()
	defer conf.cleanDir()

	writeRandomBlocksToFile(t, conf, 1, rand.NewControlledRand(t))

	conf.setVirtualChainId(conf.VirtualChainId() + 1)

	_, _, err := NewFilesystemAdapterDriver(log.DefaultTestingLogger(t), conf)
	require.Error(t, err, "expected error when trying to open a blocks file from a different virtual chain")
}
