package native

import (
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

func TestPrepareMethodArgumentsForCallWithUint32(t *testing.T) {
	s := &service{}

	methodInstance := func(a uint32) uint32 {
		return a
	}

	args := argsToArgumentArray(uint32(1997))

	inValues, err := s.prepareMethodInputArgsForCall(methodInstance, args, "funcName")
	require.NoError(t, err)

	outValues := reflect.ValueOf(methodInstance).Call(inValues)
	require.EqualValues(t, 1997, outValues[0].Uint())
}

func TestPrepareMethodArgumentsForCallWithByteArray(t *testing.T) {
	s := &service{}

	methodInstance := func(a []byte) []byte {
		return a
	}

	args := argsToArgumentArray([]byte("hello"))

	inValues, err := s.prepareMethodInputArgsForCall(methodInstance, args, "funcName")
	require.NoError(t, err)

	outValues := reflect.ValueOf(methodInstance).Call(inValues)
	require.EqualValues(t, []byte("hello"), outValues[0].Bytes())
}

func TestPrepareMethodArgumentsForCallWithArrayOfVariableLength(t *testing.T) {
	s := &service{}

	methodInstance := func(a ...string) []string {
		return a
	}

	args := argsToArgumentArray("one", "two")

	inValues, err := s.prepareMethodInputArgsForCall(methodInstance, args, "funcName")
	require.NoError(t, err)

	outValues := reflect.ValueOf(methodInstance).Call(inValues)
	require.EqualValues(t, []string{"one", "two"}, outValues[0].Interface().([]string))
}

func TestPrepareMethodArgumentsForCallWithArrayOfVariableLengthPassingNoArguments(t *testing.T) {
	s := &service{}

	methodInstance := func(a ...string) []string {
		return a
	}

	args := argsToArgumentArray()

	inValues, err := s.prepareMethodInputArgsForCall(methodInstance, args, "funcName")
	require.NoError(t, err)

	outValues := reflect.ValueOf(methodInstance).Call(inValues)
	require.EqualValues(t, []string{}, outValues[0].Interface().([]string))
}

func TestPrepareMethodArgumentsForCallWithArrayOfByteArrays(t *testing.T) {
	s := &service{}

	methodInstance := func(a ...[]byte) [][]byte {
		return a
	}

	args := argsToArgumentArray([]byte("one"), []byte("two"))

	inValues, err := s.prepareMethodInputArgsForCall(methodInstance, args, "funcName")
	require.NoError(t, err)

	outValues := reflect.ValueOf(methodInstance).Call(inValues)
	require.EqualValues(t, [][]byte{[]byte("one"), []byte("two")}, outValues[0].Interface())
}

func TestPrepareMethodArgumentsForCallWithTwoByteArrays(t *testing.T) {
	s := &service{}

	methodInstance := func(a []byte, b []byte) [][]byte {
		return [][]byte{a, b}
	}

	args := argsToArgumentArray([]byte("one"), []byte("two"))

	inValues, err := s.prepareMethodInputArgsForCall(methodInstance, args, "funcName")
	require.NoError(t, err)

	outValues := reflect.ValueOf(methodInstance).Call(inValues)
	require.EqualValues(t, [][]byte{[]byte("one"), []byte("two")}, outValues[0].Interface())
}

func TestPrepareMethodArgumentsForCallWithIncorrectNumberOfArgs(t *testing.T) {
	s := &service{}

	methodInstance := func(a uint32) uint32 {
		return a
	}

	args := argsToArgumentArray(uint32(1997), uint32(1994))

	inValues, err := s.prepareMethodInputArgsForCall(methodInstance, args, "funcName")
	require.Errorf(t, err, "method 'funcName' takes 1 args but received more")
	require.Nil(t, inValues)

	inValues, err = s.prepareMethodInputArgsForCall(methodInstance, argsToArgumentArray(), "funcName")
	require.Errorf(t, err, "method 'funcName' takes 1 args but received less")
	require.Nil(t, inValues)
}
