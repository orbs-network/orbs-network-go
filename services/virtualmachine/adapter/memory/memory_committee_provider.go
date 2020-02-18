package memory

import (
	"bytes"
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"sort"
	"sync"
)

type Config interface {
	GenesisValidatorNodes() map[string]config.ValidatorNode
}

type committeeTerm struct {
	asOf uint64
	committee []primitives.NodeAddress
}

type CommitteeProvider struct {
	logger     log.Logger
	sync.RWMutex
	committees []*committeeTerm
}

func NewCommitteeProvider(config Config, logger log.Logger) *CommitteeProvider {
	committee := getCommitteeFromConfig(config)
	return  &CommitteeProvider{committees: []*committeeTerm{{0, committee}}, logger :logger}
}


func (cp *CommitteeProvider) GetCommittee(ctx context.Context, referenceNumber uint64) ([]primitives.NodeAddress, error) {
	cp.RLock()
	defer cp.RUnlock()
	termIndex := len(cp.committees) - 1
	for ; termIndex > 0 && referenceNumber < cp.committees[termIndex].asOf ; termIndex-- {
	}
	return cp.committees[termIndex].committee, nil
}

func getCommitteeFromConfig(config Config) []primitives.NodeAddress {
	allNodes := config.GenesisValidatorNodes()
	var committee []primitives.NodeAddress

	for _, nodeAddress := range allNodes {
		committee = append(committee, nodeAddress.NodeAddress())
	}

	sort.SliceStable(committee, func(i, j int) bool {
		return bytes.Compare(committee[i], committee[j]) > 0
	})
	return committee
}

func (cp *CommitteeProvider) SetCommitteeToTestKeysWithIndices(asOf uint64, nodeIndices ...int) error {
	cp.Lock()
	defer cp.Unlock()
	if cp.committees[len(cp.committees)-1].asOf >= asOf {
		return errors.Errorf("new committee must have an 'asOf' reference bigger than %d (and not %d)", cp.committees[len(cp.committees)-1].asOf, asOf)
	}
	var committee []primitives.NodeAddress
	for _, committeeIndex := range nodeIndices {
		committee = append(committee, testKeys.EcdsaSecp256K1KeyPairForTests(committeeIndex).NodeAddress())
	}
	cp.committees = append(cp.committees, &committeeTerm{asOf, committee})
	cp.logger.Info("changing committee asof block", log.Uint64("asOfReference", asOf), log.StringableSlice("committee", committee))
	return nil
}
