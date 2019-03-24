// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package config

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestValidateConfig(t *testing.T) {
	v := validator{log.DefaultTestingLogger(t)}
	cfg := defaultProductionConfig()
	cfg.SetNodeAddress([]byte{0x0})
	cfg.SetNodePrivateKey([]byte{0x0})
	cfg.SetGenesisValidatorNodes(map[string]ValidatorNode{
		"v1": NewHardCodedValidatorNode([]byte{0x0}),
		"v2": NewHardCodedValidatorNode([]byte{0x1}),
	})

	require.NotPanics(t, func() {
		v.ValidateNodeLogic(cfg)
	})
}

func TestValidateConfig_PanicsOnInvalidValue(t *testing.T) {
	v := validator{log.DefaultTestingLogger(t)}

	cfg := defaultProductionConfig()
	cfg.SetDuration(BLOCK_SYNC_NO_COMMIT_INTERVAL, 1*time.Millisecond)

	require.Panics(t, func() {
		v.ValidateNodeLogic(cfg)
	})
}
