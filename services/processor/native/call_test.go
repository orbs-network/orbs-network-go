package native

import (
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"math/big"
	"reflect"
	"testing"
)

func TestProcessMethodCall_Basic(t *testing.T) {
	tests := []struct {
		name           string
		shouldErr      bool
		shouldFuncErr  bool
		value          []interface{}
		methodInstance interface{}
	}{
		{"simple success", false, false, builders.VarsToSlice("hello", "friend"), func(a, b string) bool { return true }},
		{"simple wrong order", true, false, builders.VarsToSlice("hello", "friend"), func(a uint32, b string) bool { return true }},
		{"internal error", false, true, builders.VarsToSlice("hello", "friend"), func(a, b string) bool { panic("x") }},
		{"wrong output var", true, false, builders.VarsToSlice("hello", "friend"), func(a, b string) float32 { return 1.1 }},
	}

	for i := range tests {
		cTest := tests[i]
		args, err := protocol.ArgumentArrayFromNatives(cTest.value)
		require.NoError(t, err, "should succeed to parse input arg in %s", cTest.name)

		_, internalErr, err := processMethodCall(nil, nil, cTest.methodInstance, args, "funcName")
		if cTest.shouldErr {
			require.Error(t, err, "should fail in the parse parts %s", cTest.name)
		} else 	if cTest.shouldFuncErr {
			require.NoError(t, err, "should not fail because of parseing in %s", cTest.name)
			require.Error(t, internalErr, "should fail in the internal func %s", cTest.name)
		} else {
			require.NoError(t, err, "should succeed %s", cTest.name)
		}
	}
}

func TestVerifyMethodInputArgs_SimpleOneInputArg(t *testing.T) {
	tests := []struct {
		name           string
		value          interface{}
		methodInstance interface{}
	}{
		// allowed as arguments
		{"bool", true, func(a bool) bool { return a }},
		{"uint32", uint32(50), func(a uint32) uint32 { return a }},
		{"uint64", uint64(50), func(a uint64) uint64 { return a }},
		{"string", "foo", func(a string) string { return a }},
		{"bytes20", [20]byte{0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01,
			0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01}, func(a [20]byte) [20]byte { return a }},
		{"bytes32", [32]byte{0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x04,
			0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x04}, func(a [32]byte) [32]byte { return a }},
		{"*big.Int", big.NewInt(55), func(a *big.Int) *big.Int { return a }},
		{"[]byte", []byte("hello"), func(a []byte) []byte { return a }},
		// not allowed as arguments types but allowed in verify
		{"other-byte-array", [30]byte{0x10}, func(a [30]byte) [30]byte { return a }},
		{"other-type-array", [32]int{7}, func(a [32]int) [32]int { return a }},
		{"other pointer", big.NewFloat(5.5), func(a *big.Float) *big.Float { return a }},
		{"[]uint32", []uint32{uint32(50)}, func(a []uint32) []uint32 { return a }},
	}

	for i := range tests {
		cTest := tests[i]
		inValues, err := verifyMethodInputArgs(cTest.methodInstance, "funcName", builders.VarsToSlice(cTest.value))
		require.NoError(t, err, "should succeed to parse %s", cTest.name)
		outValues := reflect.ValueOf(cTest.methodInstance).Call(inValues)
		require.EqualValues(t, cTest.value, outValues[0].Interface(), "return values should be equal to input.")
	}
}

// more complex cases
func TestVerifyMethodInputArgs_WithTwoByteArrays(t *testing.T) {
	methodInstance := func(a []byte, b []byte) {}
	args := builders.VarsToSlice([]byte("one"), []byte("two"))

	inValues, err := verifyMethodInputArgs(methodInstance, "funcName", args)
	require.NoError(t, err)
	require.Len(t, inValues, 2)
	require.EqualValues(t, []byte("one"), inValues[0].Interface())
	require.EqualValues(t, []byte("two"), inValues[1].Interface())
}

func TestVerifyMethodInputArgs_OneInputExpected_IncorrectNumberOfArgs(t *testing.T) {
	methodInstance := func(a uint32) {}

	inValues, err := verifyMethodInputArgs(methodInstance, "funcName", builders.VarsToSlice(uint32(1997), uint32(1994)))
	require.EqualError(t, err, "method 'funcName' takes 1 args but received more")
	require.Nil(t, inValues)

	inValues, err = verifyMethodInputArgs(methodInstance, "funcName", builders.VarsToSlice(uint32(1997), "hello"))
	require.EqualError(t, err, "method 'funcName' takes 1 args but received more")
	require.Nil(t, inValues)

	inValues, err = verifyMethodInputArgs(methodInstance, "funcName", builders.VarsToSlice())
	require.EqualError(t, err, "method 'funcName' takes 1 args but received less")
	require.Nil(t, inValues)
}

func TestVerifyMethodInputArgs_TwoInputExpected_IncorrectNumberOfArgs(t *testing.T) {
	methodInstance := func(a uint32, b []byte) {}

	inValues, err := verifyMethodInputArgs(methodInstance, "funcName", builders.VarsToSlice(uint32(1)))
	require.EqualError(t, err, "method 'funcName' takes 2 args but received less")
	require.Nil(t, inValues)

	inValues, err = verifyMethodInputArgs(methodInstance, "funcName", builders.VarsToSlice())
	require.EqualError(t, err, "method 'funcName' takes 2 args but received less")
	require.Nil(t, inValues)

	inValues, err = verifyMethodInputArgs(methodInstance, "funcName", builders.VarsToSlice(uint32(32), []byte{0x1}, uint32(5)))
	require.EqualError(t, err, "method 'funcName' takes 2 args but received more")
	require.Nil(t, inValues)
}

// variadic cases
func TestVerifyMethodInputArgs_WithArrayOfVariableLength(t *testing.T) {
	methodInstance := func(a ...string) {}
	args := builders.VarsToSlice("one", "two")

	inValues, err := verifyMethodInputArgs(methodInstance, "funcName", args)
	require.NoError(t, err)
	require.Len(t, inValues, 2)
	require.EqualValues(t, "one", inValues[0].Interface())
	require.EqualValues(t, "two", inValues[1].Interface())
}

func TestVerifyMethodInputArgs_WithArrayOfVariableLengthPassingNoArguments(t *testing.T) {
	methodInstance := func(a ...string) {}
	args := builders.VarsToSlice()

	inValues, err := verifyMethodInputArgs(methodInstance, "funcName", args)
	require.NoError(t, err)
	require.Len(t, inValues, 0)
}

func TestVerifyMethodInputArgs_WithArrayOfVariableLengthPassingArgumentsOfDifferentType(t *testing.T) {
	methodInstance := func(a uint32, b ...string) {}
	args := builders.VarsToSlice(uint32(1), "hello", uint32(2))

	_, err := verifyMethodInputArgs(methodInstance, "funcName", args)
	require.EqualError(t, err, "method 'funcName' expects arg 2 to be string but it has uint32")
}

func TestVerifyMethodInputArgs_WithNormalArgsAndArrayOfVariableLength_EmptyInput(t *testing.T) {
	methodInstance := func(a string, b ...string) {}
	_, err := verifyMethodInputArgs(methodInstance, "funcName", builders.VarsToSlice())
	require.EqualError(t, err, "method 'funcName' takes at least 1 args but received less")

	methodInstance2 := func(a uint32, b ...string) {}
	_, err = verifyMethodInputArgs(methodInstance2, "funcName", builders.VarsToSlice())
	require.EqualError(t, err, "method 'funcName' takes at least 1 args but received less")

	methodInstance3 := func(a string, b string, c ...string) {}
	_, err = verifyMethodInputArgs(methodInstance3, "funcName", builders.VarsToSlice("hello"))
	require.EqualError(t, err, "method 'funcName' takes at least 2 args but received less")
}

func TestVerifyMethodInputArgs_WithArrayOfByteArrays(t *testing.T) {
	methodInstance := func(a ...[]byte) {}
	args := builders.VarsToSlice([]byte("one"), []byte("two"))

	inValues, err := verifyMethodInputArgs(methodInstance, "funcName", args)
	require.NoError(t, err)
	require.Len(t, inValues, 2)
	require.EqualValues(t, []byte("one"), inValues[0].Interface())
	require.EqualValues(t, []byte("two"), inValues[1].Interface())
}

// Checks that show verify is not related to argumentTypes
func TestVerifyMethodInputArgs_WithArrayOfArraysOfStrings(t *testing.T) {
	methodInstance := func(a ...[]string) {}
	args := builders.VarsToSlice([]string{"one"}, []string{"two"})

	inValues, err := verifyMethodInputArgs(methodInstance, "funcName", args)
	require.NoError(t, err)
	require.Len(t, inValues, 2)
	require.EqualValues(t, []string{"one"}, inValues[0].Interface())
	require.EqualValues(t, []string{"two"}, inValues[1].Interface())
}

func TestVerifyMethodInputArgs_WithArrayOfArraysOfStringsPassingTwoByteArrays(t *testing.T) {
	methodInstance := func(a ...[]string) {}
	args := builders.VarsToSlice([]byte("one"), []byte("two"))

	_, err := verifyMethodInputArgs(methodInstance, "funcName", args)
	require.EqualError(t, err, "method 'funcName' expects arg 0 to be []string but it has []uint8")
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
		outputArgs, err := createMethodOutputArgs([]reflect.Value{reflect.ValueOf(cTest.value)}, "funcName")
		if cTest.shouldErr {
			require.Error(t, err, "should fail to parse %s", cTest.name)
		} else {
			require.NoError(t, err, "should succeed to parse %s", cTest.name)
			require.EqualValues(t, cTest.argType, outputArgs.ArgumentsIterator().NextArguments().Type(), "should be type %V is not", cTest.argType)
		}
	}
}

