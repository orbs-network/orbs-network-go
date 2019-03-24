// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package log

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

type BasicLogger interface {
	Log(level string, message string, params ...*Field)
	LogFailedExpectation(message string, expected *Field, actual *Field, params ...*Field)
	Info(message string, params ...*Field)
	Error(message string, params ...*Field)
	Metric(params ...*Field)
	WithTags(params ...*Field) BasicLogger
	Tags() []*Field
	WithOutput(writer ...Output) BasicLogger
	WithFilters(filter ...Filter) BasicLogger
	Filters() []Filter
}

type basicLogger struct {
	outputs               []Output
	tags                  []*Field
	nestingLevel          int
	sourceRootPrefixIndex int
	filters               []Filter
}

func GetLogger(params ...*Field) BasicLogger {
	logger := &basicLogger{
		tags:         params,
		nestingLevel: 4,
		outputs:      []Output{&basicOutput{writer: os.Stdout, formatter: NewHumanReadableFormatter()}},
	}

	fpcs := make([]uintptr, 2)
	n := runtime.Callers(0, fpcs)
	if n != 0 {
		frames := runtime.CallersFrames(fpcs[:n])

		for {
			frame, more := frames.Next()
			if l := strings.Index(frame.File, "orbs-network-go/"); l > -1 {
				logger.sourceRootPrefixIndex = l + len("orbs-network-go/")
				break
			}

			if !more {
				break
			}
		}
	}

	return logger
}

func (b *basicLogger) getCaller(level int) (function string, source string) {
	fpcs := make([]uintptr, 1)

	// skip levels to get to the caller of logger function
	n := runtime.Callers(level, fpcs)
	if n == 0 {
		return "n/a", "n/a"
	}

	fun := runtime.FuncForPC(fpcs[0] - 1)
	if fun == nil {
		return "n/a", "n/a"
	}

	file, line := fun.FileLine(fpcs[0] - 1)
	fName := fun.Name()
	lastSlashOfName := strings.LastIndex(fName, "/")
	if lastSlashOfName > 0 {
		fName = fName[lastSlashOfName+1:]
	}
	return fName, fmt.Sprintf("%s:%d", file[b.sourceRootPrefixIndex:], line)
}

func (b *basicLogger) Tags() []*Field {
	return b.tags
}

func (b *basicLogger) WithTags(params ...*Field) BasicLogger {
	newTags := make([]*Field, len(b.tags))
	copy(newTags, b.tags)
	newTags = append(newTags, params...)
	//prefixes := append(b.tags, params...)
	return &basicLogger{tags: newTags, nestingLevel: b.nestingLevel, outputs: b.outputs, sourceRootPrefixIndex: b.sourceRootPrefixIndex, filters: b.filters}
}

func (b *basicLogger) Metric(params ...*Field) {
	b.Log("metric", "Metric recorded", params...)
}

func (b *basicLogger) Log(level string, message string, params ...*Field) {
	function, source := b.getCaller(b.nestingLevel)

	enrichmentParams := flattenParams(
		append(
			append(
				[]*Field{Function(function), Source(source)},
				b.tags...),
			params...),
	)

	for _, f := range b.filters {
		if !f.Allows(level, message, enrichmentParams) {
			return
		}
	}

	for _, output := range b.outputs {
		output.Append(level, message, enrichmentParams...)
	}
}

func (b *basicLogger) Info(message string, params ...*Field) {
	b.Log("info", message, params...)
}

func (b *basicLogger) Error(message string, params ...*Field) {
	b.Log("error", message, params...)
}

func (b *basicLogger) LogFailedExpectation(message string, expected *Field, actual *Field, params ...*Field) {
	actual.Key = "actual-" + actual.Key
	expected.Key = "expected-" + expected.Key
	newParams := append(params, expected, actual)
	b.Log("expectation", message, newParams...)
}

func (b *basicLogger) WithOutput(writers ...Output) BasicLogger {
	b.outputs = writers
	return b
}

func (b *basicLogger) WithFilters(filter ...Filter) BasicLogger {
	b.filters = append(b.filters, filter...) // this is not thread safe, I know
	return b
}

func (b *basicLogger) Filters() []Filter {
	return b.filters
}

func flattenParams(params []*Field) []*Field {
	var flattened []*Field
	for _, param := range params {
		if !param.IsNested() {
			flattened = append(flattened, param)
		} else if nestedFields, ok := param.Value().([]*Field); ok {
			flattened = append(flattened, flattenParams(nestedFields)...)
		} else {
			panic("log field of nested type did not return []*Field")
		}
	}
	return flattened
}
