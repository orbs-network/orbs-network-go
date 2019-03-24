// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

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
