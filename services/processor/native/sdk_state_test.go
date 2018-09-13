package native

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

const EXAMPLE_CONTEXT = 0

func exampleKey() string {
	return "example-key"
}

func exampleKeyAddress() sdk.Ripmd160Sha256 {
	return sdk.Ripmd160Sha256(hash.CalcRipmd160Sha256([]byte(exampleKey())))
}

func TestWriteReadBytesByAddress(t *testing.T) {
	s := createStateSdk()
	err := s.WriteBytesByAddress(EXAMPLE_CONTEXT, exampleKeyAddress(), []byte{0x01, 0x02, 0x03})
	require.NoError(t, err, "write should succeed")

	bytes, err := s.ReadBytesByAddress(EXAMPLE_CONTEXT, exampleKeyAddress())
	require.NoError(t, err, "read should succeed")
	require.Equal(t, []byte{0x01, 0x02, 0x03}, bytes, "read should return what was written")
}

func TestWriteReadBytesByKey(t *testing.T) {
	s := createStateSdk()
	err := s.WriteBytesByKey(EXAMPLE_CONTEXT, exampleKey(), []byte{0x01, 0x02, 0x03})
	require.NoError(t, err, "write should succeed")

	bytes, err := s.ReadBytesByKey(EXAMPLE_CONTEXT, exampleKey())
	require.NoError(t, err, "read should succeed")
	require.Equal(t, []byte{0x01, 0x02, 0x03}, bytes, "read should return what was written")
}

func TestClearReadBytesByAddress(t *testing.T) {
	s := createStateSdk()
	err := s.ClearByAddress(EXAMPLE_CONTEXT, exampleKeyAddress())
	require.NoError(t, err, "clear should succeed")

	bytes, err := s.ReadBytesByAddress(EXAMPLE_CONTEXT, exampleKeyAddress())
	require.NoError(t, err, "read should succeed")
	require.Equal(t, []byte{}, bytes, "read should return what was written")
}

func TestClearReadBytesByKey(t *testing.T) {
	s := createStateSdk()
	err := s.ClearByKey(EXAMPLE_CONTEXT, exampleKey())
	require.NoError(t, err, "clear should succeed")

	bytes, err := s.ReadBytesByKey(EXAMPLE_CONTEXT, exampleKey())
	require.NoError(t, err, "read should succeed")
	require.Equal(t, []byte{}, bytes, "read should return what was written")
}

func TestWriteReadStringByAddress(t *testing.T) {
	s := createStateSdk()
	err := s.WriteStringByAddress(EXAMPLE_CONTEXT, exampleKeyAddress(), "hello")
	require.NoError(t, err, "write should succeed")

	str, err := s.ReadStringByAddress(EXAMPLE_CONTEXT, exampleKeyAddress())
	require.NoError(t, err, "read should succeed")
	require.Equal(t, "hello", str, "read should return what was written")
}

func TestWriteReadStringByKey(t *testing.T) {
	s := createStateSdk()
	err := s.WriteStringByKey(EXAMPLE_CONTEXT, exampleKey(), "hello")
	require.NoError(t, err, "write should succeed")

	str, err := s.ReadStringByKey(EXAMPLE_CONTEXT, exampleKey())
	require.NoError(t, err, "read should succeed")
	require.Equal(t, "hello", str, "read should return what was written")
}

func TestClearReadStringByAddress(t *testing.T) {
	s := createStateSdk()
	err := s.ClearByAddress(EXAMPLE_CONTEXT, exampleKeyAddress())
	require.NoError(t, err, "clear should succeed")

	str, err := s.ReadStringByAddress(EXAMPLE_CONTEXT, exampleKeyAddress())
	require.NoError(t, err, "read should succeed")
	require.Equal(t, "", str, "read should return what was written")
}

func TestClearReadStringByKey(t *testing.T) {
	s := createStateSdk()
	err := s.ClearByKey(EXAMPLE_CONTEXT, exampleKey())
	require.NoError(t, err, "clear should succeed")

	str, err := s.ReadStringByKey(EXAMPLE_CONTEXT, exampleKey())
	require.NoError(t, err, "read should succeed")
	require.Equal(t, "", str, "read should return what was written")
}

func TestWriteReadUint64ByAddress(t *testing.T) {
	s := createStateSdk()
	err := s.WriteUint64ByAddress(EXAMPLE_CONTEXT, exampleKeyAddress(), uint64(17))
	require.NoError(t, err, "write should succeed")

	num, err := s.ReadUint64ByAddress(EXAMPLE_CONTEXT, exampleKeyAddress())
	require.NoError(t, err, "read should succeed")
	require.Equal(t, uint64(17), num, "read should return what was written")
}

func TestWriteReadUint64ByKey(t *testing.T) {
	s := createStateSdk()
	err := s.WriteUint64ByKey(EXAMPLE_CONTEXT, exampleKey(), uint64(17))
	require.NoError(t, err, "write should succeed")

	num, err := s.ReadUint64ByKey(EXAMPLE_CONTEXT, exampleKey())
	require.NoError(t, err, "read should succeed")
	require.Equal(t, uint64(17), num, "read should return what was written")
}

func TestClearReadUint64ByAddress(t *testing.T) {
	s := createStateSdk()
	err := s.ClearByAddress(EXAMPLE_CONTEXT, exampleKeyAddress())
	require.NoError(t, err, "clear should succeed")

	num, err := s.ReadUint64ByAddress(EXAMPLE_CONTEXT, exampleKeyAddress())
	require.NoError(t, err, "read should succeed")
	require.Equal(t, uint64(0), num, "read should return what was written")
}

func TestClearReadUint64ByKey(t *testing.T) {
	s := createStateSdk()
	err := s.ClearByKey(EXAMPLE_CONTEXT, exampleKey())
	require.NoError(t, err, "clear should succeed")

	num, err := s.ReadUint64ByKey(EXAMPLE_CONTEXT, exampleKey())
	require.NoError(t, err, "read should succeed")
	require.Equal(t, uint64(0), num, "read should return what was written")
}

func TestWriteReadUint32ByAddress(t *testing.T) {
	s := createStateSdk()
	err := s.WriteUint32ByAddress(EXAMPLE_CONTEXT, exampleKeyAddress(), uint32(15))
	require.NoError(t, err, "write should succeed")

	num, err := s.ReadUint32ByAddress(EXAMPLE_CONTEXT, exampleKeyAddress())
	require.NoError(t, err, "read should succeed")
	require.Equal(t, uint32(15), num, "read should return what was written")
}

func TestWriteReadUint32ByKey(t *testing.T) {
	s := createStateSdk()
	err := s.WriteUint32ByKey(EXAMPLE_CONTEXT, exampleKey(), uint32(15))
	require.NoError(t, err, "write should succeed")

	num, err := s.ReadUint32ByKey(EXAMPLE_CONTEXT, exampleKey())
	require.NoError(t, err, "read should succeed")
	require.Equal(t, uint32(15), num, "read should return what was written")
}

func TestClearReadUint32ByAddress(t *testing.T) {
	s := createStateSdk()
	err := s.ClearByAddress(EXAMPLE_CONTEXT, exampleKeyAddress())
	require.NoError(t, err, "clear should succeed")

	num, err := s.ReadUint32ByAddress(EXAMPLE_CONTEXT, exampleKeyAddress())
	require.NoError(t, err, "read should succeed")
	require.Equal(t, uint32(0), num, "read should return what was written")
}

func TestClearReadUint32ByKey(t *testing.T) {
	s := createStateSdk()
	err := s.ClearByKey(EXAMPLE_CONTEXT, exampleKey())
	require.NoError(t, err, "clear should succeed")

	num, err := s.ReadUint32ByKey(EXAMPLE_CONTEXT, exampleKey())
	require.NoError(t, err, "read should succeed")
	require.Equal(t, uint32(0), num, "read should return what was written")
}

func createStateSdk() *stateSdk {
	return &stateSdk{
		handler:         &contractSdkStateCallHandlerStub{make(map[string]*protocol.MethodArgument, 0)},
		permissionScope: protocol.PERMISSION_SCOPE_SERVICE,
	}
}

type contractSdkStateCallHandlerStub struct {
	store map[string]*protocol.MethodArgument
}

func (c *contractSdkStateCallHandlerStub) HandleSdkCall(input *handlers.HandleSdkCallInput) (*handlers.HandleSdkCallOutput, error) {
	if input.PermissionScope != protocol.PERMISSION_SCOPE_SERVICE {
		panic("permissions passed to SDK are incorrect")
	}
	switch input.MethodName {
	case "read":
		return &handlers.HandleSdkCallOutput{
			OutputArguments: []*protocol.MethodArgument{c.store[string(input.InputArguments[0].BytesValue())]},
		}, nil
	case "write":
		c.store[string(input.InputArguments[0].BytesValue())] = input.InputArguments[1]
		return nil, nil
	default:
		return nil, errors.New("unknown method")
	}
}
