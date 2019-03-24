// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package sanitizer

import (
	"bytes"
	"github.com/pkg/errors"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
)

type Sanitizer struct {
	config *SanitizerConfig
}

func NewSanitizer(config *SanitizerConfig) *Sanitizer {
	return &Sanitizer{
		config: config,
	}
}

func (s *Sanitizer) Process(code string) (string, error) {
	fset := token.NewFileSet()

	astFile, err := parser.ParseFile(fset, "", code, 0)
	if err != nil {
		return "", errors.Wrap(err, "native code verifier cannot parse source file")
	}

	err = s.verifyAll(astFile)
	if err != nil {
		return "", errors.Wrap(err, "native code verification error")
	}

	var resBuffer bytes.Buffer
	err = printer.Fprint(&resBuffer, fset, astFile)
	if err != nil {
		return "", errors.Wrap(err, "native code verifier cannot print source")
	}

	return resBuffer.String(), nil
}

func (s *Sanitizer) verifyAll(astFile *ast.File) error {
	err := s.verifyImports(astFile)
	if err != nil {
		return err
	}

	return nil
}
