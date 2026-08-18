package types

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// A PermissionDomain constant is only enforceable if the database actually carries the row: CheckHasAnyPermission reads role_permission, and a role cannot be granted a permission code the permission table has never heard of. A domain added here but not seeded therefore 403s every non-admin caller — silently, because admins short-circuit the check and never see it.
//
// This also guards the reverse: a seeded code with no Go constant is dead weight that the dashboard's PermissionDomains enum will keep rendering as a grantable box that gates nothing.
func TestPermissionDomainsMatchSeed(t *testing.T) {
	t.Parallel()

	domains := parsePermissionDomains(t)
	seeded, grants := parseAuthSeed(t)

	assertSetsEqual(t, "seeded permission rows", domains, seeded)
	assertSetsEqual(t, "Admin role grants", domains, grants)
}

// Every seeded permission points at a permission_group by code, and the FK on permission.permission_group_code means a typo here fails the whole seed rather than one row.
func TestSeededPermissionGroupsExist(t *testing.T) {
	t.Parallel()

	_, groupsByPermission := parseAuthSeedGroups(t)
	groups := parsePermissionGroupSeed(t)

	for permission, group := range groupsByPermission {
		if !groups[group] {
			t.Errorf("permission %q is seeded into group %q, which 0001_static_types.sql does not seed", permission, group)
		}
	}
}

func apiRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve caller info")
	}
	// pkg/types -> pkg -> auth-service -> services -> api
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..")
}

func readSeed(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(apiRoot(t), "shared", "db", "seed", name)
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(contents)
}

// collects the PermissionDomain constant values by parsing this package's source, the only way to enumerate untyped Go constants.
func parsePermissionDomains(t *testing.T) map[string]bool {
	t.Helper()

	path := filepath.Join(apiRoot(t), "services", "auth-service", "pkg", "types", "permissions.go")
	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	domains := map[string]bool{}
	ast.Inspect(parsed, func(n ast.Node) bool {
		spec, ok := n.(*ast.ValueSpec)
		if !ok {
			return true
		}
		ident, ok := spec.Type.(*ast.Ident)
		if !ok || ident.Name != "PermissionDomain" {
			return true
		}
		for _, value := range spec.Values {
			lit, ok := value.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				continue
			}
			code, err := strconv.Unquote(lit.Value)
			if err != nil {
				t.Fatalf("unquote %s: %v", lit.Value, err)
			}
			domains[code] = true
		}
		return true
	})

	if len(domains) == 0 {
		t.Fatal("found no PermissionDomain constants")
	}
	return domains
}

var (
	seedPermissionRow = regexp.MustCompile(`\('([a-z_]+)', '([a-z_]+)', '[^']+', '([a-z_]+)'`)
	seedAdminGrantRow = regexp.MustCompile(`\('rlpm_[a-z0-9_]+', 'rl_mtg88e6u6fbu', '([a-z_]+)'`)
	seedGroupRow      = regexp.MustCompile(`\('pmgp_[a-z0-9_]+', '([a-z_]+)'`)
)

func parseAuthSeed(t *testing.T) (permissions, adminGrants map[string]bool) {
	t.Helper()
	permissions, _ = parseAuthSeedGroups(t)

	adminGrants = map[string]bool{}
	for _, match := range seedAdminGrantRow.FindAllStringSubmatch(readSeed(t, "0004_auth.sql"), -1) {
		adminGrants[match[1]] = true
	}
	return permissions, adminGrants
}

func parseAuthSeedGroups(t *testing.T) (permissions map[string]bool, groupsByPermission map[string]string) {
	t.Helper()

	permissions = map[string]bool{}
	groupsByPermission = map[string]string{}
	for _, match := range seedPermissionRow.FindAllStringSubmatch(readSeed(t, "0004_auth.sql"), -1) {
		// The permission block writes the code as both id and code; the id/code mismatch filters out other inserts with a similar shape.
		if match[1] != match[2] {
			continue
		}
		permissions[match[1]] = true
		groupsByPermission[match[1]] = match[3]
	}

	if len(permissions) == 0 {
		t.Fatal("found no seeded permission rows")
	}
	return permissions, groupsByPermission
}

func parsePermissionGroupSeed(t *testing.T) map[string]bool {
	t.Helper()

	groups := map[string]bool{}
	for _, match := range seedGroupRow.FindAllStringSubmatch(readSeed(t, "0001_static_types.sql"), -1) {
		groups[match[1]] = true
	}

	if len(groups) == 0 {
		t.Fatal("found no seeded permission_group rows")
	}
	return groups
}

func assertSetsEqual(t *testing.T, label string, want, got map[string]bool) {
	t.Helper()

	if missing := difference(want, got); len(missing) > 0 {
		t.Errorf("%s missing for PermissionDomain constants: %s", label, strings.Join(missing, ", "))
	}
	if extra := difference(got, want); len(extra) > 0 {
		t.Errorf("%s exist with no PermissionDomain constant: %s", label, strings.Join(extra, ", "))
	}
}

func difference(a, b map[string]bool) []string {
	var only []string
	for code := range a {
		if !b[code] {
			only = append(only, code)
		}
	}
	sort.Strings(only)
	return only
}
