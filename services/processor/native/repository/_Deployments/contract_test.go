package deployments_systemcontract

import (
	"fmt"
	. "github.com/orbs-network/orbs-contract-sdk/go/testing/unit"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetCodeForNonExistentContract(t *testing.T) {
	InSystemScope(nil, nil, func(m Mockery) {
		require.PanicsWithValue(t, "contract code not available", func() {
			getCode("hello")
		})

		require.PanicsWithValue(t, "contract code not available", func() {
			getCodePart("hello", 0)
		})

		require.PanicsWithValue(t, "contract not deployed", func() {
			getCodeParts("hello")
		})
	})
}

func TestGetSingleFileCodeViaOldInterface(t *testing.T) {
	diffs, _, _ := InSystemScope(nil, nil, func(m Mockery) {
		m.MockServiceCallMethod("hello", "_init", nil)

		deployService("hello", 2, []byte("contract"))
		code := getCode("hello")
		require.EqualValues(t, []byte("contract"), code)

		codeParts := getCodeParts("hello")
		require.EqualValues(t, 1, codeParts)
	})

	for _, d := range diffs {
		fmt.Println(string(d.Key), "=", string(d.Value))
	}
}

func TestGetSingleFileCodeViaNewInterface(t *testing.T) {
	diffs, _, _ := InSystemScope(nil, nil, func(m Mockery) {
		m.MockServiceCallMethod("hello", "_init", nil)

		deployService("hello", 2, []byte("contract"))
		parts := getCodeParts("hello")
		require.EqualValues(t, 1, parts)

		code := getCodePart("hello", 0)
		require.EqualValues(t, []byte("contract"), code)
	})

	for _, d := range diffs {
		fmt.Println(string(d.Key), "=", string(d.Value))
	}
}

func TestGetMultipleFilesCode(t *testing.T) {
	diffs, _, _ := InSystemScope(nil, nil, func(m Mockery) {
		m.MockServiceCallMethod("hello", "_init", nil)

		deployService("hello", 2, []byte("contract"), []byte("more contract stuff"))
		code := getCodePart("hello", 0)
		require.EqualValues(t, []byte("contract"), code)

		codeSecondPart := getCodePart("hello", 1)
		require.EqualValues(t, []byte("more contract stuff"), codeSecondPart)

		codeParts := getCodeParts("hello")
		println(codeParts)
		require.EqualValues(t, codeParts, 2)
	})

	for _, d := range diffs {
		fmt.Println(string(d.Key), "=", string(d.Value))
	}
}
