// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package sanitizer

import (
	"github.com/pkg/errors"
	"go/ast"
	"strings"
)

func (s *Sanitizer) verifyImports(astFile *ast.File, allowedPrefixes []string) error {
	for _, importSpec := range astFile.Imports {
		if importSpec.Name != nil {
			return errors.New("custom import names not allowed")
		}

		importPath := importSpec.Path.Value
		if _, ok := s.config.ImportWhitelist[importPath]; !ok {
			if !foundInPrefixes(importPath, allowedPrefixes) {
				return errors.Errorf("import not allowed '%s'", importPath)
			}
		}
	}
	return nil
}

func foundInPrefixes(candidate string, allowedPrefixes []string) bool {
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(candidate, prefix) {
			return true
		}
	}
	return false
}
