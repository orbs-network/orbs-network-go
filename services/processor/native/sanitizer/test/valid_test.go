package test

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native/sanitizer"
	"github.com/orbs-network/orbs-network-go/services/processor/native/sanitizer/test/usecases"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestValidCode(t *testing.T) {
	source := usecases.Counter
	output, err := sanitizer.NewSanitizer(SanitizerConfigForTests()).Process(source)
	require.NoError(t, err)
	require.Equal(t, source, output, "valid file content should be altered by sanitizer")
}
