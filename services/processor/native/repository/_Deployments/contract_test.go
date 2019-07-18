package deployments_systemcontract

import (
	"fmt"
	. "github.com/orbs-network/orbs-contract-sdk/go/testing/unit"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetSingleFileCode(t *testing.T) {
	diffs, _, _ := InSystemScope(nil, nil, func(m Mockery) {
		m.MockServiceCallMethod("hello", "_init", nil)

		deployService("hello", 2, []byte("contract"))
		code := getCode("hello", 0)
		require.EqualValues(t, []byte("contract"), code)

		codeParts := getCodeParts("hello")
		require.Zero(t, codeParts)
	})

	for _, d := range diffs {
		fmt.Println(string(d.Key), "=", string(d.Value))
	}
}

func TestGetMultipleFilesCode(t *testing.T) {
	diffs, _, _ := InSystemScope(nil, nil, func(m Mockery) {
		m.MockServiceCallMethod("hello", "_init", nil)

		deployService("hello", 2, []byte("contract"), []byte("more contract stuff"))
		code := getCode("hello", 0)
		require.EqualValues(t, []byte("contract"), code)

		codeSecondPart := getCode("hello", 1)
		require.EqualValues(t, []byte("more contract stuff"), codeSecondPart)

		codeParts := getCodeParts("hello")
		require.EqualValues(t, 1, codeParts)
	})

	for _, d := range diffs {
		fmt.Println(string(d.Key), "=", string(d.Value))
	}
}
