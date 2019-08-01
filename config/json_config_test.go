// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package config

import (
	"encoding/hex"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNewFromEmptyJSON(t *testing.T) {
	cfg, err := newFromJSON(`{}`)

	require.NotNil(t, cfg)
	require.NoError(t, err)
}

func TestJSONConfigSetBoolTrue(t *testing.T) {
	cfg, err := newFromJSON(`{"lean-helix-show-debug": true}`)

	require.NotNil(t, cfg)
	require.NoError(t, err)
	require.EqualValues(t, true, cfg.LeanHelixShowDebug())
}

func TestJSONConfigSetBoolFalse(t *testing.T) {
	cfg, err := newFromJSON(`{"lean-helix-show-debug": false}`)

	require.NotNil(t, cfg)
	require.NoError(t, err)
	require.EqualValues(t, false, cfg.LeanHelixShowDebug())
}

func TestJSONConfigSetUint32(t *testing.T) {
	cfg, err := newFromJSON(`{"block-sync-num-blocks-in-batch": 999}`)

	require.NotNil(t, cfg)
	require.NoError(t, err)
	require.EqualValues(t, 999, cfg.BlockSyncNumBlocksInBatch())
}

func TestJSONConfigSetDuration(t *testing.T) {
	cfg, err := newFromJSON(`{"block-sync-collect-response-timeout": "10m"}`)

	require.NotNil(t, cfg)
	require.NoError(t, err)
	require.EqualValues(t, 10*time.Minute, cfg.BlockSyncCollectResponseTimeout())
}

func TestSetNodeAddress(t *testing.T) {
	cfg, err := newFromJSON(`{"node-address": "a328846cd5b4979d68a8c58a9bdfeee657b34de7"}`)

	keyPair := keys.EcdsaSecp256K1KeyPairForTests(0)

	require.NotNil(t, cfg)
	require.NoError(t, err)
	require.EqualValues(t, keyPair.NodeAddress(), cfg.NodeAddress())
}

func TestSetNodePrivateKey(t *testing.T) {
	cfg, err := newFromJSON(`{"node-private-key": "901a1a0bfbe217593062a054e561e708707cb814a123474c25fd567a0fe088f8"}`)

	keyPair := keys.EcdsaSecp256K1KeyPairForTests(0)

	require.NotNil(t, cfg)
	require.NoError(t, err)
	require.EqualValues(t, keyPair.PrivateKey(), cfg.NodePrivateKey())
}

func TestSetBenchmarkConsensusConstantLeader(t *testing.T) {
	cfg, err := newFromJSON(`{"benchmark-consensus-constant-leader": "d27e2e7398e2582f63d0800330010b3e58952ff6"}`)

	keyPair := keys.EcdsaSecp256K1KeyPairForTests(1)

	require.NotNil(t, cfg)
	require.NoError(t, err)
	require.EqualValues(t, keyPair.NodeAddress(), cfg.BenchmarkConsensusConstantLeader())
}

func TestSetActiveConsensusAlgo(t *testing.T) {
	cfg, err := newFromJSON(`{"active-consensus-algo": 999}`)

	require.NotNil(t, cfg)
	require.NoError(t, err)
	require.EqualValues(t, 999, cfg.ActiveConsensusAlgo())
}

func TestSetGenesisValidatorNodes(t *testing.T) {
	cfg, err := newFromJSON(`{
	"genesis-validator-addresses": [
    "a328846cd5b4979d68a8c58a9bdfeee657b34de7",
    "d27e2e7398e2582f63d0800330010b3e58952ff6",
    "6e2cb55e4cbe97bf5b1e731d51cc2c285d83cbf9"
	]
}`)

	require.NotNil(t, cfg)
	require.NoError(t, err)
	require.EqualValues(t, 3, len(cfg.GenesisValidatorNodes()))

	for k, v := range cfg.GenesisValidatorNodes() {
		t.Log(hex.EncodeToString([]byte(k)), v.NodeAddress())
	}

	keyPair := keys.EcdsaSecp256K1KeyPairForTests(0)

	node1 := &hardCodedValidatorNode{
		nodeAddress: keyPair.NodeAddress(),
	}

	require.EqualValues(t, node1, cfg.GenesisValidatorNodes()[keyPair.NodeAddress().KeyForMap()])
}

func TestSetGossipPeers(t *testing.T) {
	cfg, err := newFromJSON(`{
	"federation-nodes": [
    {"address":"a328846cd5b4979d68a8c58a9bdfeee657b34de7","ip":"192.168.199.2","port":4400},
    {"address":"d27e2e7398e2582f63d0800330010b3e58952ff6","ip":"192.168.199.3","port":4400},
    {"address":"6e2cb55e4cbe97bf5b1e731d51cc2c285d83cbf9","ip":"192.168.199.4","port":4400}
	]
}`)

	require.NotNil(t, cfg)
	require.NoError(t, err)
	require.EqualValues(t, 3, len(cfg.GossipPeers()))

	keyPair := keys.EcdsaSecp256K1KeyPairForTests(0)

	node1 := &hardCodedGossipPeer{
		gossipEndpoint: "192.168.199.2",
		gossipPort:     4400,
		hexOrbsAddress: "a328846cd5b4979d68a8c58a9bdfeee657b34de7",
	}

	require.EqualValues(t, node1, cfg.GossipPeers()[keyPair.NodeAddress().KeyForMap()])
}

func TestSetEthereumFinalityBlocksComponent(t *testing.T) {
	cfg, err := newFromJSON(`{"ethereum-finality-blocks-component": 17}`)

	require.NotNil(t, cfg)
	require.NoError(t, err)
	require.EqualValues(t, 17, cfg.EthereumFinalityBlocksComponent())
}

func TestSetGossipPort(t *testing.T) {
	cfg, err := newFromJSON(`{"gossip-port": 4500}`)

	require.NotNil(t, cfg)
	require.NoError(t, err)
	require.EqualValues(t, 4500, cfg.GossipListenPort())
}

func TestMergeWithFileConfig(t *testing.T) {
	cfg := ForProduction("/")

	require.EqualValues(t, 0, len(cfg.GenesisValidatorNodes()))

	cfg.MergeWithJSONConfig(`
{
	"lean-helix-show-debug": true,
	"profiling": true,
	"block-sync-num-blocks-in-batch": 999,
	"block-sync-collect-response-timeout": "10m",
	"node-address": "a328846cd5b4979d68a8c58a9bdfeee657b34de7",
	"node-private-key": "901a1a0bfbe217593062a054e561e708707cb814a123474c25fd567a0fe088f8",
	"benchmark-consensus-constant-leader": "a328846cd5b4979d68a8c58a9bdfeee657b34de7",
	"active-consensus-algo": 999,
	"gossip-port": 4500,
	"genesis-validator-addresses": [
    "a328846cd5b4979d68a8c58a9bdfeee657b34de7",
    "d27e2e7398e2582f63d0800330010b3e58952ff6",
    "6e2cb55e4cbe97bf5b1e731d51cc2c285d83cbf9"
	],
	"federation-nodes": [
    {"address":"a328846cd5b4979d68a8c58a9bdfeee657b34de7","ip":"192.168.199.2","port":4400},
    {"address":"d27e2e7398e2582f63d0800330010b3e58952ff6","ip":"192.168.199.3","port":4400},
    {"address":"6e2cb55e4cbe97bf5b1e731d51cc2c285d83cbf9","ip":"192.168.199.4","port":4400}
	]
}
`)

	newKeyPair := keys.EcdsaSecp256K1KeyPairForTests(0)

	require.EqualValues(t, 3, len(cfg.GenesisValidatorNodes()))
	require.EqualValues(t, true, cfg.LeanHelixShowDebug())
	require.EqualValues(t, true, cfg.Profiling())
	require.EqualValues(t, newKeyPair.NodeAddress(), cfg.NodeAddress())
}

func TestConfig_EthereumEndpoint(t *testing.T) {
	cfg, err := newFromJSON(`{"ethereum-endpoint":"http://172.31.1.100:8545"}`)
	require.NoError(t, err)

	require.EqualValues(t, "http://172.31.1.100:8545", cfg.EthereumEndpoint())
}

func TestConfig_ContractCodeSanitation(t *testing.T) {
	cfg, err := newFromJSON(`{"PROCESSOR-SANITIZE-DEPLOYED-CONTRACTS":true}`)
	require.NoError(t, err)

	require.EqualValues(t, true, cfg.ProcessorSanitizeDeployedContracts())
}
