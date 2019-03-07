package sanitizer

import (
	"github.com/pkg/errors"
	"go/ast"
)

func (s *sanitizer) verifyImports(astFile *ast.File) error {
	for _, importSpec := range astFile.Imports {
		importPath := importSpec.Path.Value
		if _, ok := s.config.ImportWhitelist[importPath]; !ok {
			return errors.Errorf("import not allowed '%s'", importPath)
		}
	}
	return nil
}
