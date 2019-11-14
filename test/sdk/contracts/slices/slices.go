package slices

import (
	"bytes"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"
	"math/big"
)

var PUBLIC = sdk.Export(get, check)
var SYSTEM = sdk.Export(_init)

func _init() {
}

var boolArray = []bool{true, false, true, false, false, true}
var uint32Array = []uint32{1, 10, 100, 1000, 10000, 100000, 3}
var uint64Array = []uint64{1, 10, 100, 1000, 10000, 100000, 3}
var uint256Array = []*big.Int{big.NewInt(1), big.NewInt(1000000), big.NewInt(555555555555)}
var stringArray = []string{"picture", "yourself", "in", "a", "boat", "on", "a", "river"}
var bytesArray = [][]byte{{0x11, 0x12}, {0xa, 0xb, 0xc, 0xd}, {0x1, 0x2}}
var bytes20Array = [][20]byte{{0xaa, 0xbb}, {0x11, 0x12}, {0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01,
	0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01}, {0x1, 0x2}}
var bytes32Array = [][32]byte{{0x11, 0x12}, {0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x04,
	0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x04}, {0xa, 0xb, 0xc, 0xd}, {0x1, 0x2}}


func get() ([]bool, []uint32, []uint64, []*big.Int, []string, [][]byte, [][20]byte, [][32]byte) {
	return boolArray, uint32Array, uint64Array, uint256Array, stringArray, bytesArray, bytes20Array, bytes32Array
}

func check(boolsIn []bool, uint32sIn[]uint32, uint64sIn []uint64, uint256sIn []*big.Int, stringsIn []string,
	bytesIn [][]byte, bytes20In [][20]byte, bytes32In[][32]byte) (bool, string) {
	if !checkBools(boolsIn) {
		return false, "bools"
	}
	if !checkUint32s(uint32sIn) {
		return false, "uint32s"
	}
	if !checkUint64s(uint64sIn) {
		return false, "uint64s"
	}
	if !checkUint256s(uint256sIn) {
		return false, "uint256s"
	}
	if !checkStrings(stringsIn) {
		return false, "strings"
	}
	if !checkBytes(bytesIn) {
		return false, "bytes"
	}
	if !checkBytes20s(bytes20In) {
		return false, "bytes20s"
	}
	if !checkBytes32s(bytes32In) {
		return false, "bytes32s"
	}
	return true, ""
}

func checkBools(in []bool) bool {
	if len(in) != len(boolArray) {
		return false
	}
	for i := range boolArray {
		if boolArray[i] != in[i] {
			return false
		}
	}
	return true
}

func checkUint32s(in []uint32) bool {
	if len(in) != len(uint32Array) {
		return false
	}
	for i := range uint32Array {
		if uint32Array[i] != in[i] {
			return false
		}
	}
	return true
}

func checkUint64s(in []uint64) bool {
	if len(in) != len(uint64Array) {
		return false
	}
	for i := range uint64Array {
		if uint64Array[i] != in[i] {
			return false
		}
	}
	return true
}

func checkUint256s(in []*big.Int) bool {
	if len(in) != len(uint256Array) {
		return false
	}
	for i := range uint256Array {
		if uint256Array[i].Cmp(in[i]) != 0 {
			return false
		}
	}
	return true
}

func checkStrings(in []string) bool {
	if len(in) != len(stringArray) {
		return false
	}
	for i := range stringArray {
		if stringArray[i] != in[i] {
			return false
		}
	}
	return true
}

func checkBytes(in [][]byte) bool {
	if len(in) != len(bytesArray) {
		return false
	}
	for i := range bytesArray {
		if !bytes.Equal(bytesArray[i], in[i]) {
			return false
		}
	}
	return true
}

func checkBytes20s(in [][20]byte) bool {
	if len(in) != len(bytes20Array) {
		return false
	}
	for i := range bytes20Array {
		if !bytes.Equal(bytes20Array[i][:], in[i][:]) {
			return false
		}
	}
	return true
}

func checkBytes32s(in [][32]byte) bool {
	if len(in) != len(bytes32Array) {
		return false
	}
	for i := range bytes32Array {
		if !bytes.Equal(bytes32Array[i][:], in[i][:]) {
			return false
		}
	}
	return true
}
