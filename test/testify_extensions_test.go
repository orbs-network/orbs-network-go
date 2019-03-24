// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestContainsNil(t *testing.T) {
	tests := []struct {
		name            string
		objPtrGenerator func() interface{}
		expectedResult  bool
	}{
		{
			"*int: nil",
			func() interface{} {
				var obj *int = nil
				return &obj
			},
			true,
		},
		{
			"int: 12",
			func() interface{} {
				obj := 12
				return &obj
			},
			false,
		},
		{
			"slice: empty",
			func() interface{} {
				obj := []byte{}
				return &obj
			},
			false,
		},
		{
			"struct: noNil",
			func() interface{} {
				obj := struct{ a int }{a: 1}
				return &obj
			},
			false,
		},
		{
			"struct: unexportedNil",
			func() interface{} {
				obj := struct {
					a int
					b *int
				}{a: 1, b: nil}
				return &obj
			},
			false, // b is not exported
		},
		{
			"struct: containsNil",
			func() interface{} {
				obj := struct {
					A int
					B *int
				}{A: 1, B: nil}
				return &obj
			},
			true, // b is exported
		},
		{
			"nestedStruct: containsNil",
			func() interface{} {
				type nestedStruct struct {
					A int
					B struct {
						C int
						D *int
					}
				}

				obj := &nestedStruct{
					A: 1,
					B: struct {
						C int
						D *int
					}{C: 1, D: nil},
				}
				return &obj
			},
			true, // b is exported
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expectedResult, RequireDoesNotContainNil(t, tt.objPtrGenerator()))
		})
	}
}
