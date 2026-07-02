package services

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
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

func TestStructure_DomainFactoriesExistsWithRepositories(t *testing.T) {
	t.Parallel()
	dir := servicesDir(t)
	for _, svc := range listServices(t) {
		t.Run(svc, func(t *testing.T) {
			repoPath := filepath.Join(dir, svc, "internal", "domain", "repositories.go")
			if _, err := os.Stat(repoPath); os.IsNotExist(err) {
				return
			}
			factoryPath := filepath.Join(dir, svc, "internal", "domain", "factories.go")
			if _, err := os.Stat(factoryPath); os.IsNotExist(err) {
				t.Errorf("internal/domain/repositories.go exists without internal/domain/factories.go")
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

// mockComponent describes a domain interface-bearing component file and the
// mock subdir/package it must generate. Mirrors the component->dir->package
// mapping in scripts/generate-mocks.sh, which is the source of truth.
type mockComponent struct {
	dir     string // mock subdir under internal/domain/mock/
	pkgName string // generated Go package name inside that subdir
}

// canonicalMockComponents maps a domain component file base name (without .go)
// to its required mock subdir and package name.
var canonicalMockComponents = map[string]mockComponent{
	"factories":    {dir: "factory", pkgName: "factorymock"},
	"mediators":    {dir: "mediator", pkgName: "mediatormock"},
	"publishers":   {dir: "publisher", pkgName: "publishermock"},
	"repositories": {dir: "repository", pkgName: "repositorymock"},
	"services":     {dir: "service", pkgName: "servicemock"},
	"clients":      {dir: "client", pkgName: "clientmock"},
	"utils":        {dir: "utils", pkgName: "utilsmock"},
}

// interfaceDeclRe mirrors the exact line match scripts/generate-mocks.sh uses to
// decide whether a component file produces a mock (`grep -q "^type .*interface"`):
// a line beginning with `type ` that also contains `interface`. The generator is
// the source of truth, so this stays a line match rather than a semantic AST
// check on purpose -- an AST scan would diverge from the script on grouped
// `type (...)` interface blocks (which the line-anchored grep misses) and on
// non-interface types whose name merely contains "interface" (which the grep
// matches), producing spurious failures in these structure tests.
var interfaceDeclRe = regexp.MustCompile(`(?m)^type .*interface`)

// fileDeclaresInterface reports whether the Go file at path declares a mockable
// interface using the same line match the mock generator uses (see above).
func fileDeclaresInterface(t *testing.T, path string) bool {
	t.Helper()
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	return interfaceDeclRe.Match(src)
}

// mockDirPackageName returns the package name declared by the generated mock
// files in dir, or "" if the directory has no parseable Go files.
func mockDirPackageName(t *testing.T, dir string) string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, filepath.Join(dir, entry.Name()), nil, parser.PackageClauseOnly)
		if err != nil {
			t.Fatalf("failed to parse mock file %s: %v", entry.Name(), err)
		}
		return file.Name.Name
	}
	return ""
}

// TestStructure_InterfaceFilesHaveMatchingMocks asserts that every canonical
// domain component file declaring an interface has a generated mock package in
// the correct subdir with the correct package name.
func TestStructure_InterfaceFilesHaveMatchingMocks(t *testing.T) {
	t.Parallel()
	dir := servicesDir(t)
	for _, svc := range listServices(t) {
		t.Run(svc, func(t *testing.T) {
			domainDir := filepath.Join(dir, svc, "internal", "domain")
			for component, mc := range canonicalMockComponents {
				srcPath := filepath.Join(domainDir, component+".go")
				if _, err := os.Stat(srcPath); os.IsNotExist(err) {
					continue
				}
				if !fileDeclaresInterface(t, srcPath) {
					continue
				}
				mockDir := filepath.Join(domainDir, "mock", mc.dir)
				if _, err := os.Stat(mockDir); os.IsNotExist(err) {
					t.Errorf("%s.go declares an interface but mock/%s/ is missing (run `make mocks %s`)", component, mc.dir, svc)
					continue
				}
				if got := mockDirPackageName(t, mockDir); got != mc.pkgName {
					t.Errorf("mock/%s/ package is %q, want %q", mc.dir, got, mc.pkgName)
				}
			}
		})
	}
}

// TestStructure_NoOrphanMockDirs asserts that every subdir under
// internal/domain/mock/ corresponds to a canonical component file that exists
// and declares an interface; stale or hand-created mock dirs are flagged.
func TestStructure_NoOrphanMockDirs(t *testing.T) {
	t.Parallel()
	dir := servicesDir(t)

	dirToComponent := make(map[string]string, len(canonicalMockComponents))
	for component, mc := range canonicalMockComponents {
		dirToComponent[mc.dir] = component
	}

	for _, svc := range listServices(t) {
		t.Run(svc, func(t *testing.T) {
			domainDir := filepath.Join(dir, svc, "internal", "domain")
			mockRoot := filepath.Join(domainDir, "mock")
			entries, err := os.ReadDir(mockRoot)
			if os.IsNotExist(err) {
				return
			}
			if err != nil {
				t.Fatalf("failed to read mock dir: %v", err)
			}
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				component, ok := dirToComponent[entry.Name()]
				if !ok {
					t.Errorf("mock/%s/ is not a recognized mock subdir", entry.Name())
					continue
				}
				srcPath := filepath.Join(domainDir, component+".go")
				if _, err := os.Stat(srcPath); os.IsNotExist(err) {
					t.Errorf("mock/%s/ exists but %s.go does not (stale mock)", entry.Name(), component)
					continue
				}
				if !fileDeclaresInterface(t, srcPath) {
					t.Errorf("mock/%s/ exists but %s.go declares no interface (stale mock)", entry.Name(), component)
				}
			}
		})
	}
}

// TestStructure_ApiGatewayDomainNotRetrofitted guards the documented exception:
// api-gateway keeps a minimal domain package and must not adopt the backend
// services.go / repositories.go / transactional-factories layout.
func TestStructure_ApiGatewayDomainNotRetrofitted(t *testing.T) {
	t.Parallel()
	domainDir := filepath.Join(servicesDir(t), "api-gateway", "internal", "domain")
	for _, forbidden := range []string{"services.go", "repositories.go", "factories.go"} {
		if _, err := os.Stat(filepath.Join(domainDir, forbidden)); err == nil {
			t.Errorf("api-gateway/internal/domain/%s exists; api-gateway must keep a minimal domain package (see domain-layer-patterns.md Exceptions)", forbidden)
		}
	}
}
