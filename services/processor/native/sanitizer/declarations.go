// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package sanitizer

import (
	"github.com/pkg/errors"
	"go/ast"
	"go/token"
)

func (s *Sanitizer) verifyDeclarationsAndStatements(astFile *ast.File) (err error) {
	for _, decl := range astFile.Decls {
		ast.Inspect(decl, func(node ast.Node) bool {
			switch node.(type) {
			case *ast.ChanType:
				err = errors.New("channels not allowed")
				return false
			case *ast.GoStmt:
				err = errors.New("goroutines not allowed")
				return false
			case *ast.UnaryExpr:
				expr := node.(*ast.UnaryExpr)
				if expr.Op == token.ARROW {
					err = errors.New("sending to channels not allowed")
					return false
				}
			}
			return true
		})
	}

	return
}
