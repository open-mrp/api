//go:build integration

package sqlc

import (
	"context"
	"database/sql"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func TestPrepareQueriesSmoke(t *testing.T) {
	t.Parallel()
	dsn := os.Getenv("SQL_PREPARE_TEST_DSN")
	if dsn == "" {
		dsn = "root:Testing123!@tcp(localhost:3306)/openmrp?parseTime=true"
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping mysql: %v", err)
	}

	queries := collectSQLQueries(t)
	if len(queries) == 0 {
		t.Fatal("no SQL queries found in package")
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			stmt, err := db.PrepareContext(ctx, query)
			if err != nil {
				t.Errorf("prepare %s: %v", name, err)
				return
			}
			stmt.Close()
		})
	}
}

// collectSQLQueries parses all .sql.go files in the package directory and
// extracts unexported string constants that contain SQL queries.
func collectSQLQueries(t *testing.T) map[string]string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
		return strings.HasSuffix(fi.Name(), ".sql.go")
	}, 0)
	if err != nil {
		t.Fatalf("parse dir: %v", err)
	}

	queries := make(map[string]string)
	for _, pkg := range pkgs {
		for fpath, file := range pkg.Files {
			base := filepath.Base(fpath)
			if !strings.HasSuffix(base, ".sql.go") {
				continue
			}
			for _, decl := range file.Decls {
				gd, ok := decl.(*ast.GenDecl)
				if !ok || gd.Tok != token.CONST {
					continue
				}
				for _, spec := range gd.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok || len(vs.Values) != 1 {
						continue
					}
					bl, ok := vs.Values[0].(*ast.BasicLit)
					if !ok || bl.Kind != token.STRING {
						continue
					}
					name := vs.Names[0].Name
					// Only include unexported consts (sqlc-generated query strings)
					if ast.IsExported(name) {
						continue
					}
					val := strings.Trim(bl.Value, "`\"")
					// Strip the sqlc comment prefix (e.g. "-- name: FooBar :one\n")
					if idx := strings.Index(val, "\n"); idx != -1 {
						val = val[idx+1:]
					}
					queries[name] = val
				}
			}
		}
	}
	return queries
}
