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

	String      string
	StringArray []string
	Int         int64
	Uint        uint64
	Bytes       []byte
	Float       float64

	Error error
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

func (this *Field) Equal(other *Field) bool {
	return this.Type == other.Type && this.Value() == other.Value() && this.Key == other.Key
}

type FieldType uint8

func Node(value string) *Field {
	return &Field{Key: "node", String: value, Type: NodeType}
}

func Service(value string) *Field {
	return &Field{Key: "service", String: value, Type: ServiceType}
}

func Function(value string) *Field {
	return &Field{Key: "function", String: value, Type: FunctionType}
}

func Source(value string) *Field {
	return &Field{Key: "source", String: value, Type: SourceType}
}

func String(key string, value string) *Field {
	return &Field{Key: key, String: value, Type: StringType}
}

func Stringable(key string, value fmt.Stringer) *Field {
	return &Field{Key: key, String: value.String(), Type: StringType}
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
	return &Field{Key: "error", Error: value, Type: ErrorType}
}

func BlockHeight(value primitives.BlockHeight) *Field {
	return &Field{Key: "block-height", String: value.String(), Type: StringType}
}

func VirtualChainId(value primitives.VirtualChainId) *Field {
	return &Field{Key: "vcid", String: value.String(), Type: StringType}
}

func (f *Field) Value() interface{} {
	if f == nil {
		return "<nil>"
	}
	switch f.Type {
	case NodeType:
		return f.String
	case ServiceType:
		return f.String
	case FunctionType:
		return f.String
	case SourceType:
		return f.String
	case StringType:
		return f.String
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
		return f.Error.Error()
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
