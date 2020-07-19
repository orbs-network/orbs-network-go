// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package config

import (
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestConfig_FillEmptyConfig(t *testing.T) {
	// setup
	cfg := emptyConfig()
	// execute
	mergeTest(cfg)
	// assert
	checkMerged(t, cfg)
}

func TestConfig_OverrideConfig(t *testing.T) {
	// setup
	cfg := emptyConfig()
	modifyFromJson(cfg, `
{
	"benchmark-consensus-constant-leader": "bb28846cd5b4979d68a8c58a9bdfeee657b34de7",
	"active-consensus-algo": 4,
	"node-address": "bb28846cd5b4979d68a8c58a9bdfeee657b34de7",
	"node-private-key": "bbbb1a0bfbe217593062a054e561e708707cb814a123474c25fd567a0fe088f8",
	"profiling": false,
	"block-sync-num-blocks-in-batch": 4,
	"block-sync-collect-response-timeout": "10s",
	"ethereum-endpoint":"http://0.0.0.100:8545"
}`)
	// execute
	mergeTest(cfg)
	// assert
	checkMerged(t, cfg)
}

func TestConfig_OverrideProductionConfig(t *testing.T) {
	// setup
	cfg := ForProduction("/")
	// execute
	mergeTest(cfg)
	// assert
	checkMerged(t, cfg)
}

func TestConfig_ParsesZeroValues(t *testing.T) {
	// setup
	cfg := emptyConfig()
	mergeTest(cfg)
	// execute
	modifyFromJson(cfg, `
{
	"active-consensus-algo": 0,
	"profiling": false,
	"block-sync-collect-response-timeout": "0s",
	"ethereum-endpoint":""
}`)
	// assert
	require.EqualValues(t, 0, cfg.ActiveConsensusAlgo())
	require.EqualValues(t, false, cfg.Profiling())
	require.EqualValues(t, 0, cfg.BlockSyncCollectResponseTimeout())
	require.EqualValues(t, "", cfg.EthereumEndpoint())
}

func mergeTest(cfg mutableNodeConfig) {
	modifyFromJson(cfg, `
{
	"benchmark-consensus-constant-leader": "a328846cd5b4979d68a8c58a9bdfeee657b34de7",
	"active-consensus-algo": 999,
	"node-address": "a328846cd5b4979d68a8c58a9bdfeee657b34de7",
	"node-private-key": "901a1a0bfbe217593062a054e561e708707cb814a123474c25fd567a0fe088f8",
	"profiling": true,
	"block-sync-num-blocks-in-batch": 9988,
	"block-sync-collect-response-timeout": "10m",
	"ethereum-endpoint":"http://172.31.1.100:8545"
}`)
}

func checkMerged(t *testing.T, cfg mutableNodeConfig) {
	newKeyPair := keys.EcdsaSecp256K1KeyPairForTests(0)

	require.EqualValues(t, newKeyPair.NodeAddress(), cfg.BenchmarkConsensusConstantLeader())
	require.EqualValues(t, 999, cfg.ActiveConsensusAlgo())
	require.EqualValues(t, newKeyPair.NodeAddress(), cfg.NodeAddress())
	require.EqualValues(t, newKeyPair.PrivateKey(), cfg.NodePrivateKey())
	require.EqualValues(t, true, cfg.Profiling())
	require.EqualValues(t, 9988, cfg.BlockSyncNumBlocksInBatch())
	require.EqualValues(t, 10*time.Minute, cfg.BlockSyncCollectResponseTimeout())
	require.EqualValues(t, "http://172.31.1.100:8545", cfg.EthereumEndpoint())
}
