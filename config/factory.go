package config

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"time"
)

func ForProduction(
	federationNodes map[string]FederationNode,
	nodePublicKey primitives.Ed25519PublicKey,
	nodePrivateKey primitives.Ed25519PrivateKey,
	constantConsensusLeader primitives.Ed25519PublicKey,
	activeConsensusAlgo consensus.ConsensusAlgoType,
) NodeConfig {

	benchmarkConsensusRetryInterval := 2000 * time.Millisecond
	minimumTransactionsInBlock := uint32(1)
	minimalBlockDelay := 20 * time.Millisecond
	queryGraceTimeout := 100 * time.Millisecond
	sendTransactionTimeout := 30 * time.Second
	return newHardCodedConfig(
		federationNodes,
		nodePublicKey,
		nodePrivateKey,
		constantConsensusLeader,
		activeConsensusAlgo,
		benchmarkConsensusRetryInterval,
		minimumTransactionsInBlock,
		minimalBlockDelay, // longer than in acceptance test because otherwise e2e flakes. TODO figure out why
		queryGraceTimeout,
		sendTransactionTimeout)

}

func ForAcceptanceTests(
	federationNodes map[string]FederationNode,
	nodePublicKey primitives.Ed25519PublicKey,
	nodePrivateKey primitives.Ed25519PrivateKey,
	constantConsensusLeader primitives.Ed25519PublicKey,
	activeConsensusAlgo consensus.ConsensusAlgoType,
) NodeConfig {

	benchmarkConsensusRetryInterval := 1 * time.Millisecond
	minimumTransactionsInBlock := uint32(1)
	minimalBlockDelay := 1 * time.Millisecond
	queryGraceTimeout := 5 * time.Millisecond
	sendTransactionTimeout := 30 * time.Millisecond
	return newHardCodedConfig(
		federationNodes,
		nodePublicKey,
		nodePrivateKey,
		constantConsensusLeader,
		activeConsensusAlgo,
		benchmarkConsensusRetryInterval,
		minimumTransactionsInBlock,
		minimalBlockDelay,
		queryGraceTimeout,
		sendTransactionTimeout)
}

func EmptyConfig() NodeConfig {
	return &config{
		kv: make(map[string]NodeConfigValue),
	}
}
