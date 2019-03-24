// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package subscription

import (
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = abi.U256
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// SubscriptionABI is the input ABI used to generate the binding from.
const SubscriptionABI = "[{\"constant\":true,\"inputs\":[{\"name\":\"_id\",\"type\":\"bytes32\"}],\"name\":\"getSubscriptionData\",\"outputs\":[{\"name\":\"id\",\"type\":\"bytes32\"},{\"name\":\"profile\",\"type\":\"string\"},{\"name\":\"startTime\",\"type\":\"uint256\"},{\"name\":\"tokens\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"}]"

// SubscriptionBin is the compiled bytecode used for deploying new contracts.
const SubscriptionBin = `608060405234801561001057600080fd5b50610224806100206000396000f300608060405260043610610041576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff168063c005063714610046575b600080fd5b34801561005257600080fd5b50610075600480360381019080803560001916906020019092919050505061010d565b60405180856000191660001916815260200180602001848152602001838152602001828103825285818151815260200191508051906020019080838360005b838110156100cf5780820151818401526020810190506100b4565b50505050905090810190601f1680156100fc5780820380516001836020036101000a031916815260200191505b509550505050505060405180910390f35b60006060600080600085600190049050602a81141561017c578560006101346119c86101e4565b6040805190810160405280600281526020017f4234000000000000000000000000000000000000000000000000000000000000815250919081915094509450945094506101dc565b60118114156101db578560006101936103e86101e4565b6040805190810160405280600281526020017f4232000000000000000000000000000000000000000000000000000000000000815250919081915094509450945094506101dc565b5b509193509193565b6000670de0b6b3a7640000820290509190505600a165627a7a723058208395b6e2e1306754e575639f25d915962476eabb392d2cfe695cd742819052750029`

// DeploySubscription deploys a new Ethereum contract, binding an instance of Subscription to it.
func DeploySubscription(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Subscription, error) {
	parsed, err := abi.JSON(strings.NewReader(SubscriptionABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(SubscriptionBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Subscription{SubscriptionCaller: SubscriptionCaller{contract: contract}, SubscriptionTransactor: SubscriptionTransactor{contract: contract}, SubscriptionFilterer: SubscriptionFilterer{contract: contract}}, nil
}

// Subscription is an auto generated Go binding around an Ethereum contract.
type Subscription struct {
	SubscriptionCaller     // Read-only binding to the contract
	SubscriptionTransactor // Write-only binding to the contract
	SubscriptionFilterer   // Log filterer for contract events
}

// SubscriptionCaller is an auto generated read-only Go binding around an Ethereum contract.
type SubscriptionCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SubscriptionTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SubscriptionTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SubscriptionFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SubscriptionFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SubscriptionSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SubscriptionSession struct {
	Contract     *Subscription     // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// SubscriptionCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SubscriptionCallerSession struct {
	Contract *SubscriptionCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts       // Call options to use throughout this session
}

// SubscriptionTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SubscriptionTransactorSession struct {
	Contract     *SubscriptionTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// SubscriptionRaw is an auto generated low-level Go binding around an Ethereum contract.
type SubscriptionRaw struct {
	Contract *Subscription // Generic contract binding to access the raw methods on
}

// SubscriptionCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SubscriptionCallerRaw struct {
	Contract *SubscriptionCaller // Generic read-only contract binding to access the raw methods on
}

// SubscriptionTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SubscriptionTransactorRaw struct {
	Contract *SubscriptionTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSubscription creates a new instance of Subscription, bound to a specific deployed contract.
func NewSubscription(address common.Address, backend bind.ContractBackend) (*Subscription, error) {
	contract, err := bindSubscription(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Subscription{SubscriptionCaller: SubscriptionCaller{contract: contract}, SubscriptionTransactor: SubscriptionTransactor{contract: contract}, SubscriptionFilterer: SubscriptionFilterer{contract: contract}}, nil
}

// NewSubscriptionCaller creates a new read-only instance of Subscription, bound to a specific deployed contract.
func NewSubscriptionCaller(address common.Address, caller bind.ContractCaller) (*SubscriptionCaller, error) {
	contract, err := bindSubscription(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SubscriptionCaller{contract: contract}, nil
}

// NewSubscriptionTransactor creates a new write-only instance of Subscription, bound to a specific deployed contract.
func NewSubscriptionTransactor(address common.Address, transactor bind.ContractTransactor) (*SubscriptionTransactor, error) {
	contract, err := bindSubscription(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SubscriptionTransactor{contract: contract}, nil
}

// NewSubscriptionFilterer creates a new log filterer instance of Subscription, bound to a specific deployed contract.
func NewSubscriptionFilterer(address common.Address, filterer bind.ContractFilterer) (*SubscriptionFilterer, error) {
	contract, err := bindSubscription(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SubscriptionFilterer{contract: contract}, nil
}

// bindSubscription binds a generic wrapper to an already deployed contract.
func bindSubscription(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(SubscriptionABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Subscription *SubscriptionRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Subscription.Contract.SubscriptionCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Subscription *SubscriptionRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Subscription.Contract.SubscriptionTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Subscription *SubscriptionRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Subscription.Contract.SubscriptionTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Subscription *SubscriptionCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Subscription.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Subscription *SubscriptionTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Subscription.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Subscription *SubscriptionTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Subscription.Contract.contract.Transact(opts, method, params...)
}

// GetSubscriptionData is a free data retrieval call binding the contract method 0xc0050637.
//
// Solidity: function getSubscriptionData(bytes32 _id) constant returns(bytes32 id, string profile, uint256 startTime, uint256 tokens)
func (_Subscription *SubscriptionCaller) GetSubscriptionData(opts *bind.CallOpts, _id [32]byte) (struct {
	Id        [32]byte
	Profile   string
	StartTime *big.Int
	Tokens    *big.Int
}, error) {
	ret := new(struct {
		Id        [32]byte
		Profile   string
		StartTime *big.Int
		Tokens    *big.Int
	})
	out := ret
	err := _Subscription.contract.Call(opts, out, "getSubscriptionData", _id)
	return *ret, err
}

// GetSubscriptionData is a free data retrieval call binding the contract method 0xc0050637.
//
// Solidity: function getSubscriptionData(bytes32 _id) constant returns(bytes32 id, string profile, uint256 startTime, uint256 tokens)
func (_Subscription *SubscriptionSession) GetSubscriptionData(_id [32]byte) (struct {
	Id        [32]byte
	Profile   string
	StartTime *big.Int
	Tokens    *big.Int
}, error) {
	return _Subscription.Contract.GetSubscriptionData(&_Subscription.CallOpts, _id)
}

// GetSubscriptionData is a free data retrieval call binding the contract method 0xc0050637.
//
// Solidity: function getSubscriptionData(bytes32 _id) constant returns(bytes32 id, string profile, uint256 startTime, uint256 tokens)
func (_Subscription *SubscriptionCallerSession) GetSubscriptionData(_id [32]byte) (struct {
	Id        [32]byte
	Profile   string
	StartTime *big.Int
	Tokens    *big.Int
}, error) {
	return _Subscription.Contract.GetSubscriptionData(&_Subscription.CallOpts, _id)
}
