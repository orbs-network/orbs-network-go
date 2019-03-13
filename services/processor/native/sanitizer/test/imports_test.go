package test

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native/sanitizer"
	"github.com/orbs-network/orbs-network-go/services/processor/native/sanitizer/test/usecases"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCodeWithInvalidImport(t *testing.T) {
	source := usecases.AccessFilesystem
	output, err := sanitizer.NewSanitizer(SanitizerConfigForTests()).Process(source)
	require.Error(t, err)
	require.Equal(t, `native code verification error: import not allowed '"io/ioutil"'`, err.Error())
	require.Empty(t, output)
}
