package resourcetest

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
	noop := func(fl validator.FieldLevel) bool { return true }
	// Register no-ops for the "enum" tag and all enum value fragments that
	// the validator might interpret as separate tags. The "enum" tag is
	// handled separately via reflection in handler.go; without these no-ops
	// the validator panics on unknown tags.
	//
	// For example, `validate:"required,enum=user,api_key"` is parsed by
	// the validator as three tags: "required", "enum=user", and "api_key".
	_ = validate.RegisterValidation("enum", noop)
	// Tags that appear as OR-fragments or comma-fragments in enum values.
	// e.g. "enum=user,api_key" → tags "enum=user" and "api_key"
	// e.g. "enum=user|api_key|agent" → tags "enum=user", "api_key", "agent"
	_ = validate.RegisterValidation("api_key", noop)
	_ = validate.RegisterValidation("agent", noop)
	_ = validate.RegisterValidation("active", noop)
	_ = validate.RegisterValidation("inactive", noop)
}

// ValidateResourceStruct validates a presenter output struct by:
//  1. Running validate:"required" tag checks via go-playground/validator.
//  2. Checking that every exported, non-pointer field is non-zero. This catches
//     any field the presenter forgot to map. Test fixtures MUST use non-zero
//     values for all fields (e.g. true for bools, 1 for ints) so that a zero
//     value in the output always means "presenter didn't set this."
//  3. Recursively validating non-nil nested struct pointer fields and slice
//     elements.
func ValidateResourceStruct(t *testing.T, name string, resource any) {
	t.Helper()

	if resource == nil {
		t.Errorf("%s: resource is nil", name)
		return
	}

	// Build a set of expandable field namespaces so we can skip validation
	// errors from nested expandable sub-resource structs. The go-playground
	// validator recurses into non-nil struct pointer fields, but expandable
	// sub-resources are deliberately partial (ID + object only).
	expandableNS := collectExpandableNamespaces(resource)

	err := validate.Struct(resource)
	if err != nil {
		validationErrors, ok := err.(validator.ValidationErrors)
		if !ok {
			t.Errorf("%s: unexpected validation error: %v", name, err)
			return
		}
		for _, fe := range validationErrors {
			if isInsideExpandable(fe.Namespace(), expandableNS) {
				continue
			}
			jsonName := resolveJSONName(resource, fe.StructField())
			t.Errorf("%s: field %s (json:%q) failed %q validation", name, fe.StructField(), jsonName, fe.Tag())
		}
	}

	rv := reflect.ValueOf(resource)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return
	}
	rt := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		fv := rv.Field(i)
		ft := rt.Field(i)
		if !ft.IsExported() {
			continue
		}

		switch fv.Kind() {
		case reflect.Ptr:
			// Expandable pointer fields are deliberately partial sub-resource
			// references (ID + object only) until the caller requests expansion.
			// Skip recursive validation for these fields.
			if ft.Tag.Get("expandable") == "true" {
				break
			}
			// Pointer fields are optional — only recurse if non-nil.
			if !fv.IsNil() && fv.Elem().Kind() == reflect.Struct {
				if hasValidateTags(fv.Elem().Type()) || hasJSONTags(fv.Elem().Type()) {
					ValidateResourceStruct(t, fmt.Sprintf("%s.%s", name, ft.Name), fv.Interface())
				}
			}
		case reflect.Slice:
			// Expandable slice fields are deliberately partial sub-resource
			// references until the caller requests expansion. Skip validation.
			if ft.Tag.Get("expandable") == "true" {
				break
			}
			// Validate each struct element in non-empty slices.
			for j := 0; j < fv.Len(); j++ {
				elem := fv.Index(j)
				if elem.Kind() == reflect.Struct {
					ValidateResourceStruct(t, fmt.Sprintf("%s.%s[%d]", name, ft.Name, j), elem.Interface())
				}
			}
		case reflect.Map:
			// Maps are optional (nil = not included), similar to pointers.
		default:
			// PageInfo on List is often empty for embedded sub-resource lists; skip zero check.
			if ft.Name == "PageInfo" {
				break
			}
			// Every non-pointer, non-slice field must be non-zero.
			if fv.IsZero() {
				jsonName := resolveJSONName(resource, ft.Name)
				t.Errorf("%s: field %s (json:%q) is zero — presenter did not set it", name, ft.Name, jsonName)
			}
			// Recurse into inline struct fields (e.g. embedded structs).
			if fv.Kind() == reflect.Struct && (hasValidateTags(fv.Type()) || hasJSONTags(fv.Type())) {
				ValidateResourceStruct(t, fmt.Sprintf("%s.%s", name, ft.Name), fv.Interface())
			}
		}
	}
}

func hasValidateTags(t reflect.Type) bool {
	for i := 0; i < t.NumField(); i++ {
		if t.Field(i).Tag.Get("validate") != "" {
			return true
		}
	}
	return false
}

func hasJSONTags(t reflect.Type) bool {
	for i := 0; i < t.NumField(); i++ {
		if t.Field(i).Tag.Get("json") != "" {
			return true
		}
	}
	return false
}

// collectExpandableNamespaces returns the validator namespace prefixes for
// expandable struct pointer fields. For example, if the struct type is "Sandbox"
// and field "OwnerAccount" has tag expandable:"true", the returned set contains
// "Sandbox.OwnerAccount.".
func collectExpandableNamespaces(resource any) map[string]bool {
	rv := reflect.ValueOf(resource)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil
	}
	rt := rv.Type()
	ns := make(map[string]bool)
	for i := 0; i < rt.NumField(); i++ {
		ft := rt.Field(i)
		if ft.Tag.Get("expandable") == "true" {
			prefix := rt.Name() + "." + ft.Name
			// For pointer fields the namespace is "Type.Field.SubField".
			// For slice fields the namespace is "Type.Field[0].SubField".
			// We store the prefix without trailing separator to match both.
			ns[prefix+"."] = true
			ns[prefix+"["] = true
		}
	}
	return ns
}

// isInsideExpandable checks whether a validator namespace path falls inside
// an expandable sub-resource.
func isInsideExpandable(namespace string, expandableNS map[string]bool) bool {
	for prefix := range expandableNS {
		if strings.HasPrefix(namespace, prefix) {
			return true
		}
	}
	return false
}

// ValidateExpandableStubs validates that non-nil expandable sub-resource fields
// have their required string-typed fields populated. This catches the common bug
// where a presenter builds a stub (ID + Object only) but forgets enum fields
// like CustomerPortalVisibility, Status, or Type that default to "".
//
// Only exported, non-pointer, string-kind fields with validate:"required" are
// checked. Time, bool, pointer, slice, and map fields are skipped because stubs
// legitimately omit those.
func ValidateExpandableStubs(t *testing.T, name string, resource any) {
	t.Helper()
	if resource == nil {
		return
	}
	rv := reflect.ValueOf(resource)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return
	}
	rt := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		fv := rv.Field(i)
		ft := rt.Field(i)
		if ft.Tag.Get("expandable") != "true" || !ft.IsExported() {
			continue
		}
		switch fv.Kind() {
		case reflect.Ptr:
			if fv.IsNil() || fv.Elem().Kind() != reflect.Struct {
				continue
			}
			validateStubFields(t, fmt.Sprintf("%s.%s", name, ft.Name), fv.Elem())
		case reflect.Slice:
			for j := 0; j < fv.Len(); j++ {
				elem := fv.Index(j)
				if elem.Kind() == reflect.Ptr {
					if elem.IsNil() {
						continue
					}
					elem = elem.Elem()
				}
				if elem.Kind() == reflect.Struct {
					validateStubFields(t, fmt.Sprintf("%s.%s[%d]", name, ft.Name, j), elem)
				}
			}
		}
	}
}

// validateStubFields checks that required string-kind fields on a sub-resource
// struct are non-zero. Skips time.Time, bool, pointer, slice, and map fields.
func validateStubFields(t *testing.T, path string, rv reflect.Value) {
	t.Helper()
	rt := rv.Type()
	timeType := reflect.TypeOf((*time.Time)(nil)).Elem()
	for i := 0; i < rv.NumField(); i++ {
		fv := rv.Field(i)
		ft := rt.Field(i)
		if !ft.IsExported() {
			continue
		}
		// Skip types that are legitimately zero on stubs.
		if ft.Type == timeType || fv.Kind() == reflect.Bool ||
			fv.Kind() == reflect.Ptr || fv.Kind() == reflect.Slice ||
			fv.Kind() == reflect.Map {
			continue
		}
		// Only check string-kind fields (string and typed string constants).
		if fv.Kind() != reflect.String {
			continue
		}
		// Only check fields tagged validate:"required".
		if !strings.Contains(ft.Tag.Get("validate"), "required") {
			continue
		}
		if fv.IsZero() {
			jsonName := resolveJSONNameFromType(rt, ft.Name)
			t.Errorf("%s: field %s (json:%q) is zero on expandable stub — presenter did not set it", path, ft.Name, jsonName)
		}
	}
}

// resolveJSONNameFromType resolves the JSON tag name for a field by type.
func resolveJSONNameFromType(rt reflect.Type, fieldName string) string {
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if f.Name == fieldName {
			tag := f.Tag.Get("json")
			if tag != "" {
				return strings.Split(tag, ",")[0]
			}
		}
	}
	return fieldName
}

func resolveJSONName(resource any, fieldName string) string {
	rv := reflect.ValueOf(resource)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if f.Name == fieldName {
			tag := f.Tag.Get("json")
			if tag != "" {
				return strings.Split(tag, ",")[0]
			}
		}
	}
	return fieldName
}
