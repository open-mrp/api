package services

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// servicesDir returns the absolute path to the services directory.
func servicesDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	return dir
}

// listServices returns all service directory names (excluding api-gateway).
func listServices(t *testing.T) []string {
	t.Helper()
	dir := servicesDir(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read services directory: %v", err)
	}

	var services []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Skip api-gateway (HTTP service, different structure) and hidden dirs.
		if entry.Name() == "api-gateway" || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		// Only include directories that look like services (contain cmd/).
		if _, err := os.Stat(filepath.Join(dir, entry.Name(), "cmd")); err == nil {
			services = append(services, entry.Name())
		}
	}
	return services
}

func TestStructure_CmdMainExists(t *testing.T) {
	t.Parallel()
	dir := servicesDir(t)
	for _, svc := range listServices(t) {
		t.Run(svc, func(t *testing.T) {
			path := filepath.Join(dir, svc, "cmd", "main.go")
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("missing cmd/main.go")
			}
		})
	}
}

func TestStructure_CmdRunExists(t *testing.T) {
	t.Parallel()
	dir := servicesDir(t)
	for _, svc := range listServices(t) {
		t.Run(svc, func(t *testing.T) {
			path := filepath.Join(dir, svc, "cmd", "run.go")
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("missing cmd/run.go")
			}
		})
	}
}

func TestStructure_CmdConfigExists(t *testing.T) {
	t.Parallel()
	dir := servicesDir(t)
	for _, svc := range listServices(t) {
		t.Run(svc, func(t *testing.T) {
			path := filepath.Join(dir, svc, "cmd", "config.go")
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("missing cmd/config.go")
			}
		})
	}
}

func TestStructure_DomainModelsExists(t *testing.T) {
	t.Parallel()
	dir := servicesDir(t)
	for _, svc := range listServices(t) {
		t.Run(svc, func(t *testing.T) {
			path := filepath.Join(dir, svc, "internal", "domain", "models.go")
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("missing internal/domain/models.go")
			}
		})
	}
}

func TestStructure_DomainServicesExists(t *testing.T) {
	t.Parallel()
	dir := servicesDir(t)
	for _, svc := range listServices(t) {
		t.Run(svc, func(t *testing.T) {
			path := filepath.Join(dir, svc, "internal", "domain", "services.go")
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("missing internal/domain/services.go")
			}
		})
	}
}

func TestStructure_MainCallsRun(t *testing.T) {
	t.Parallel()
	dir := servicesDir(t)
	for _, svc := range listServices(t) {
		t.Run(svc, func(t *testing.T) {
			path := filepath.Join(dir, svc, "cmd", "main.go")
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				t.Fatalf("failed to parse main.go: %v", err)
			}

			hasMain := false
			callsRun := false

			for _, decl := range file.Decls {
				funcDecl, ok := decl.(*ast.FuncDecl)
				if !ok || funcDecl.Name.Name != "main" {
					continue
				}
				hasMain = true
				ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
					callExpr, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}
					ident, ok := callExpr.Fun.(*ast.Ident)
					if ok && ident.Name == "Run" {
						callsRun = true
						return false
					}
					return true
				})
			}

			if !hasMain {
				t.Errorf("main.go does not contain a main() function")
			}
			if !callsRun {
				t.Errorf("main() does not call Run()")
			}
		})
	}
}
