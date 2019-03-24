// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package log

import (
	"encoding/hex"
	"fmt"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"reflect"
	"time"
)

type AggregateField interface {
	NestedFields() []*Field
}

type Field struct {
	Key  string
	Type FieldType

	StringVal   string
	StringArray []string
	Int         int64
	Uint        uint64
	Bytes       []byte
	Float       float64

	Error  error
	Nested AggregateField
}

const (
	NoType = iota
	ErrorType
	NodeType
	ServiceType
	StringType
	IntType
	UintType
	BytesType
	FloatType
	FunctionType
	SourceType
	StringArrayType
	TimeType
	AggregateType
)

func (f *Field) Equal(other *Field) bool {
	return f.Type == other.Type && f.Value() == other.Value() && f.Key == other.Key
}

type FieldType uint8

func Node(value string) *Field {
	return &Field{Key: "node", StringVal: value, Type: NodeType}
}

func Service(value string) *Field {
	return &Field{Key: "service", StringVal: value, Type: ServiceType}
}

func Function(value string) *Field {
	return &Field{Key: "function", StringVal: value, Type: FunctionType}
}

func Source(value string) *Field {
	return &Field{Key: "source", StringVal: value, Type: SourceType}
}

func String(key string, value string) *Field {
	return &Field{Key: key, StringVal: value, Type: StringType}
}

func Stringable(key string, value fmt.Stringer) *Field {
	return &Field{Key: key, StringVal: value.String(), Type: StringType}
}

func Transaction(txHash primitives.Sha256) *Field {
	return Stringable("txHash", txHash)
}

func Query(queryHash primitives.Sha256) *Field {
	return Stringable("queryHash", queryHash)
}

func StringableSlice(key string, values interface{}) *Field {
	var strings []string
	switch reflect.TypeOf(values).Kind() {
	case reflect.Slice:
		s := reflect.ValueOf(values)

		strings = make([]string, 0, s.Len())

		for i := 0; i < s.Len(); i++ {
			if stringer, ok := s.Index(i).Interface().(fmt.Stringer); ok {
				strings = append(strings, stringer.String())
			}
		}
	}

	return &Field{Key: key, StringArray: strings, Type: StringArrayType}
}

func Int(key string, value int) *Field {
	return &Field{Key: key, Int: int64(value), Type: IntType}
}

func Int32(key string, value int32) *Field {
	return &Field{Key: key, Int: int64(value), Type: IntType}
}

func Int64(key string, value int64) *Field {
	return &Field{Key: key, Int: int64(value), Type: IntType}
}

func Bytes(key string, value []byte) *Field {
	return &Field{Key: key, Bytes: value, Type: BytesType}
}

func Uint(key string, value uint) *Field {
	return &Field{Key: key, Uint: uint64(value), Type: UintType}
}

func Uint32(key string, value uint32) *Field {
	return &Field{Key: key, Uint: uint64(value), Type: UintType}
}

func Uint64(key string, value uint64) *Field {
	return &Field{Key: key, Uint: value, Type: UintType}
}

func Float32(key string, value float32) *Field {
	return &Field{Key: key, Float: float64(value), Type: FloatType}
}

func Float64(key string, value float64) *Field {
	return &Field{Key: key, Float: value, Type: FloatType}
}

func TimestampNano(key string, value primitives.TimestampNano) *Field {
	return &Field{Key: key, Int: int64(value), Type: TimeType}
}

func Timestamp(key string, value time.Time) *Field {
	return &Field{Key: key, Int: value.UnixNano(), Type: TimeType}
}

func Error(value error) *Field {
	if value == nil {
		panic("error field must have non-nil error value")
	}
	return &Field{Key: "error", Error: value, Type: ErrorType}
}

func BlockHeight(value primitives.BlockHeight) *Field {
	return &Field{Key: "block-height", Uint: uint64(value), Type: UintType}
}

func VirtualChainId(value primitives.VirtualChainId) *Field {
	return &Field{Key: "vcid", Uint: uint64(value), Type: UintType}
}

func (f *Field) Value() interface{} {
	if f == nil {
		return "<nil>"
	}
	switch f.Type {
	case NodeType:
		return f.StringVal
	case ServiceType:
		return f.StringVal
	case FunctionType:
		return f.StringVal
	case SourceType:
		return f.StringVal
	case StringType:
		return f.StringVal
	case IntType:
		return f.Int
	case TimeType:
		return time.Unix(0, f.Int)
	case UintType:
		return f.Uint
	case BytesType:
		return hex.EncodeToString(f.Bytes)
	case FloatType:
		return f.Float
	case ErrorType:
		if f.Error != nil {
			return f.Error.Error()
		} else {
			return "<nil>"
		}
	case StringArrayType:
		return f.StringArray
	case AggregateType:
		return f.Nested.NestedFields()
	}

	return nil
}

func (f *Field) IsNested() bool {
	return f.Type == AggregateType
}

func (f *Field) String() string {
	return fmt.Sprintf("Field: key=%s, value=%v", f.Key, f.Value())
}
