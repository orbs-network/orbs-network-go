package codec

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestContainsNil(t *testing.T) {
	tests := []struct {
		name           string
		obj            interface{}
		expectedResult bool
	}{
		{
			"nil",
			nil,
			true,
		},
		{
			"12",
			12,
			false,
		},
		{
			"[]byte{}",
			[]byte{},
			false,
		},
		{
			"struct{}{}",
			struct{}{},
			false,
		},
		{
			"struct{int}{a:1}",
			struct{ a int }{a: 1},
			false,
		},
		{
			"struct{int,*int}{a:1,b:nil}",
			struct {
				a int
				b *int
			}{a: 1, b: nil},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expectedResult, containsNil(tt.obj))
		})
	}
}
