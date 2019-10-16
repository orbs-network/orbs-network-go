package native

import (
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

func TestPrepareMethodArgumentsForCallWithUint32(t *testing.T) {
	methodInstance := func(a uint32) uint32 {
		return a
	}

	args := builders.ArgumentsArray(uint32(1997))

	inValues, err := prepareMethodInputArgsForCall(methodInstance, args, "funcName")
	require.NoError(t, err)

	outValues := reflect.ValueOf(methodInstance).Call(inValues)
	require.EqualValues(t, 1997, outValues[0].Uint())
}

func TestPrepareMethodArgumentsForCallWithByteArray(t *testing.T) {
	methodInstance := func(a []byte) []byte {
		return a
	}

	args := builders.ArgumentsArray([]byte("hello"))

	inValues, err := prepareMethodInputArgsForCall(methodInstance, args, "funcName")
	require.NoError(t, err)

	outValues := reflect.ValueOf(methodInstance).Call(inValues)
	require.EqualValues(t, []byte("hello"), outValues[0].Bytes())
}

func TestPrepareMethodArgumentsForCallWithBytes20(t *testing.T) {
	methodInstance := func(a [20]byte) [20]byte {
		return a
	}

	val := [20]byte{0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01,
		0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01}

	args := builders.ArgumentsArray(val)

	inValues, err := prepareMethodInputArgsForCall(methodInstance, args, "funcName")
	require.NoError(t, err)

	outValues := reflect.ValueOf(methodInstance).Call(inValues)
	require.EqualValues(t, val, outValues[0].Interface().([20]byte))
}

func TestPrepareMethodArgumentsForCallWithBytes32(t *testing.T) {
	methodInstance := func(a [32]byte) [32]byte {
		return a
	}

	val := [32]byte{0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x04,
		0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x04}

	args := builders.ArgumentsArray(val)

	inValues, err := prepareMethodInputArgsForCall(methodInstance, args, "funcName")
	require.NoError(t, err)

	outValues := reflect.ValueOf(methodInstance).Call(inValues)
	require.EqualValues(t, val, outValues[0].Interface().([32]byte))
}

func TestPrepareMethodArgumentsForCallWithArrayOfVariableLength(t *testing.T) {
	methodInstance := func(a ...string) []string {
		return a
	}

	args := builders.ArgumentsArray("one", "two")

	inValues, err := prepareMethodInputArgsForCall(methodInstance, args, "funcName")
	require.NoError(t, err)

	outValues := reflect.ValueOf(methodInstance).Call(inValues)
	require.EqualValues(t, []string{"one", "two"}, outValues[0].Interface().([]string))
}

func TestPrepareMethodArgumentsForCallWithArrayOfVariableLengthPassingNoArguments(t *testing.T) {
	methodInstance := func(a ...string) []string {
		return a
	}

	args := builders.ArgumentsArray()

	inValues, err := prepareMethodInputArgsForCall(methodInstance, args, "funcName")
	require.NoError(t, err)

	outValues := reflect.ValueOf(methodInstance).Call(inValues)
	require.EqualValues(t, []string{}, outValues[0].Interface().([]string))
}

func TestPrepareMethodArgumentsForCallWithArrayOfVariableLengthPassingArgumentsOfDifferentType(t *testing.T) {
	methodInstance := func(a uint32, b ...string) []string {
		return b
	}

	args := builders.ArgumentsArray(uint32(1), "hello", uint32(2))

	_, err := prepareMethodInputArgsForCall(methodInstance, args, "funcName")
	require.EqualError(t, err, "method 'funcName' expects arg 2 to be string but it has (Uint32Value)2")
}

func TestPrepareMethodArgumentsForCallWithArrayOfVariableLengthSkippingByteArrayArgument(t *testing.T) {
	methodInstance := func(a uint32, b []byte) []byte {
		return b
	}

	args := builders.ArgumentsArray(uint32(1))

	_, err := prepareMethodInputArgsForCall(methodInstance, args, "funcName")
	require.EqualError(t, err, "method 'funcName' takes 2 args but received less")
}

func TestPrepareMethodArgumentsForCallWithArrayOfByteArrays(t *testing.T) {
	methodInstance := func(a ...[]byte) [][]byte {
		return a
	}

	args := builders.ArgumentsArray([]byte("one"), []byte("two"))

	inValues, err := prepareMethodInputArgsForCall(methodInstance, args, "funcName")
	require.NoError(t, err)

	outValues := reflect.ValueOf(methodInstance).Call(inValues)
	require.EqualValues(t, [][]byte{[]byte("one"), []byte("two")}, outValues[0].Interface())
}

func TestPrepareMethodArgumentsForCallWithArrayOfArraysOfStringsPassingTwoByteArrays(t *testing.T) {
	methodInstance := func(a ...[]string) [][]string {
		return a
	}

	args := builders.ArgumentsArray([]byte("one"), []byte("two"))

	_, err := prepareMethodInputArgsForCall(methodInstance, args, "funcName")
	require.EqualError(t, err, "method 'funcName' expects arg 0 to be [][]byte but it has (BytesValue)6f6e65")
}

func TestPrepareMethodArgumentsForCallWithTwoByteArrays(t *testing.T) {
	methodInstance := func(a []byte, b []byte) [][]byte {
		return [][]byte{a, b}
	}

	args := builders.ArgumentsArray([]byte("one"), []byte("two"))

	inValues, err := prepareMethodInputArgsForCall(methodInstance, args, "funcName")
	require.NoError(t, err)

	outValues := reflect.ValueOf(methodInstance).Call(inValues)
	require.EqualValues(t, [][]byte{[]byte("one"), []byte("two")}, outValues[0].Interface())
}

func TestPrepareMethodArgumentsForCallWithIncorrectNumberOfArgs(t *testing.T) {
	methodInstance := func(a uint32) uint32 {
		return a
	}

	args := builders.ArgumentsArray(uint32(1997), uint32(1994))

	inValues, err := prepareMethodInputArgsForCall(methodInstance, args, "funcName")
	require.Errorf(t, err, "method 'funcName' takes 1 args but received more")
	require.Nil(t, inValues)

	inValues, err = prepareMethodInputArgsForCall(methodInstance, builders.ArgumentsArray(), "funcName")
	require.Errorf(t, err, "method 'funcName' takes 1 args but received less")
	require.Nil(t, inValues)
}
