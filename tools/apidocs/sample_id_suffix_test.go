package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestSampleIDNanoSuffixesUnique ensures each Sample* string ID uses a distinct nano segment
// (substring after the first "_") so docs examples do not look like the same ID with a
// different type prefix. Exception: multiple constants may intentionally share the same *full*
// literal string (rare); that would produce one tail entry with one source name only.
func TestSampleIDNanoSuffixesUnique(t *testing.T) {
	root := filepath.Join("..", "..", "services", "api-gateway", "pkg", "resource")
	tailToNames := map[string][]string{}
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, root, nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
			ast.Inspect(f, func(n ast.Node) bool {
				gs, ok := n.(*ast.GenDecl)
				if !ok || gs.Tok != token.CONST {
					return true
				}
				for _, spec := range gs.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok || len(vs.Names) != 1 || len(vs.Values) != 1 {
						continue
					}
					name := vs.Names[0].Name
					if !strings.HasPrefix(name, "Sample") {
						continue
					}
					if !strings.HasSuffix(name, "ID") && !strings.HasPrefix(name, "SamplePlanTypeID") {
						continue
					}
					bl, ok := vs.Values[0].(*ast.BasicLit)
					if !ok || bl.Kind != token.STRING {
						continue
					}
					s, err := strconv.Unquote(bl.Value)
					if err != nil {
						continue
					}
					i := strings.IndexByte(s, '_')
					if i < 0 || i+1 >= len(s) {
						t.Errorf("%s: missing underscore in sample id %q", name, s)
						continue
					}
					tail := s[i+1:]
					tailToNames[tail] = append(tailToNames[tail], name)
				}
				return true
			})
		}
	}
	for tail, names := range tailToNames {
		if len(names) < 2 {
			continue
		}
		// Only report true duplicates: distinct const names must not share the same nano tail.
		uniq := map[string]struct{}{}
		for _, n := range names {
			uniq[n] = struct{}{}
		}
		if len(uniq) < 2 {
			continue
		}
		t.Errorf("sample id nano %q is reused by %v — use a distinct suffix per const", tail, names)
	}
}
