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

// These tests statically enforce two conventions on field.Optional[T] and
// field.Clearable[T] request fields. Both are checked by scanning endpoint
// source rather than at runtime, so a violation fails the build.
//
//   1. They are never tagged validate:"required" (TestOptionalFieldsNeverValidateRequired).
//   2. Their json tag always carries ,omitzero (TestOptionalFieldsUseOmitzero).

// wrapperField is a field.Optional[T] / field.Clearable[T] declaration found in
// a request struct, captured for the static convention checks below.
type wrapperField struct {
	pos         string
	name        string
	kind        string // "field.Optional" or "field.Clearable"
	jsonTag     string
	validateTag string
}

// requestStructDirs are the source roots that hold API request structs and the
// shared input fragments they embed. field.Optional/field.Clearable only ever
// belong on request inputs, so these are the trees we police.
func requestStructDirs(t *testing.T) []string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve caller info")
	}
	// internal/http -> internal -> api-gateway
	base := filepath.Join(filepath.Dir(thisFile), "..", "..")
	return []string{
		filepath.Join(base, "endpoints"),
		filepath.Join(base, "pkg", "request"),
	}
}

// collectWrapperFields statically scans the request source trees and returns
// every field.Optional[T] / field.Clearable[T] field it finds.
func collectWrapperFields(t *testing.T) []wrapperField {
	t.Helper()
	fset := token.NewFileSet()
	var found []wrapperField

	for _, root := range requestStructDirs(t) {
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			parsed, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return err
			}
			found = append(found, wrapperFieldsInFile(fset, parsed)...)
			return nil
		})
		if err != nil {
			t.Fatalf("scanning %s: %v", root, err)
		}
	}

	// Guard against a silently-empty scan (e.g. the roots moved): the checks
	// below would vacuously pass and stop protecting anything.
	if len(found) == 0 {
		t.Fatal("found no field.Optional/field.Clearable request fields; scan roots may be wrong")
	}
	return found
}

// wrapperFieldsInFile returns every field.Optional[T] / field.Clearable[T] field
// declared in any struct type within file.
func wrapperFieldsInFile(fset *token.FileSet, file *ast.File) []wrapperField {
	var found []wrapperField
	ast.Inspect(file, func(n ast.Node) bool {
		st, ok := n.(*ast.StructType)
		if !ok || st.Fields == nil {
			return true
		}
		for _, f := range st.Fields.List {
			kind := wrapperKind(f.Type)
			if kind == "" {
				continue
			}
			name := "<embedded>"
			if len(f.Names) > 0 {
				name = f.Names[0].Name
			}
			var jsonTag, validateTag string
			if f.Tag != nil {
				if tagText, err := strconv.Unquote(f.Tag.Value); err == nil {
					tag := reflect.StructTag(tagText)
					jsonTag = tag.Get("json")
					validateTag = tag.Get("validate")
				}
			}
			found = append(found, wrapperField{
				pos:         fset.Position(f.Pos()).String(),
				name:        name,
				kind:        kind,
				jsonTag:     jsonTag,
				validateTag: validateTag,
			})
		}
		return true
	})
	return found
}

// requiredRule returns the offending validate rule (e.g. "required", "required_if")
// when validateTag asserts any form of requiredness, since wrapper fields must
// always be optional.
func requiredRule(validateTag string) (string, bool) {
	for _, rule := range strings.Split(validateTag, ",") {
		rule = strings.TrimSpace(rule)
		if rule == "required" || strings.HasPrefix(rule, "required_") {
			return rule, true
		}
	}
	return "", false
}

// usesOmitzero reports whether jsonTag omits the field when unset: either it
// carries the ,omitzero option or it is not serialized at all (json:"-").
func usesOmitzero(jsonTag string) bool {
	opts := strings.Split(jsonTag, ",")
	if len(opts) > 0 && opts[0] == "-" {
		return true // not serialized; omission rules do not apply
	}
	for _, o := range opts[1:] {
		if strings.TrimSpace(o) == "omitzero" {
			return true
		}
	}
	return false
}

// wrapperKind returns "field.Optional" or "field.Clearable" when expr is one of
// those generic types (or a pointer to one), otherwise "".
func wrapperKind(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.StarExpr:
		return wrapperKind(e.X)
	case *ast.IndexExpr: // field.Optional[T]
		return wrapperKind(e.X)
	case *ast.IndexListExpr:
		return wrapperKind(e.X)
	case *ast.SelectorExpr:
		pkg, ok := e.X.(*ast.Ident)
		if !ok || pkg.Name != "field" {
			return ""
		}
		switch e.Sel.Name {
		case "Optional":
			return "field.Optional"
		case "Clearable":
			return "field.Clearable"
		}
	}
	return ""
}

func TestOptionalFieldsNeverValidateRequired(t *testing.T) {
	t.Parallel()
	// field.Optional[T] and field.Clearable[T] model inputs the client may always
	// omit, so the OpenAPI generator never marks them required. A required rule on
	// one is a contradiction: the schema would say optional while runtime validation
	// rejects an absent value. A genuinely required field uses a plain value type T
	// with validate:"required".
	var violations []string
	for _, wf := range collectWrapperFields(t) {
		if rule, ok := requiredRule(wf.validateTag); ok {
			violations = append(violations, fmt.Sprintf(
				"%s: %s %s[...] has validate=%q — %s is always optional; drop the %q rule (use a plain value type T for a required field)",
				wf.pos, wf.name, wf.kind, wf.validateTag, wf.kind, rule))
		}
	}
	failOnWrapperViolations(t, violations, "field.Optional/field.Clearable fields tagged required")
}

func TestOptionalFieldsUseOmitzero(t *testing.T) {
	t.Parallel()
	// encoding/json decides omission from the struct tag, not the field's type.
	// Only the ,omitzero option (Go 1.24+) makes the encoder call IsZero() and drop
	// an unset wrapper; without it, marshaling an unset field.Optional/field.Clearable
	// returns "patch: cannot marshal unset field". ,omitempty does NOT help — it never
	// omits a struct — so every wrapper field must use json:"<name>,omitzero".
	var violations []string
	for _, wf := range collectWrapperFields(t) {
		if !usesOmitzero(wf.jsonTag) {
			violations = append(violations, fmt.Sprintf(
				"%s: %s %s[...] has json=%q — wrapper fields must use json:\"<name>,omitzero\"",
				wf.pos, wf.name, wf.kind, wf.jsonTag))
		}
	}
	failOnWrapperViolations(t, violations, "field.Optional/field.Clearable fields missing ,omitzero")
}

func failOnWrapperViolations(t *testing.T, violations []string, summary string) {
	t.Helper()
	if len(violations) == 0 {
		return
	}
	sort.Strings(violations)
	t.Fatalf("found %d %s:\n%s", len(violations), summary, strings.Join(violations, "\n"))
}

// TestWrapperConventionDetection proves the scanner and predicates actually flag
// violations, so the guardrail tests above can't pass vacuously.
func TestWrapperConventionDetection(t *testing.T) {
	t.Parallel()
	const src = `package fixture

import "github.com/augno/api/shared/field"

type Req struct {
	Plain     string                 ` + "`json:\"plain\" validate:\"required\"`" + `
	Good      field.Optional[string] ` + "`json:\"good,omitzero\" validate:\"omitempty,max=255\"`" + `
	GoodClear field.Clearable[string] ` + "`json:\"good_clear,omitzero\"`" + `
	BadReq    field.Optional[string] ` + "`json:\"bad_req,omitzero\" validate:\"required\"`" + `
	BadReqIf  field.Clearable[string] ` + "`json:\"bad_req_if,omitzero\" validate:\"required_if=Plain x\"`" + `
	BadOmit   field.Optional[string] ` + "`json:\"bad_omit\"`" + `
}
`
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, "fixture.go", src, 0)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	fields := wrapperFieldsInFile(fset, parsed)

	got := map[string]wrapperField{}
	for _, f := range fields {
		got[f.name] = f
	}

	// The plain string field must NOT be picked up as a wrapper.
	if _, ok := got["Plain"]; ok {
		t.Error("plain string field was misdetected as a wrapper")
	}
	// All five wrapper fields must be detected (and nothing else).
	wantNames := []string{"Good", "GoodClear", "BadReq", "BadReqIf", "BadOmit"}
	if len(fields) != len(wantNames) {
		t.Errorf("expected %d wrapper fields, got %d: %+v", len(wantNames), len(fields), fields)
	}
	for _, n := range wantNames {
		if _, ok := got[n]; !ok {
			t.Errorf("expected to detect wrapper field %q", n)
		}
	}

	// requiredRule must flag the required / required_if fields and nothing else.
	for name, wantRule := range map[string]string{"BadReq": "required", "BadReqIf": "required_if=Plain x"} {
		if rule, ok := requiredRule(got[name].validateTag); !ok || rule != wantRule {
			t.Errorf("%s: requiredRule(%q) = (%q, %v), want (%q, true)", name, got[name].validateTag, rule, ok, wantRule)
		}
	}
	for _, name := range []string{"Good", "GoodClear", "BadOmit"} {
		if rule, ok := requiredRule(got[name].validateTag); ok {
			t.Errorf("%s: requiredRule(%q) unexpectedly flagged %q", name, got[name].validateTag, rule)
		}
	}

	// usesOmitzero must accept the ,omitzero fields and reject the bare one.
	for _, name := range []string{"Good", "GoodClear", "BadReq", "BadReqIf"} {
		if !usesOmitzero(got[name].jsonTag) {
			t.Errorf("%s: usesOmitzero(%q) = false, want true", name, got[name].jsonTag)
		}
	}
	if usesOmitzero(got["BadOmit"].jsonTag) {
		t.Errorf("BadOmit: usesOmitzero(%q) = true, want false", got["BadOmit"].jsonTag)
	}
}
