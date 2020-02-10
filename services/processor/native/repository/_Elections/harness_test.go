// Copyright 2019 the orbs-ethereum-contracts authors
// This file is part of the orbs-ethereum-contracts library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package elections_systemcontract

import (
	"fmt"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
	. "github.com/orbs-network/orbs-contract-sdk/go/testing/unit"
	"math/big"
)

/***
 * driver
 */

type harness struct {
	isTimeBased bool

	electionBlock uint64

	processTime  uint64
	electionTime uint64

	nextGuardianAddress      byte
	nextDelegatorAddress     byte
	nextValidatorAddress     byte
	nextValidatorOrbsAddress byte

	guardians  []*guardian
	delegators []*delegator
	validators []*validator
}

type actor struct {
	stake   int
	lockedStake int
	address [20]byte
}

type guardian struct {
	actor
	voteBlock       uint64
	votedValidators [][20]byte
	isGuardian      bool
}

func (g *guardian) withIsGuardian(isGuardian bool) *guardian {
	g.isGuardian = isGuardian
	return g
}

func (g *guardian) vote(asOfBlock uint64, validators ...*validator) {
	g.voteBlock = asOfBlock
	g.votedValidators = getValidatorAddresses(validators)
}

func getValidatorAddresses(validatorObjs []*validator) [][20]byte {
	addresses := make([][20]byte, 0)
	for _, v := range validatorObjs {
		addresses = append(addresses, v.address)
	}
	return addresses
}

type delegator struct {
	actor
	delegate [20]byte
}

func (d *delegator) withLockedStake(lockedStake int) *delegator {
	d.lockedStake = lockedStake
	return d
}

type validator struct {
	actor
	orbsAddress [20]byte
}

func newHarness(isTime bool) *harness {
	ETHEREUM_STAKE_FACTOR = big.NewInt(int64(10000))
	MIN_ELECTED_VALIDATORS = 3
	MAX_ELECTED_VALIDATORS = 10
	return &harness{isTimeBased: isTime, nextGuardianAddress: 0xa1, nextDelegatorAddress: 0xb1, nextValidatorAddress: 0xd1, nextValidatorOrbsAddress: 0xe1}
}

func newHarnessTimeBased() *harness {
	return newHarness(true)
}

func newHarnessBlockBased() *harness {
	return newHarness(false)
}

func (f *harness) addGuardian(stake int) *guardian {
	g := &guardian{actor: actor{stake: stake, address: [20]byte{f.nextGuardianAddress}}, isGuardian: true}
	f.nextGuardianAddress++
	f.guardians = append(f.guardians, g)
	return g
}

func (f *harness) addDelegator(stake int, delegate [20]byte) *delegator {
	d := &delegator{actor: actor{stake: stake, address: [20]byte{f.nextDelegatorAddress}}, delegate: delegate}
	f.nextDelegatorAddress++
	f.delegators = append(f.delegators, d)
	return d
}

func (f *harness) addValidator() *validator {
	return f.addValidatorWithStake(0)
}
func (f *harness) addValidatorWithStake(stake int) *validator {
	v := &validator{actor: actor{stake: stake, address: [20]byte{f.nextValidatorAddress}}, orbsAddress: [20]byte{f.nextValidatorOrbsAddress}}
	f.nextValidatorAddress++
	f.nextValidatorOrbsAddress++
	f.validators = append(f.validators, v)
	return v
}

func (f *harness) getBlockForElection() *validator {
	return f.addValidatorWithStake(0)
}

func (f *harness) setupOrbsStateBeforeProcessMachine() {
	_setProcessCurrentElection(f.electionTime, f.electionBlock, f.electionBlock-VOTE_VALID_PERIOD_LENGTH_IN_BLOCKS)
	f.mockDelegationsInOrbsBeforeProcessMachine()
	f.mockGuardianInOrbsBeforeProcessMachine()
	f.mockGuardianVotesInOrbsBeforeProcessMachine()
	f.mockValidatorsInOrbsBeforeProcessMachine()
}

func (f *harness) mockGuardianInOrbsBeforeProcessMachine() {
	addresses := make([][20]byte, 0, len(f.guardians))
	for _, g := range f.guardians {
		if g.isGuardian {
			addresses = append(addresses, g.address)
		}
	}
	_setGuardians(addresses)
}

func (f *harness) mockGuardianVotesInOrbsBeforeProcessMachine() {
	_setNumberOfGuardians(len(f.guardians))
	for i, guardian := range f.guardians {
		_setCandidates(guardian.address[:], guardian.votedValidators)
		if guardian.voteBlock != 0 && guardian.voteBlock >= _getProcessCurrentElectionEarliestValidVoteBlockNumber() {
			_setGuardianVoteBlockNumber(guardian.address[:], guardian.voteBlock)
		}
		_setGuardianStake(guardian.address[:], uint64(guardian.stake))
		_setGuardianAtIndex(i, guardian.address[:])
	}
}

func (f *harness) mockDelegationsInOrbsBeforeProcessMachine() {
	_setNumberOfDelegators(len(f.delegators))
	for i, d := range f.delegators {
		state.WriteBytes(_formatDelegatorAgentKey(d.address[:]), d.delegate[:])
		state.WriteBytes(_formatDelegatorIterator(i), d.address[:])
	}
}

func (f *harness) mockValidatorsInOrbsBeforeProcessMachine() {
	_setNumberOfValidators(len(f.validators))
	for i, v := range f.validators {
		state.WriteBytes(_formatValidaorIterator(i), v.address[:])
	}
}

func (f *harness) runProcessVoteMachineNtimes(maxNumberOfRuns int) ([][20]byte, int) {
	elected := _processVotingStateMachine()
	i := 0
	if maxNumberOfRuns <= 0 {
		maxNumberOfRuns = 100
	}
	for i := 0; i < maxNumberOfRuns && elected == nil; i++ {
		elected = _processVotingStateMachine()
	}
	return elected, i
}

func (f *harness) setupEthereumStateBeforeProcess(m Mockery) {
	f.setupEthereumValidatorsBeforeProcess(m)

	mockGuardiansInEthereum(m, f.electionBlock, f.guardians)
	f.setupEthereumGuardiansDataBeforeProcess(m)

	for _, d := range f.delegators {
		mockStakedAndLockedInEthereum(m, f.electionBlock, d.address, d.stake, d.lockedStake)
	}
}

func (f *harness) setupEthereumGuardiansDataBeforeProcess(m Mockery) {
	for _, a := range f.guardians {
		if a.isGuardian {
			mockGuardianVoteInEthereum(m, f.electionBlock, a.address, a.votedValidators, a.voteBlock)
			if a.voteBlock >= _getProcessCurrentElectionEarliestValidVoteBlockNumber() {
				mockStakedAndLockedInEthereum(m, f.electionBlock, a.address, a.stake, a.lockedStake)
			}
		}
	}
}

func (f *harness) setupEthereumValidatorsBeforeProcess(m Mockery) {
	if len(f.validators) != 0 {
		validatorAddresses := make([][20]byte, len(f.validators))
		for i, a := range f.validators {
			validatorAddresses[i] = a.address
			mockStakedAndLockedInEthereum(m, f.electionBlock, a.address, a.stake, a.lockedStake)
			mockValidatorOrbsAddressInEthereum(m, f.electionBlock, a.address, a.orbsAddress)
		}
		mockValidatorsInEthereum(m, f.electionBlock, validatorAddresses)
	}
}

func mockGuardianInEthereum(m Mockery, blockNumber uint64, address [20]byte, isGuardian bool) {
	m.MockEthereumCallMethodAtBlock(blockNumber, getGuardiansEthereumContractAddress(), getGuardiansAbi(), "isGuardian", func(out interface{}) {
		i, ok := out.(*bool)
		if ok {
			*i = isGuardian
		} else {
			panic(fmt.Sprintf("wrong something %s", out))
		}
	}, address)
}

func mockGuardianVoteInEthereum(m Mockery, blockNumber uint64, address [20]byte, candidates [][20]byte, voteBlockNumber uint64) {
	vote := Vote{
		ValidatorsBytes20: candidates,
		BlockNumber:       big.NewInt(int64(voteBlockNumber)),
	}
	m.MockEthereumCallMethodAtBlock(blockNumber, getVotingEthereumContractAddress(), getVotingAbi(), "getCurrentVoteBytes20", func(out interface{}) {
		i, ok := out.(*Vote)
		if ok {
			*i = vote
		} else {
			panic(fmt.Sprintf("wrong something %s", out))
		}
	}, address)
}

func mockGuardiansInEthereum(m Mockery, blockNumber uint64, guardians []*guardian) {
	addresses := make([][20]byte, 0, len(guardians))
	for _, g := range guardians {
		if g.isGuardian {
			addresses = append(addresses, g.address)
		}
	}
	m.MockEthereumCallMethodAtBlock(blockNumber, getGuardiansEthereumContractAddress(), getGuardiansAbi(), "getGuardiansBytes20", func(out interface{}) {
		ethAddresses, ok := out.(*[][20]byte)
		if ok {
			if len(addresses) > 50 {
				*ethAddresses = addresses[:50]
			} else {
				*ethAddresses = addresses
			}
		} else {
			panic(fmt.Sprintf("wrong type %s", out))
		}
	}, big.NewInt(0), big.NewInt(50))
	if len(addresses) > 50 {
		m.MockEthereumCallMethodAtBlock(blockNumber, getGuardiansEthereumContractAddress(), getGuardiansAbi(), "getGuardiansBytes20", func(out interface{}) {
			ethAddresses, ok := out.(*[][20]byte)
			if ok {
				*ethAddresses = addresses[50:]
			} else {
				panic(fmt.Sprintf("wrong type %s", out))
			}
		}, big.NewInt(50), big.NewInt(50))
	}
}

func mockValidatorsInEthereum(m Mockery, blockNumber uint64, addresses [][20]byte) {
	m.MockEthereumCallMethodAtBlock(blockNumber, getValidatorsEthereumContractAddress(), getValidatorsAbi(), "getValidatorsBytes20", func(out interface{}) {
		ethAddresses, ok := out.(*[][20]byte)
		if ok {
			*ethAddresses = addresses
		} else {
			panic(fmt.Sprintf("wrong type %s", out))
		}
	})
}

func mockValidatorOrbsAddressInEthereum(m Mockery, blockNumber uint64, validatorAddress [20]byte, orbsValidatorAddress [20]byte) {
	m.MockEthereumCallMethodAtBlock(blockNumber, getValidatorsRegistryEthereumContractAddress(), getValidatorsRegistryAbi(),
		"getOrbsAddress", func(out interface{}) {
			orbsAddress, ok := out.(*[20]byte)
			if ok {
				*orbsAddress = orbsValidatorAddress
			} else {
				panic(fmt.Sprintf("wrong something %s", out))
			}
		}, validatorAddress)
}

func mockStakedAndLockedInEthereum(m Mockery, blockNumber uint64, address [20]byte, stake int, lockedStake int) {
	mockStakeInEthereum(m, blockNumber, address, stake)
	mockLockedStakeInEthereum(m, blockNumber, address, lockedStake)
}

func mockStakeInEthereum(m Mockery, blockNumber uint64, address [20]byte, stake int) {
	stakeValue := big.NewInt(int64(stake))
	stakeValue = stakeValue.Mul(stakeValue, ETHEREUM_STAKE_FACTOR)
	m.MockEthereumCallMethodAtBlock(blockNumber, getTokenEthereumContractAddress(), getTokenAbi(), "balanceOf", func(out interface{}) {
		i, ok := out.(**big.Int)
		if ok {
			*i = stakeValue
		} else {
			panic(fmt.Sprintf("wrong something %s", out))
		}
	}, address)
}

func mockLockedStakeInEthereum(m Mockery, blockNumber uint64, address [20]byte, stake int) {
	stakeValue := big.NewInt(int64(stake))
	stakeValue = stakeValue.Mul(stakeValue, ETHEREUM_STAKE_FACTOR)
	m.MockEthereumCallMethodAtBlock(blockNumber, getStakingEthereumContractAddress(), getStakingAbi(), "getStakeBalanceOf", func(out interface{}) {
		i, ok := out.(**big.Int)
		if ok {
			*i = stakeValue
		} else {
			panic(fmt.Sprintf("wrong something %s", out))
		}
	}, address)
}

func startTimeBasedGetElectionTime() uint64 {
	switchToTimeBasedElections()
	electionDate := 2 * ELECTION_PERIOD_LENGTH_IN_NANOS
	_setElectedValidatorsTimeInNanosAtIndex(0, electionDate)
	return electionDate + ELECTION_PERIOD_LENGTH_IN_NANOS
}
