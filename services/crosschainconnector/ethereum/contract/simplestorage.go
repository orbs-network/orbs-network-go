// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

// this was generated by abigen and then edited a bit to fit our tests
// most of the code is not used and mainly here so we can deploy this in the simulator

package contract

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// SimpleStorageABI is the input ABI used to generate the binding from.
const SimpleStorageABI = "[{\"constant\":true,\"inputs\":[],\"name\":\"getValues\",\"outputs\":[{\"name\":\"intValue\",\"type\":\"uint256\"},{\"name\":\"stringValue\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"getInt\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_multiple\",\"type\":\"uint256\"}],\"name\":\"getIntMultiple\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"getString\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"_intValue\",\"type\":\"uint256\"},{\"name\":\"_stringValue\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"}]"

// SimpleStorageBin is the compiled bytecode used for deploying new contracts.
const SimpleStorageBin = `0x608060405234801561001057600080fd5b5060405161043938038061043983398101604052805160208201519091016100418282640100000000610048810204565b5050610100565b60008290558051610060906001906020840190610065565b505050565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f106100a657805160ff19168380011785556100d3565b828001600101855582156100d3579182015b828111156100d35782518255916020019190600101906100b8565b506100df9291506100e3565b5090565b6100fd91905b808211156100df57600081556001016100e9565b90565b61032a8061010f6000396000f3006080604052600436106100615763ffffffff7c010000000000000000000000000000000000000000000000000000000060003504166319eb4a90811461006657806362738998146100fa57806382fa8ab21461012157806389ea642f14610139575b600080fd5b34801561007257600080fd5b5061007b6101c3565b6040518083815260200180602001828103825283818151815260200191508051906020019080838360005b838110156100be5781810151838201526020016100a6565b50505050905090810190601f1680156100eb5780820380516001836020036101000a031916815260200191505b50935050505060405180910390f35b34801561010657600080fd5b5061010f61025c565b60408051918252519081900360200190f35b34801561012d57600080fd5b5061010f600435610262565b34801561014557600080fd5b5061014e610269565b6040805160208082528351818301528351919283929083019185019080838360005b83811015610188578181015183820152602001610170565b50505050905090810190601f1680156101b55780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b60005460018054604080516020601f600260001961010087891615020190951694909404938401819004810282018101909252828152606093909290918301828280156102515780601f1061022657610100808354040283529160200191610251565b820191906000526020600020905b81548152906001019060200180831161023457829003601f168201915b505050505090509091565b60005490565b6000540290565b60018054604080516020601f600260001961010087891615020190951694909404938401819004810282018101909252828152606093909290918301828280156102f45780601f106102c9576101008083540402835291602001916102f4565b820191906000526020600020905b8154815290600101906020018083116102d757829003601f168201915b50505050509050905600a165627a7a723058201a962a26053e2848fa68cd6b898b846d9976529ad3e69319204f3307757a92f40029`

// DeploySimpleStorage deploys a new Ethereum contract, binding an instance of SimpleStorage to it.
func DeploySimpleStorage(auth *bind.TransactOpts, backend bind.ContractBackend, _intValue *big.Int, _stringValue string) (common.Address, *types.Transaction, *SimpleStorage, error) {
	parsed, err := abi.JSON(strings.NewReader(SimpleStorageABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(SimpleStorageBin), backend, _intValue, _stringValue)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &SimpleStorage{SimpleStorageCaller: SimpleStorageCaller{contract: contract}, SimpleStorageTransactor: SimpleStorageTransactor{contract: contract}, SimpleStorageFilterer: SimpleStorageFilterer{contract: contract}}, nil
}

// SimpleStorage is an auto generated Go binding around an Ethereum contract.
type SimpleStorage struct {
	SimpleStorageCaller     // Read-only binding to the contract
	SimpleStorageTransactor // Write-only binding to the contract
	SimpleStorageFilterer   // Log filterer for contract events
}

// SimpleStorageCaller is an auto generated read-only Go binding around an Ethereum contract.
type SimpleStorageCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SimpleStorageTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SimpleStorageTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SimpleStorageFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SimpleStorageFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SimpleStorageSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SimpleStorageSession struct {
	Contract     *SimpleStorage    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// SimpleStorageCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SimpleStorageCallerSession struct {
	Contract *SimpleStorageCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// SimpleStorageTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SimpleStorageTransactorSession struct {
	Contract     *SimpleStorageTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// SimpleStorageRaw is an auto generated low-level Go binding around an Ethereum contract.
type SimpleStorageRaw struct {
	Contract *SimpleStorage // Generic contract binding to access the raw methods on
}

// SimpleStorageCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SimpleStorageCallerRaw struct {
	Contract *SimpleStorageCaller // Generic read-only contract binding to access the raw methods on
}

// SimpleStorageTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SimpleStorageTransactorRaw struct {
	Contract *SimpleStorageTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSimpleStorage creates a new instance of SimpleStorage, bound to a specific deployed contract.
func NewSimpleStorage(address common.Address, backend bind.ContractBackend) (*SimpleStorage, error) {
	contract, err := bindSimpleStorage(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &SimpleStorage{SimpleStorageCaller: SimpleStorageCaller{contract: contract}, SimpleStorageTransactor: SimpleStorageTransactor{contract: contract}, SimpleStorageFilterer: SimpleStorageFilterer{contract: contract}}, nil
}

// NewSimpleStorageCaller creates a new read-only instance of SimpleStorage, bound to a specific deployed contract.
func NewSimpleStorageCaller(address common.Address, caller bind.ContractCaller) (*SimpleStorageCaller, error) {
	contract, err := bindSimpleStorage(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SimpleStorageCaller{contract: contract}, nil
}

// NewSimpleStorageTransactor creates a new write-only instance of SimpleStorage, bound to a specific deployed contract.
func NewSimpleStorageTransactor(address common.Address, transactor bind.ContractTransactor) (*SimpleStorageTransactor, error) {
	contract, err := bindSimpleStorage(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SimpleStorageTransactor{contract: contract}, nil
}

// NewSimpleStorageFilterer creates a new log filterer instance of SimpleStorage, bound to a specific deployed contract.
func NewSimpleStorageFilterer(address common.Address, filterer bind.ContractFilterer) (*SimpleStorageFilterer, error) {
	contract, err := bindSimpleStorage(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SimpleStorageFilterer{contract: contract}, nil
}

// bindSimpleStorage binds a generic wrapper to an already deployed contract.
func bindSimpleStorage(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(SimpleStorageABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SimpleStorage *SimpleStorageRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _SimpleStorage.Contract.SimpleStorageCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SimpleStorage *SimpleStorageRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SimpleStorage.Contract.SimpleStorageTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SimpleStorage *SimpleStorageRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SimpleStorage.Contract.SimpleStorageTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_SimpleStorage *SimpleStorageCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _SimpleStorage.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_SimpleStorage *SimpleStorageTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _SimpleStorage.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_SimpleStorage *SimpleStorageTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _SimpleStorage.Contract.contract.Transact(opts, method, params...)
}

// GetInt is a free data retrieval call binding the contract method 0x62738998.
//
// Solidity: function getInt() constant returns(uint256)
func (_SimpleStorage *SimpleStorageCaller) GetInt(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _SimpleStorage.contract.Call(opts, out, "getInt")
	return *ret0, err
}

// GetInt is a free data retrieval call binding the contract method 0x62738998.
//
// Solidity: function getInt() constant returns(uint256)
func (_SimpleStorage *SimpleStorageSession) GetInt() (*big.Int, error) {
	return _SimpleStorage.Contract.GetInt(&_SimpleStorage.CallOpts)
}

// GetInt is a free data retrieval call binding the contract method 0x62738998.
//
// Solidity: function getInt() constant returns(uint256)
func (_SimpleStorage *SimpleStorageCallerSession) GetInt() (*big.Int, error) {
	return _SimpleStorage.Contract.GetInt(&_SimpleStorage.CallOpts)
}

// GetIntMultiple is a free data retrieval call binding the contract method 0x82fa8ab2.
//
// Solidity: function getIntMultiple(_multiple uint256) constant returns(uint256)
func (_SimpleStorage *SimpleStorageCaller) GetIntMultiple(opts *bind.CallOpts, _multiple *big.Int) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _SimpleStorage.contract.Call(opts, out, "getIntMultiple", _multiple)
	return *ret0, err
}

// GetIntMultiple is a free data retrieval call binding the contract method 0x82fa8ab2.
//
// Solidity: function getIntMultiple(_multiple uint256) constant returns(uint256)
func (_SimpleStorage *SimpleStorageSession) GetIntMultiple(_multiple *big.Int) (*big.Int, error) {
	return _SimpleStorage.Contract.GetIntMultiple(&_SimpleStorage.CallOpts, _multiple)
}

// GetIntMultiple is a free data retrieval call binding the contract method 0x82fa8ab2.
//
// Solidity: function getIntMultiple(_multiple uint256) constant returns(uint256)
func (_SimpleStorage *SimpleStorageCallerSession) GetIntMultiple(_multiple *big.Int) (*big.Int, error) {
	return _SimpleStorage.Contract.GetIntMultiple(&_SimpleStorage.CallOpts, _multiple)
}

// GetString is a free data retrieval call binding the contract method 0x89ea642f.
//
// Solidity: function getString() constant returns(string)
func (_SimpleStorage *SimpleStorageCaller) GetString(opts *bind.CallOpts) (string, error) {
	var (
		ret0 = new(string)
	)
	out := ret0
	err := _SimpleStorage.contract.Call(opts, out, "getString")
	return *ret0, err
}

// GetString is a free data retrieval call binding the contract method 0x89ea642f.
//
// Solidity: function getString() constant returns(string)
func (_SimpleStorage *SimpleStorageSession) GetString() (string, error) {
	return _SimpleStorage.Contract.GetString(&_SimpleStorage.CallOpts)
}

// GetString is a free data retrieval call binding the contract method 0x89ea642f.
//
// Solidity: function getString() constant returns(string)
func (_SimpleStorage *SimpleStorageCallerSession) GetString() (string, error) {
	return _SimpleStorage.Contract.GetString(&_SimpleStorage.CallOpts)
}

// GetValues is a free data retrieval call binding the contract method 0x19eb4a90.
//
// Solidity: function getValues() constant returns(intValue uint256, stringValue string)
func (_SimpleStorage *SimpleStorageCaller) GetValues(opts *bind.CallOpts) (struct {
	IntValue    *big.Int
	StringValue string
}, error) {
	ret := new(struct {
		IntValue    *big.Int
		StringValue string
	})
	out := ret
	err := _SimpleStorage.contract.Call(opts, out, "getValues")
	return *ret, err
}

// GetValues is a free data retrieval call binding the contract method 0x19eb4a90.
//
// Solidity: function getValues() constant returns(intValue uint256, stringValue string)
func (_SimpleStorage *SimpleStorageSession) GetValues() (struct {
	IntValue    *big.Int
	StringValue string
}, error) {
	return _SimpleStorage.Contract.GetValues(&_SimpleStorage.CallOpts)
}

// GetValues is a free data retrieval call binding the contract method 0x19eb4a90.
//
// Solidity: function getValues() constant returns(intValue uint256, stringValue string)
func (_SimpleStorage *SimpleStorageCallerSession) GetValues() (struct {
	IntValue    *big.Int
	StringValue string
}, error) {
	return _SimpleStorage.Contract.GetValues(&_SimpleStorage.CallOpts)
}
