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

	benchmarkConsensusRetryInterval := 5000 * time.Millisecond
	minimumTransactionsInBlock := uint32(1)
	minimalBlockDelay := 1000 * time.Millisecond // this is the time between empty blocks when no transactions, need to be large so we don't close infinite blocks on idle
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
		minimalBlockDelay,
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
	minimalBlockDelay := 10 * time.Millisecond
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

func ForDevelopment(
	federationNodes map[string]FederationNode,
	nodePublicKey primitives.Ed25519PublicKey,
	nodePrivateKey primitives.Ed25519PrivateKey,
	constantConsensusLeader primitives.Ed25519PublicKey,
	activeConsensusAlgo consensus.ConsensusAlgoType,
) NodeConfig {

	benchmarkConsensusRetryInterval := 1000 * time.Millisecond
	minimumTransactionsInBlock := uint32(1)
	minimalBlockDelay := 500 * time.Millisecond // this is the time between empty blocks when no transactions, need to be large so we don't close infinite blocks on idle
	queryGraceTimeout := 100 * time.Millisecond
	sendTransactionTimeout := 10 * time.Second
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
