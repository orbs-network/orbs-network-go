package adapter

import "github.com/orbs-network/orbs-spec/types/go/primitives"

type BlockHeightReporter interface {
	IncrementTo(height primitives.BlockHeight)
}
