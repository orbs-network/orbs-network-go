package log

import (
	"encoding/json"
	"github.com/orbs-network/orbs-network-go/crypto/base58"
	"strconv"
	"strings"
	"time"
)

type LogFormatter interface {
	FormatRow(level string, message string, params ...*Field) (formattedRow string)
}

type jsonFormatter struct {
}

const TIMESTAMP_FORMAT = "2006-01-02T15:04:05.999999999Z"

func (j *jsonFormatter) FormatRow(level string, message string, params ...*Field) (formattedRow string) {
	logLine := make(map[string]interface{})

	logLine["level"] = level
	logLine["timestamp"] = time.Now().UTC().Format(TIMESTAMP_FORMAT)
	logLine["message"] = message

	logFields(params, logLine)

	logLineAsJson, _ := json.Marshal(logLine)

	return string(logLineAsJson)
}

func logFields(params []*Field, logLine map[string]interface{}) {
	for _, param := range params {
		if !param.IsNested() {
			logLine[param.Key] = param.Value()
		} else if nestedFields, ok := param.Value().([]*Field); ok {
			logFields(nestedFields, logLine)
		} else {
			panic("log field of nested type did not return []*Field")
		}
	}
}

func NewJsonFormatter() LogFormatter {
	return &jsonFormatter{}
}

type humanReadableFormatter struct {
}

const (
	SPACE  = " "
	EQUALS = "="
)

func findFieldByType(fieldType FieldType, params []*Field) (index int, result *Field) {
	for idx, param := range params {
		if param.Type == fieldType {
			return idx, param
		}
	}

	return -1, nil
}

func printParam(builder *strings.Builder, param *Field) {
	if param == nil {
		return
	}

	var value string

	switch param.Type {
	case StringType:
		value = param.String
	case NodeType:
		value = param.String
	case ServiceType:
		value = param.String
	case FunctionType:
		value = param.String
	case SourceType:
		value = param.String
	case IntType:
		value = strconv.FormatInt(param.Int, 10)
	case UintType:
		value = strconv.FormatUint(param.Uint, 10)
	case BytesType:
		value = string(base58.Encode(param.Bytes))
	case FloatType:
		value = strconv.FormatFloat(param.Float, 'f', -1, 64)
	case ErrorType:
		if param.Error != nil {
			value = param.Error.Error()
		} else {
			value = "<nil>"
		}
	case StringArrayType:
		json, err := json.MarshalIndent(param.StringArray, "", "\t")
		if err != nil {
			value = ""
		} else {
			value = string(json)
		}
	}

	builder.WriteString(param.Key)
	builder.WriteString(EQUALS)
	builder.WriteString(value)
	builder.WriteString(SPACE)
}

func cut(i int, params []*Field) []*Field {
	copy(params[i:], params[i+1:])
	params[len(params)-1] = nil
	params = params[:len(params)-1]
	return params
}

func extractParamByTypePrintAndRemove(params []*Field, ft FieldType, builder *strings.Builder) (*Field, []*Field) {
	return extractParamByType(params, ft, true, true, builder)
}

func extractParamByTypeAndRemove(params []*Field, ft FieldType) (*Field, []*Field) {
	return extractParamByType(params, ft, false, true, nil)
}

func extractParamByType(params []*Field, ft FieldType, shouldPrint, shouldRemove bool, builder *strings.Builder) (*Field, []*Field) {
	if idx, param := findFieldByType(ft, params); param != nil {
		if shouldPrint {
			printParam(builder, param)
		}
		if shouldRemove {
			params = cut(idx, params)
		}

		return param, params
	}

	return nil, params
}

func extractParamByConditionAndRemove(params []*Field, condition func(param *Field) bool) (results []*Field, newParams []*Field) {
	for _, param := range params {
		if condition(param) {
			results = append(results, param)
		} else {
			newParams = append(newParams, param)
		}
	}

	return results, newParams
}

func (j *humanReadableFormatter) FormatRow(level string, message string, params ...*Field) (formattedRow string) {
	builder := strings.Builder{}

	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05.000000Z07:00")

	builder.WriteString(level)
	builder.WriteString(SPACE)
	builder.WriteString(timestamp)
	builder.WriteString(SPACE)

	builder.WriteString(message)
	builder.WriteString(SPACE)

	var newParams = make([]*Field, len(params))
	copy(newParams, params)

	_, newParams = extractParamByTypePrintAndRemove(newParams, NodeType, &builder)
	_, newParams = extractParamByTypePrintAndRemove(newParams, ServiceType, &builder)
	functionParam, newParams := extractParamByTypeAndRemove(newParams, FunctionType)
	sourceParam, newParams := extractParamByTypeAndRemove(newParams, SourceType)
	underscoreParams, newParams := extractParamByConditionAndRemove(newParams, func(param *Field) bool {
		return strings.Index(param.Key, "_") == 0
	})

	printParams(&builder, newParams)

	// append the function/source
	printParam(&builder, functionParam)
	printParam(&builder, sourceParam)

	for _, param := range underscoreParams {
		printParam(&builder, param)
	}
	return builder.String()
}

func printParams(builder *strings.Builder, params []*Field) {
	for _, param := range params {
		if !param.IsNested() {
			printParam(builder, param)
		} else if nestedFields, ok := param.Value().([]*Field); ok {
			printParams(builder, nestedFields)
		} else {
			panic("log field of nested type did not return []*Field")
		}
	}
}

func NewHumanReadableFormatter() LogFormatter {
	return &humanReadableFormatter{}
}
