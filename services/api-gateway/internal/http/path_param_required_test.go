package httptransport

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func TestPathParamsRequireValidateRequired(t *testing.T) {
	t.Parallel(
	// Rule: any request struct field tagged with `path:"..."` must also have
	// `validate:"required"` so that missing/misbound path params are rejected.
	//
	// This is intentionally enforced via static analysis of endpoint source,
	// not runtime binding.
	)

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve caller info")
	}

	// internal/http -> internal -> api-gateway -> endpoints
	endpointsDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "endpoints")

	var violations []string
	fset := token.NewFileSet()

	walkErr := filepath.WalkDir(endpointsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		parsed, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return err
		}

		ast.Inspect(parsed, func(n ast.Node) bool {
			structType, ok := n.(*ast.StructType)
			if !ok || structType.Fields == nil {
				return true
			}

			for _, field := range structType.Fields.List {
				if field.Tag == nil {
					continue
				}

				tagText, err := strconv.Unquote(field.Tag.Value)
				if err != nil {
					pos := fset.Position(field.Pos())
					violations = append(violations, fmt.Sprintf("%s:%d: failed to unquote struct tags: %v", path, pos.Line, err))
					continue
				}

				st := reflect.StructTag(tagText)
				pathKey := st.Get("path")
				if pathKey == "" {
					continue
				}

				validateRules := st.Get("validate")
				// Footgun guard: if someone uses `omitempty`, `required` may be bypassed.
				if validateRules == "" || !strings.Contains(validateRules, "required") || strings.Contains(validateRules, "omitempty") {
					pos := fset.Position(field.Pos())
					fieldName := "<anonymous>"
					if len(field.Names) > 0 && field.Names[0] != nil {
						fieldName = field.Names[0].Name
					}
					violations = append(violations, fmt.Sprintf("%s:%d: field %s has path=%q but validate=%q (must include required and not include omitempty)", path, pos.Line, fieldName, pathKey, validateRules))
				}
			}

			return true
		})

		return nil
	})

	if walkErr != nil {
		t.Fatalf("failed scanning endpoints for path params: %v", walkErr)
	}

	if len(violations) == 0 {
		return
	}

	sort.Strings(violations)
	t.Fatalf("found path params without validate:\"required\":\n%s", strings.Join(violations, "\n"))
}
