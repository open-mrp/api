package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

type DocReader struct {
	fset *token.FileSet
	docs map[string]map[string]TypeDoc // pkgPath -> typeName -> TypeDoc
}

type TypeDoc struct {
	Doc    string
	Fields map[string]string // fieldName -> doc
}

func NewDocReader() *DocReader {
	return &DocReader{
		fset: token.NewFileSet(),
		docs: make(map[string]map[string]TypeDoc),
	}
}

func (r *DocReader) GetTypeDoc(t reflect.Type) TypeDoc {
	pkgPath := t.PkgPath()
	typeName := t.Name()

	if pkgPath == "" {
		return TypeDoc{}
	}

	// Handle generic types by using the base name for documentation lookup
	if idx := strings.Index(typeName, "["); idx != -1 {
		typeName = typeName[:idx]
	}

	if pkgDocs, ok := r.docs[pkgPath]; ok {
		if typeDoc, ok := pkgDocs[typeName]; ok {
			return typeDoc
		}
	}

	r.loadPackage(pkgPath)

	if pkgDocs, ok := r.docs[pkgPath]; ok {
		return pkgDocs[typeName]
	}

	return TypeDoc{}
}

func (r *DocReader) loadPackage(pkgPath string) {
	localDir := strings.TrimPrefix(pkgPath, "github.com/open-mrp/api/")
	if localDir == pkgPath {
		return
	}

	// Try to find the directory. It might be relative to the project root
	// or already relative to the current working directory.
	dir := localDir
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// Walk up to find a go.mod whose directory contains our target package.
		current, _ := os.Getwd()
		for {
			if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
				candidate := filepath.Join(current, localDir)
				if _, err := os.Stat(candidate); err == nil {
					dir = candidate
					break
				}
			}
			parent := filepath.Dir(current)
			if parent == current {
				break
			}
			current = parent
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("Warning: could not read directory %s: %v", dir, err)
		return
	}

	pkgDocs := make(map[string]TypeDoc)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())
		file, err := parser.ParseFile(r.fset, filePath, nil, parser.ParseComments)
		if err != nil {
			log.Printf("Warning: could not parse file %s: %v", filePath, err)
			continue
		}

		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}

			for _, spec := range gen.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				typeName := typeSpec.Name.Name
				typeDoc := TypeDoc{
					Doc:    strings.TrimSpace(gen.Doc.Text()),
					Fields: make(map[string]string),
				}

				structType, ok := typeSpec.Type.(*ast.StructType)
				if ok {
					for _, field := range structType.Fields.List {
						if len(field.Names) > 0 {
							fieldName := field.Names[0].Name
							typeDoc.Fields[fieldName] = strings.TrimSpace(field.Doc.Text())
						}
					}
				}

				pkgDocs[typeName] = typeDoc
			}
		}
	}

	r.docs[pkgPath] = pkgDocs
}
