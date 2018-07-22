package config

import "github.com/orbs-network/orbs-spec/types/go/primitives"

type NodeConfig interface {
	NodePublicKey() primitives.Ed25519Pkey
	NetworkSize(asOfBlock uint64) uint32
	ConstantConsensusLeader() primitives.Ed25519Pkey
}
