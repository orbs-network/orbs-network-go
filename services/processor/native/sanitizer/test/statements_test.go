package test

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native/sanitizer"
	"github.com/orbs-network/orbs-network-go/services/processor/native/sanitizer/test/usecases"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCodeWithGoroutineStatement(t *testing.T) {
	source := usecases.UseGoroutine
	output, err := sanitizer.NewSanitizer(SanitizerConfigForTests()).Process(source)
	require.Error(t, err)
	require.Equal(t, `native code verification error: goroutines not allowed`, err.Error())
	require.Empty(t, output)
}

func TestCodeWithSendStatement(t *testing.T) {
	source := usecases.SendToChannel
	output, err := sanitizer.NewSanitizer(SanitizerConfigForTests()).Process(source)
	require.Error(t, err)
	require.Equal(t, `native code verification error: sending to channels not allowed`, err.Error())
	require.Empty(t, output)
}

func TestCodeWithChannelDeclaration(t *testing.T) {
	source := usecases.CreateChannel
	output, err := sanitizer.NewSanitizer(SanitizerConfigForTests()).Process(source)
	require.Error(t, err)
	require.Equal(t, `native code verification error: channels not allowed`, err.Error())
	require.Empty(t, output)
}

func TestCodeWithTimeSleep(t *testing.T) {
	source := usecases.Sleep
	output, err := sanitizer.NewSanitizer(SanitizerConfigForTests()).Process(source)
	require.Error(t, err)
	require.Equal(t, `native code verification error: time.Sleep not allowed`, err.Error())
	require.Empty(t, output)
}
