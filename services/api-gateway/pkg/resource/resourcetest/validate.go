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
	// Register no-ops for the "enum" tag and all enum value fragments that the validator might interpret as separate tags. The "enum" tag is handled separately via reflection in handler.go; without these no-ops the validator panics on unknown tags.
	//
	// For example, `validate:"required,enum=user,api_key"` is parsed by the validator as three tags: "required", "enum=user", and "api_key".
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

	validateStructRequiredFields(t, name, resource)

	rv := reflect.ValueOf(resource)
	if rv.Kind() == reflect.Pointer {
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
		case reflect.Pointer:
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

func validateStructRequiredFields(t *testing.T, name string, resource any) {
	t.Helper()

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
}

func hasValidateTags(t reflect.Type) bool {
	for field := range t.Fields() {
		if field.Tag.Get("validate") != "" {
			return true
		}
	}
	return false
}

func hasJSONTags(t reflect.Type) bool {
	for field := range t.Fields() {
		if field.Tag.Get("json") != "" {
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
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil
	}
	rt := rv.Type()
	ns := make(map[string]bool)
	for ft := range rt.Fields() {
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
	if rv.Kind() == reflect.Pointer {
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
		case reflect.Pointer:
			if fv.IsNil() || fv.Elem().Kind() != reflect.Struct {
				continue
			}
			validateStubFields(t, fmt.Sprintf("%s.%s", name, ft.Name), fv.Elem())
		case reflect.Slice:
			for j := 0; j < fv.Len(); j++ {
				elem := fv.Index(j)
				if elem.Kind() == reflect.Pointer {
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

// ValidatePopulatedExpandableFields recursively validates non-nil expandable
// fields that are expected to be fully populated resources rather than stubs.
// It reuses ValidateResourceStruct for the nested resource itself, then
// descends into that nested resource's own expandable fields.
func ValidatePopulatedExpandableFields(t *testing.T, name string, resource any) {
	t.Helper()
	if resource == nil {
		return
	}
	rv := reflect.ValueOf(resource)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return
		}
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
		validatePopulatedExpandableValue(t, fmt.Sprintf("%s.%s", name, ft.Name), fv)
	}
}

func validatePopulatedExpandableValue(t *testing.T, path string, v reflect.Value) {
	t.Helper()

	if !v.IsValid() {
		return
	}

	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return
		}
		validatePopulatedExpandableValue(t, path, v.Elem())
	case reflect.Struct:
		if hasValidateTags(v.Type()) || hasJSONTags(v.Type()) {
			validateStructRequiredFields(t, path, v.Interface())
			ValidatePopulatedExpandableFields(t, path, v.Interface())
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			validatePopulatedExpandableValue(t, fmt.Sprintf("%s[%d]", path, i), v.Index(i))
		}
	}
}

// AssertExpandablesNil asserts that every expandable:"true" field reachable from resource is unset (nil pointer/slice/map, or zero value). This encodes the include contract: base mappers and gated *FromProto builders must leave expandable sub-resources empty so they are populated only when the caller explicitly requests them via ?include=. A non-nil expandable field returned straight from a mapper is the over-hydration bug this guards against (e.g. a sales-order line carrying a half-populated product stub, or a deleted product embedding its item/product_line).
//
// It recurses through non-expandable nested structs, pointers, and slices so an embedded resource (e.g. an order line inside a list) is checked too.
func AssertExpandablesNil(t *testing.T, name string, resource any) {
	t.Helper()
	rv := reflect.ValueOf(resource)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return
		}
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
		if ft.Tag.Get("expandable") == "true" {
			if !fv.IsZero() {
				t.Errorf("%s: expandable field %s (json:%q) is set — base mappers must leave it unset until requested via ?include=",
					name, ft.Name, resolveJSONNameFromType(rt, ft.Name))
			}
			continue
		}
		// Recurse into non-expandable nested structs/pointers/slices, which may themselves carry expandable fields.
		switch fv.Kind() {
		case reflect.Pointer:
			if !fv.IsNil() && fv.Elem().Kind() == reflect.Struct {
				AssertExpandablesNil(t, name+"."+ft.Name, fv.Interface())
			}
		case reflect.Struct:
			AssertExpandablesNil(t, name+"."+ft.Name, fv.Interface())
		case reflect.Slice:
			for j := 0; j < fv.Len(); j++ {
				el := fv.Index(j)
				if el.Kind() == reflect.Pointer || el.Kind() == reflect.Struct {
					AssertExpandablesNil(t, fmt.Sprintf("%s.%s[%d]", name, ft.Name, j), el.Interface())
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
	timeType := reflect.TypeFor[time.Time]()
	for i := 0; i < rv.NumField(); i++ {
		fv := rv.Field(i)
		ft := rt.Field(i)
		if !ft.IsExported() {
			continue
		}
		// Skip types that are legitimately zero on stubs.
		if ft.Type == timeType || fv.Kind() == reflect.Bool ||
			fv.Kind() == reflect.Pointer || fv.Kind() == reflect.Slice ||
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
	for f := range rt.Fields() {
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
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	rt := rv.Type()
	for f := range rt.Fields() {
		if f.Name == fieldName {
			tag := f.Tag.Get("json")
			if tag != "" {
				return strings.Split(tag, ",")[0]
			}
		}
	}
	return fieldName
}
