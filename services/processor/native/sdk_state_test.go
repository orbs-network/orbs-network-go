package native

import (
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

const EXAMPLE_CONTEXT = 0

func exampleKey() string {
	return "example-key"
}

func exampleKeyAddress() primitives.Ripmd160Sha256 {
	return hash.CalcRipmd160Sha256([]byte(exampleKey()))
}

func TestWriteReadBytesByAddress(t *testing.T) {
	s := createStateSdk()
	err := s.WriteBytesByAddress(EXAMPLE_CONTEXT, exampleKeyAddress(), []byte{0x01, 0x02, 0x03})
	assert.NoError(t, err, "write should succeed")

	bytes, err := s.ReadBytesByAddress(EXAMPLE_CONTEXT, exampleKeyAddress())
	assert.NoError(t, err, "read should succeed")
	assert.Equal(t, []byte{0x01, 0x02, 0x03}, bytes, "read should return what was written")
}

func TestWriteReadBytesByKey(t *testing.T) {
	s := createStateSdk()
	err := s.WriteBytesByKey(EXAMPLE_CONTEXT, exampleKey(), []byte{0x01, 0x02, 0x03})
	assert.NoError(t, err, "write should succeed")

	bytes, err := s.ReadBytesByKey(EXAMPLE_CONTEXT, exampleKey())
	assert.NoError(t, err, "read should succeed")
	assert.Equal(t, []byte{0x01, 0x02, 0x03}, bytes, "read should return what was written")
}

func TestClearReadBytesByAddress(t *testing.T) {
	s := createStateSdk()
	err := s.ClearByAddress(EXAMPLE_CONTEXT, exampleKeyAddress())
	assert.NoError(t, err, "clear should succeed")

	bytes, err := s.ReadBytesByAddress(EXAMPLE_CONTEXT, exampleKeyAddress())
	assert.NoError(t, err, "read should succeed")
	assert.Equal(t, []byte{}, bytes, "read should return what was written")
}

func TestClearReadBytesByKey(t *testing.T) {
	s := createStateSdk()
	err := s.ClearByKey(EXAMPLE_CONTEXT, exampleKey())
	assert.NoError(t, err, "clear should succeed")

	bytes, err := s.ReadBytesByKey(EXAMPLE_CONTEXT, exampleKey())
	assert.NoError(t, err, "read should succeed")
	assert.Equal(t, []byte{}, bytes, "read should return what was written")
}

func TestWriteReadStringByAddress(t *testing.T) {
	s := createStateSdk()
	err := s.WriteStringByAddress(EXAMPLE_CONTEXT, exampleKeyAddress(), "hello")
	assert.NoError(t, err, "write should succeed")

	str, err := s.ReadStringByAddress(EXAMPLE_CONTEXT, exampleKeyAddress())
	assert.NoError(t, err, "read should succeed")
	assert.Equal(t, "hello", str, "read should return what was written")
}

func TestWriteReadStringByKey(t *testing.T) {
	s := createStateSdk()
	err := s.WriteStringByKey(EXAMPLE_CONTEXT, exampleKey(), "hello")
	assert.NoError(t, err, "write should succeed")

	str, err := s.ReadStringByKey(EXAMPLE_CONTEXT, exampleKey())
	assert.NoError(t, err, "read should succeed")
	assert.Equal(t, "hello", str, "read should return what was written")
}

func TestClearReadStringByAddress(t *testing.T) {
	s := createStateSdk()
	err := s.ClearByAddress(EXAMPLE_CONTEXT, exampleKeyAddress())
	assert.NoError(t, err, "clear should succeed")

	str, err := s.ReadStringByAddress(EXAMPLE_CONTEXT, exampleKeyAddress())
	assert.NoError(t, err, "read should succeed")
	assert.Equal(t, "", str, "read should return what was written")
}

func TestClearReadStringByKey(t *testing.T) {
	s := createStateSdk()
	err := s.ClearByKey(EXAMPLE_CONTEXT, exampleKey())
	assert.NoError(t, err, "clear should succeed")

	str, err := s.ReadStringByKey(EXAMPLE_CONTEXT, exampleKey())
	assert.NoError(t, err, "read should succeed")
	assert.Equal(t, "", str, "read should return what was written")
}

func TestWriteReadUint64ByAddress(t *testing.T) {
	s := createStateSdk()
	err := s.WriteUint64ByAddress(EXAMPLE_CONTEXT, exampleKeyAddress(), uint64(17))
	assert.NoError(t, err, "write should succeed")

	num, err := s.ReadUint64ByAddress(EXAMPLE_CONTEXT, exampleKeyAddress())
	assert.NoError(t, err, "read should succeed")
	assert.Equal(t, uint64(17), num, "read should return what was written")
}

func TestWriteReadUint64ByKey(t *testing.T) {
	s := createStateSdk()
	err := s.WriteUint64ByKey(EXAMPLE_CONTEXT, exampleKey(), uint64(17))
	assert.NoError(t, err, "write should succeed")

	num, err := s.ReadUint64ByKey(EXAMPLE_CONTEXT, exampleKey())
	assert.NoError(t, err, "read should succeed")
	assert.Equal(t, uint64(17), num, "read should return what was written")
}

func TestClearReadUint64ByAddress(t *testing.T) {
	s := createStateSdk()
	err := s.ClearByAddress(EXAMPLE_CONTEXT, exampleKeyAddress())
	assert.NoError(t, err, "clear should succeed")

	num, err := s.ReadUint64ByAddress(EXAMPLE_CONTEXT, exampleKeyAddress())
	assert.NoError(t, err, "read should succeed")
	assert.Equal(t, uint64(0), num, "read should return what was written")
}

func TestClearReadUint64ByKey(t *testing.T) {
	s := createStateSdk()
	err := s.ClearByKey(EXAMPLE_CONTEXT, exampleKey())
	assert.NoError(t, err, "clear should succeed")

	num, err := s.ReadUint64ByKey(EXAMPLE_CONTEXT, exampleKey())
	assert.NoError(t, err, "read should succeed")
	assert.Equal(t, uint64(0), num, "read should return what was written")
}

func TestWriteReadUint32ByAddress(t *testing.T) {
	s := createStateSdk()
	err := s.WriteUint32ByAddress(EXAMPLE_CONTEXT, exampleKeyAddress(), uint32(15))
	assert.NoError(t, err, "write should succeed")

	num, err := s.ReadUint32ByAddress(EXAMPLE_CONTEXT, exampleKeyAddress())
	assert.NoError(t, err, "read should succeed")
	assert.Equal(t, uint32(15), num, "read should return what was written")
}

func TestWriteReadUint32ByKey(t *testing.T) {
	s := createStateSdk()
	err := s.WriteUint32ByKey(EXAMPLE_CONTEXT, exampleKey(), uint32(15))
	assert.NoError(t, err, "write should succeed")

	num, err := s.ReadUint32ByKey(EXAMPLE_CONTEXT, exampleKey())
	assert.NoError(t, err, "read should succeed")
	assert.Equal(t, uint32(15), num, "read should return what was written")
}

func TestClearReadUint32ByAddress(t *testing.T) {
	s := createStateSdk()
	err := s.ClearByAddress(EXAMPLE_CONTEXT, exampleKeyAddress())
	assert.NoError(t, err, "clear should succeed")

	num, err := s.ReadUint32ByAddress(EXAMPLE_CONTEXT, exampleKeyAddress())
	assert.NoError(t, err, "read should succeed")
	assert.Equal(t, uint32(0), num, "read should return what was written")
}

func TestClearReadUint32ByKey(t *testing.T) {
	s := createStateSdk()
	err := s.ClearByKey(EXAMPLE_CONTEXT, exampleKey())
	assert.NoError(t, err, "clear should succeed")

	num, err := s.ReadUint32ByKey(EXAMPLE_CONTEXT, exampleKey())
	assert.NoError(t, err, "read should succeed")
	assert.Equal(t, uint32(0), num, "read should return what was written")
}

func createStateSdk() *stateSdk {
	return &stateSdk{
		handler: &contractSdkCallHandlerStub{make(map[string]*protocol.MethodArgument, 0)},
	}
}

type contractSdkCallHandlerStub struct {
	store map[string]*protocol.MethodArgument
}

func (c *contractSdkCallHandlerStub) HandleSdkCall(input *handlers.HandleSdkCallInput) (*handlers.HandleSdkCallOutput, error) {
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
