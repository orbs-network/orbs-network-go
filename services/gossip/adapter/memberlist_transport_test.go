package adapter

import (
	"testing"
	"reflect"
)

func TestPayloadPackUnpack(t *testing.T) {
	payload := [][]byte{{1, 2, 3}, {4, 5, 6}}
	b := encodeByteArray(payload)
	result := decodeByteArray(b)

	if !reflect.DeepEqual(result, payload) {
		t.Fatalf("result %v does not match payload %v", result, payload)
	}
}