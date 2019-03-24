// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package consensuscontext

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Elections"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"time"
)

const CALL_ELECTIONS_CONTRACT_INTERVAL = 200 * time.Millisecond

func (s *service) getElectedValidators(ctx context.Context, currentBlockHeight primitives.BlockHeight) ([]primitives.NodeAddress, error) {
	lastCommittedBlockHeight := currentBlockHeight - 1

	genesisValidatorNodes := s.config.GenesisValidatorNodes()
	genesisValidatorNodesAddresses := toNodeAddresses(genesisValidatorNodes)

	// genesis
	if lastCommittedBlockHeight == 0 {
		return genesisValidatorNodesAddresses, nil
	}

	s.logger.Info("querying elected validators", log.BlockHeight(lastCommittedBlockHeight), log.Stringable("interval-between-attempts", CALL_ELECTIONS_CONTRACT_INTERVAL))
	electedValidatorsAddresses, err := s.callElectionsSystemContractUntilSuccess(ctx, lastCommittedBlockHeight)
	if err != nil {
		return nil, err
	}
	s.logger.Info("queried elected validators", log.Int("num-results", len(electedValidatorsAddresses)), log.BlockHeight(lastCommittedBlockHeight))

	// elections not active yet
	if len(electedValidatorsAddresses) == 0 {
		return genesisValidatorNodesAddresses, nil
	}

	return electedValidatorsAddresses, nil
}

func (s *service) callElectionsSystemContractUntilSuccess(ctx context.Context, blockHeight primitives.BlockHeight) ([]primitives.NodeAddress, error) {
	attempts := 1
	for {

		// exit on system shutdown
		if ctx.Err() != nil {
			return nil, errors.New("context terminated during callElectionsSystemContractUntilSuccess")
		}

		electedValidatorsAddresses, err := s.callElectionsSystemContract(ctx, blockHeight)
		if err == nil {
			return electedValidatorsAddresses, nil
		}

		// log every 500 failures
		if attempts%500 == 1 {
			if ctx.Err() == nil { // this may fail rightfully on graceful shutdown (ctx.Done), we don't want to report an error in this case
				s.logger.Error("cannot get elected validators from system contract", log.Error(err), log.BlockHeight(blockHeight), log.Int("attempts", attempts))
			}
		}

		// sleep or wait for ctx done, whichever comes first
		sleepOrShutdown, _ := context.WithTimeout(ctx, CALL_ELECTIONS_CONTRACT_INTERVAL)
		<-sleepOrShutdown.Done()

		attempts++
	}
}

func (s *service) callElectionsSystemContract(ctx context.Context, blockHeight primitives.BlockHeight) ([]primitives.NodeAddress, error) {
	systemContractName := primitives.ContractName(elections_systemcontract.CONTRACT_NAME)
	systemMethodName := primitives.MethodName(elections_systemcontract.METHOD_GET_ELECTED_VALIDATORS)

	output, err := s.virtualMachine.CallSystemContract(ctx, &services.CallSystemContractInput{
		BlockHeight:        blockHeight,
		BlockTimestamp:     0, // unfortunately we don't know the timestamp here, this limits which contract SDK API can be used
		ContractName:       systemContractName,
		MethodName:         systemMethodName,
		InputArgumentArray: (&protocol.ArgumentArrayBuilder{}).Build(),
	})
	if err != nil {
		return nil, err
	}
	if output.CallResult != protocol.EXECUTION_RESULT_SUCCESS {
		return nil, errors.Errorf("%s.%s call result is %s", systemContractName, systemMethodName, output.CallResult)
	}

	argIterator := output.OutputArgumentArray.ArgumentsIterator()
	if !argIterator.HasNext() {
		return nil, errors.Errorf("call system %s.%s returned corrupt output value", systemContractName, systemMethodName)
	}
	arg0 := argIterator.NextArguments()
	if !arg0.IsTypeBytesValue() {
		return nil, errors.Errorf("call system %s.%s returned corrupt output value", systemContractName, systemMethodName)
	}
	joinedAddresses := arg0.BytesValue()

	numAddresses := len(joinedAddresses) / digest.NODE_ADDRESS_SIZE_BYTES
	res := make([]primitives.NodeAddress, numAddresses)
	for i := 0; i < numAddresses; i++ {
		res[i] = joinedAddresses[20*i : 20*(i+1)]
	}
	return res, nil
}

func toNodeAddresses(nodes map[string]config.ValidatorNode) []primitives.NodeAddress {
	nodeAddresses := make([]primitives.NodeAddress, len(nodes))
	i := 0
	for _, value := range nodes {
		nodeAddresses[i] = value.NodeAddress()
		i++
	}
	return nodeAddresses
}
