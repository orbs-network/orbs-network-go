package native

import (
	"github.com/orbs-network/orbs-network-go/services/processor/sdk"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"math/big"
	"reflect"
	"testing"
)

func TestPrepareMethodArgumentsAndCall_SimpleOneInputArg(t *testing.T) {
	tests := []struct {
		name           string
		shouldErr      bool
		value          interface{}
		methodInstance interface{}
	}{
		// allowed
		{"bool", false, true, func(a bool) bool { return a }},
		{"uint32", false, uint32(50), func(a uint32) uint32 { return a }},
		{"uint64", false, uint64(50), func(a uint64) uint64 { return a }},
		{"string", false, "foo", func(a string) string { return a }},
		{"bytes20", false, [20]byte{0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01,
			0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01}, func(a [20]byte) [20]byte { return a }},
		{"bytes32", false, [32]byte{0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x04,
			0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x04}, func(a [32]byte) [32]byte { return a }},
		{"*big.Int", false, big.NewInt(55), func(a *big.Int) *big.Int { return a }},
		{"[]byte", false, []byte("hello"), func(a []byte) []byte { return a }},
		// not allowed
		{"other-byte-array", true, [30]byte{0x10}, func(a [30]byte) [30]byte { return a }},
		{"other-type-array", true, [32]int{7}, func(a [32]int) [32]int { return a }},
		{"other pointer", true, big.NewFloat(5.5), func(a *big.Float) *big.Float { return a }},
		{"[]uint32", true, []uint32{uint32(50)}, func(a []uint32) []uint32 { return a }},
	}

	for i := range tests {
		cTest := tests[i]
		args := sdk.ArgsToArgumentArray(cTest.value)
		inValues, err := prepareMethodInputArgsForCall(cTest.methodInstance, args, "funcName")
		if cTest.shouldErr {
			require.Error(t, err, "should fail to parse %s", cTest.name)
		} else {
			require.NoError(t, err, "should succeed to parse %s", cTest.name)

			outValues := reflect.ValueOf(cTest.methodInstance).Call(inValues)
			require.EqualValues(t, cTest.value, outValues[0].Interface(), "return values should be equal to input.")
		}
	}
}

func TestCreatMethodOutputArgs(t *testing.T) {
	tests := []struct {
		name           string
		shouldErr      bool
		value          interface{}
		argType        protocol.ArgumentType
		methodInstance interface{}
	}{
		// allowed
		{"bool", false, true, protocol.ARGUMENT_TYPE_BOOL_VALUE, func() bool { return false }},
		{"uint32", false, uint32(50), protocol.ARGUMENT_TYPE_UINT_32_VALUE, func() uint32 { return 0 }},
		{"uint64", false, uint64(50), protocol.ARGUMENT_TYPE_UINT_64_VALUE, func() uint64 { return 0 }},
		{"string", false, "foo", protocol.ARGUMENT_TYPE_STRING_VALUE, func() string { return "bar" }},
		{"bytes20", false, [20]byte{0x10}, protocol.ARGUMENT_TYPE_BYTES_20_VALUE, func() [20]byte { return [20]byte{} }},
		{"bytes32", false, [32]byte{0x10}, protocol.ARGUMENT_TYPE_BYTES_32_VALUE, func() [32]byte { return [32]byte{} }},
		{"*big.Int", false, big.NewInt(55), protocol.ARGUMENT_TYPE_UINT_256_VALUE, func() *big.Int { return big.NewInt(0) }},
		{"[]byte", false, []byte{0x10, 0x11}, protocol.ARGUMENT_TYPE_BYTES_VALUE, func() []byte { return []byte{} }},
		// not allowed
		{"other-byte-array", true, [30]byte{0x10}, protocol.ARGUMENT_TYPE_BYTES_32_VALUE, func() [30]byte { return [30]byte{} }},
		{"other-type-array", true, [32]int{7}, protocol.ARGUMENT_TYPE_BYTES_32_VALUE, func() [32]int { return [32]int{} }},
		{"other-pointer", true, big.NewFloat(5.5), protocol.ARGUMENT_TYPE_UINT_32_VALUE, func() *big.Float { return nil }},
		{"[]uint32", true, []uint32{uint32(50)}, protocol.ARGUMENT_TYPE_UINT_32_VALUE, func() []uint32 { return []uint32{} }},
		{"[][]byte", true, [][]byte{{0x11, 0x10}, {0x20, 0x21}}, protocol.ARGUMENT_TYPE_UINT_32_VALUE, func() [][]byte { return [][]byte{} }},
	}

	for i := range tests {
		cTest := tests[i]
		outputArgs, err := createMethodOutputArgs(cTest.methodInstance, []reflect.Value{reflect.ValueOf(cTest.value)}, "funcName")
		if cTest.shouldErr {
			require.Error(t, err, "should fail to parse %s", cTest.name)
		} else {
			require.NoError(t, err, "should succeed to parse %s", cTest.name)
			require.EqualValues(t, cTest.argType, outputArgs.ArgumentsIterator().NextArguments().Type(), "should be type %V is not", cTest.argType)
		}
	}
}

// more complex cases
func TestPrepareMethodArgumentsForCallWithTwoByteArrays(t *testing.T) {
	methodInstance := func(a []byte, b []byte) {}
	args := sdk.ArgsToArgumentArray([]byte("one"), []byte("two"))

	inValues, err := prepareMethodInputArgsForCall(methodInstance, args, "funcName")
	require.NoError(t, err)
	require.Len(t, inValues, 2)
	require.EqualValues(t, []byte("one"), inValues[0].Interface())
	require.EqualValues(t, []byte("two"), inValues[1].Interface())
}

func TestPrepareMethodArgumentsForCall_OneInputExpected_IncorrectNumberOfArgs(t *testing.T) {
	methodInstance := func(a uint32) {}

	inValues, err := prepareMethodInputArgsForCall(methodInstance, sdk.ArgsToArgumentArray(uint32(1997), uint32(1994)), "funcName")
	require.EqualError(t, err, "method 'funcName' takes 1 args but received more")
	require.Nil(t, inValues)

	inValues, err = prepareMethodInputArgsForCall(methodInstance, sdk.ArgsToArgumentArray(uint32(1997), "hello"), "funcName")
	require.EqualError(t, err, "method 'funcName' takes 1 args but received more")
	require.Nil(t, inValues)

	inValues, err = prepareMethodInputArgsForCall(methodInstance, sdk.ArgsToArgumentArray(), "funcName")
	require.EqualError(t, err, "method 'funcName' takes 1 args but received less")
	require.Nil(t, inValues)
}

func TestPrepareMethodArgumentsForCall_TwoInputExpected_IncorrectNumberOfArgs(t *testing.T) {
	methodInstance := func(a uint32, b []byte) {}

	inValues, err := prepareMethodInputArgsForCall(methodInstance, sdk.ArgsToArgumentArray(uint32(1)), "funcName")
	require.EqualError(t, err, "method 'funcName' takes 2 args but received less")
	require.Nil(t, inValues)

	inValues, err = prepareMethodInputArgsForCall(methodInstance, sdk.ArgsToArgumentArray(), "funcName")
	require.EqualError(t, err, "method 'funcName' takes 2 args but received less")
	require.Nil(t, inValues)

	inValues, err = prepareMethodInputArgsForCall(methodInstance, sdk.ArgsToArgumentArray(uint32(32), []byte{0x1}, uint32(5)), "funcName")
	require.EqualError(t, err, "method 'funcName' takes 2 args but received more")
	require.Nil(t, inValues)
}

// variadic cases
func TestPrepareMethodArgumentsForCall_WithArrayOfVariableLength(t *testing.T) {
	methodInstance := func(a ...string) {}
	args := sdk.ArgsToArgumentArray("one", "two")

	inValues, err := prepareMethodInputArgsForCall(methodInstance, args, "funcName")
	require.NoError(t, err)
	require.Len(t, inValues, 2)
	require.EqualValues(t, "one", inValues[0].Interface())
	require.EqualValues(t, "two", inValues[1].Interface())
}

func TestPrepareMethodArgumentsForCall_WithArrayOfVariableLengthPassingNoArguments(t *testing.T) {
	methodInstance := func(a ...string) {}
	args := sdk.ArgsToArgumentArray()

	inValues, err := prepareMethodInputArgsForCall(methodInstance, args, "funcName")
	require.NoError(t, err)
	require.Len(t, inValues, 0)
}

func TestPrepareMethodArgumentsForCall_WithArrayOfVariableLengthPassingArgumentsOfDifferentType(t *testing.T) {
	methodInstance := func(a uint32, b ...string) {}
	args := sdk.ArgsToArgumentArray(uint32(1), "hello", uint32(2))

	_, err := prepareMethodInputArgsForCall(methodInstance, args, "funcName")
	require.EqualError(t, err, "method 'funcName' expects arg 2 to be string but it has (Uint32Value)2")
}

func TestPrepareMethodArgumentsForCall_WithNormalArgsAndArrayOfVariableLength_EmptyInput(t *testing.T) {
	methodInstance := func(a string, b ...string) {}
	_, err := prepareMethodInputArgsForCall(methodInstance, sdk.ArgsToArgumentArray(), "funcName")
	require.EqualError(t, err, "method 'funcName' takes at least 1 args but received less")

	methodInstance2 := func(a uint32, b ...string) {}
	_, err = prepareMethodInputArgsForCall(methodInstance2, sdk.ArgsToArgumentArray(), "funcName")
	require.EqualError(t, err, "method 'funcName' takes at least 1 args but received less")

	methodInstance3 := func(a string, b string, c ...string) {}
	_, err = prepareMethodInputArgsForCall(methodInstance3, sdk.ArgsToArgumentArray("hello"), "funcName")
	require.EqualError(t, err, "method 'funcName' takes at least 2 args but received less")
}

func TestPrepareMethodArgumentsForCallWithArrayOfByteArrays(t *testing.T) {
	methodInstance := func(a ...[]byte) {}
	args := sdk.ArgsToArgumentArray([]byte("one"), []byte("two"))

	inValues, err := prepareMethodInputArgsForCall(methodInstance, args, "funcName")
	require.NoError(t, err)
	require.Len(t, inValues, 2)
	require.EqualValues(t, []byte("one"), inValues[0].Interface())
	require.EqualValues(t, []byte("two"), inValues[1].Interface())
}

func TestPrepareMethodArgumentsForCallWithArrayOfArraysOfStringsPassingTwoByteArrays(t *testing.T) {
	methodInstance := func(a ...[]string) {}
	args := sdk.ArgsToArgumentArray([]byte("one"), []byte("two"))

	_, err := prepareMethodInputArgsForCall(methodInstance, args, "funcName")
	require.EqualError(t, err, "method 'funcName' expects arg 0 to be [][]byte but it has (BytesValue)6f6e65")
}
