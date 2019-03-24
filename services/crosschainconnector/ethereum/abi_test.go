// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package ethereum

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/stretchr/testify/require"
	"math/big"
	"strings"
	"testing"
)

func TestABI_PackFunctionInputNoArgs(t *testing.T) {
	sampleABI := `[{"inputs":[],"name":"say","outputs":[{"name":"","type":"string"}],"type":"function"}]`
	parsedAbi := parseABIForTests(t, sampleABI)
	methodNameInABI := "say"

	x, err := ABIPackFunctionInputArguments(parsedAbi, methodNameInABI, nil)
	require.NoError(t, err, "failed to parse and pack the ABI")
	require.Equal(t, []byte{0x95, 0x4a, 0xb4, 0xb2}, x, "output byte array mismatch")
}

func TestABI_PackFunctionInputWithArgs(t *testing.T) {
	ABIStorage := `[{"constant":true,"inputs":[],"name":"getValues","outputs":[{"name":"intValue","type":"uint256"},{"name":"stringValue","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"getInt","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"_multiple","type":"uint256"}],"name":"getIntMultiple","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"getString","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"inputs":[{"name":"_intValue","type":"uint256"},{"name":"_stringValue","type":"string"}],"payable":false,"stateMutability":"nonpayable","type":"constructor"}]`
	parsedABI := parseABIForTests(t, ABIStorage)
	param1 := big.NewInt(2)
	args := []interface{}{param1}
	methodNameInABI := "getIntMultiple"

	x, err := ABIPackFunctionInputArguments(parsedABI, methodNameInABI, args)
	require.NoError(t, err, "failed to parse and pack the ABI")
	expectedPackedBytes := []byte{0x82, 0xfa, 0x8a, 0xb2, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2}
	require.Equal(t, expectedPackedBytes, x, "output byte array mismatch")
}

func TestABI_UnpackFunctionOutput(t *testing.T) {
	ABIStorage := `[{"constant":true,"inputs":[],"name":"getValues","outputs":[{"name":"intValue","type":"uint256"},{"name":"stringValue","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"getInt","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"_multiple","type":"uint256"}],"name":"getIntMultiple","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"getString","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"inputs":[{"name":"_intValue","type":"uint256"},{"name":"_stringValue","type":"string"}],"payable":false,"stateMutability":"nonpayable","type":"constructor"}]`
	parsedAbi := parseABIForTests(t, ABIStorage)
	outputData := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 15, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 64, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 16, 97, 114, 101, 32, 98, 101, 108, 111, 110, 103, 32, 116, 111, 32, 117, 115, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

	ret := new(struct { // this is the struct this data will fit into
		IntValue    *big.Int
		StringValue string
	})

	err := ABIUnpackFunctionOutputArguments(parsedAbi, ret, "getValues", outputData)
	require.NoError(t, err, "unpack should not fail")
	require.EqualValues(t, 15, ret.IntValue.Int64(), "number part from eth")
	require.Equal(t, "are belong to us", ret.StringValue, "text part from eth")
}

func TestABI_UnpackAllEventArgument(t *testing.T) {
	ABIEvent := `[{"anonymous": false,"inputs": [{"indexed": true,"name": "tuid","type": "uint256"},{"indexed": false,"name": "value","type": "uint256"}],"name": "MyEvent","type": "event"}]`
	parsedAbi := parseABIForTests(t, ABIEvent)
	outputData := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 11, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 22}

	ret := new(struct { // this is the struct this data will fit into
		Tuid  *big.Int
		Value *big.Int
	})

	err := ABIUnpackAllEventArguments(parsedAbi, ret, "MyEvent", outputData)
	require.NoError(t, err, "unpack should not fail")
	require.EqualValues(t, 11, ret.Tuid.Int64(), "Tuid from eth")
	require.EqualValues(t, 22, ret.Value.Int64(), "Value from eth")
}

func parseABIForTests(t *testing.T, jsonAbi string) abi.ABI {
	parsedABI, err := abi.JSON(strings.NewReader(jsonAbi))
	require.NoError(t, err, "problem parsing ABI")
	return parsedABI
}
