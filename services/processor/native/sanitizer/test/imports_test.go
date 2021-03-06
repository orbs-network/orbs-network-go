// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native"
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

func TestCodeWithNamedImport(t *testing.T) {
	source := usecases.NamedImport
	output, err := sanitizer.NewSanitizer(SanitizerConfigForTests()).Process(source)
	require.Error(t, err)
	require.Equal(t, `native code verification error: custom import names not allowed`, err.Error())
	require.Empty(t, output)
}

func TestCodeWithJsonAndHex(t *testing.T) {
	source := usecases.Serialization
	output, err := sanitizer.NewSanitizer(native.SanitizerConfigForProduction()).Process(source)
	require.NoError(t, err)
	require.Equal(t, source, output, "valid file content should not be altered by sanitizer")
}

func TestWildcardImport(t *testing.T) {
	source := usecases.WildcardImport
	output, err := sanitizer.NewSanitizer(SanitizerConfigWithWildcardForTests()).Process(source)
	require.NoError(t, err)
	require.Equal(t, source, output, "valid file content with wildcards should not be rejected by sanitizer")
}
