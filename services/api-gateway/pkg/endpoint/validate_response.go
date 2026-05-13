package apiendpoint

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

// ValidateExpandableFields checks that non-nil expandable sub-resource fields
// included in the API response satisfy validate:"required" on each exported
// string-kind and time.Time field (matching resourcetest.ValidateExpandableStubs).
// Non-requested expandable fields are collapsed to null before the client sees
// the response; fields left in the payload after an include must be complete.
//
// Nested include paths (e.g. flat_rate.unit) are validated by descending into
// non-expandable wrapper structs (e.g. Quantity under flat_rate). List wrappers
// (*List[T]) validate each element of Data.
//
// Only fields in the requested include set are validated.
func ValidateExpandableFields(resource any, requested map[string]bool, config *IncludeConfig) error {
	if config == nil || resource == nil {
		return nil
	}

	// Build set of JSON paths that will remain in the response.
	includedPaths := buildIncludedPaths(requested, config)
	if len(includedPaths) == 0 {
		return nil
	}

	rv := reflect.ValueOf(resource)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil
	}

	// Handle list responses: validate each element in Data.
	if dataField := rv.FieldByName("Data"); dataField.IsValid() && dataField.Kind() == reflect.Slice {
		var errs []string
		for i := 0; i < dataField.Len(); i++ {
			elem := dataField.Index(i)
			if elem.Kind() == reflect.Pointer {
				if elem.IsNil() {
					continue
				}
				elem = elem.Elem()
			}
			if elem.Kind() == reflect.Struct {
				if err := validateExpandableOnStruct(elem, includedPaths); err != nil {
					errs = append(errs, fmt.Sprintf("data[%d]: %s", i, err))
				}
			}
		}
		if len(errs) > 0 {
			return fmt.Errorf("expandable field validation failed: %s", strings.Join(errs, "; "))
		}
		return nil
	}

	return validateExpandableOnStruct(rv, includedPaths)
}

func jsonFieldName(ft reflect.StructField) string {
	tag := ft.Tag.Get("json")
	if tag == "" || tag == "-" {
		return ""
	}
	return strings.Split(tag, ",")[0]
}

// pathsNestedUnderPrefix returns included path keys with the prefix removed,
// for continuing validation inside a nested JSON object.
func pathsNestedUnderPrefix(includedPaths map[string]bool, prefix string) map[string]bool {
	out := make(map[string]bool)
	for p := range includedPaths {
		if strings.HasPrefix(p, prefix) {
			suffix := strings.TrimPrefix(p, prefix)
			if suffix != "" {
				out[suffix] = true
			}
		}
	}
	return out
}

// buildIncludedPaths returns the set of top-level JSON paths that will remain
// in the response after collapse.
func buildIncludedPaths(requested map[string]bool, config *IncludeConfig) map[string]bool {
	paths := make(map[string]bool)
	fields := config.FieldsByKey()
	for key := range requested {
		if f, ok := fields[key]; ok {
			for _, p := range f.JSONPaths {
				paths[p] = true
			}
		}
	}
	return paths
}

// validateExpandableOnStruct validates expandable fields on a single struct value.
func validateExpandableOnStruct(rv reflect.Value, includedPaths map[string]bool) error {
	rt := rv.Type()
	var errs []string

	for i := 0; i < rt.NumField(); i++ {
		ft := rt.Field(i)
		if ft.Tag.Get("expandable") != "true" || !ft.IsExported() {
			continue
		}

		jsonName := jsonFieldName(ft)
		if jsonName == "" {
			continue
		}

		if !includedPaths[jsonName] {
			// Nested includes under this expandable field (e.g. owner.account).
			hasNestedInclude := false
			for p := range includedPaths {
				if strings.HasPrefix(p, jsonName+".") {
					hasNestedInclude = true
					break
				}
			}
			if !hasNestedInclude {
				continue
			}
			fv := rv.Field(i)
			if fv.Kind() == reflect.Pointer {
				if fv.IsNil() {
					continue
				}
				fv = fv.Elem()
			}
			if fv.Kind() == reflect.Struct {
				subPaths := pathsNestedUnderPrefix(includedPaths, jsonName+".")
				if err := validateExpandableOnStruct(fv, subPaths); err != nil {
					errs = append(errs, fmt.Sprintf("%s.%s", jsonName, err))
				}
			}
			continue
		}

		fv := rv.Field(i)
		switch fv.Kind() {
		case reflect.Pointer:
			if fv.IsNil() || fv.Elem().Kind() != reflect.Struct {
				continue
			}
			inner := fv.Elem()
			if fieldErrs := validateIncludedExpandableStub(inner); len(fieldErrs) > 0 {
				for _, e := range fieldErrs {
					errs = append(errs, fmt.Sprintf("%s.%s", jsonName, e))
				}
			}
			errs = append(errs, validateExpandableListDataElements(inner, jsonName)...)
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
					if fieldErrs := validateIncludedExpandableStub(elem); len(fieldErrs) > 0 {
						for _, e := range fieldErrs {
							errs = append(errs, fmt.Sprintf("%s[%d].%s", jsonName, j, e))
						}
					}
				}
			}
		}
	}

	// Descend through non-expandable structs when includes target nested paths
	// (e.g. flat_rate.unit → Quantity.unit).
	for i := 0; i < rt.NumField(); i++ {
		ft := rt.Field(i)
		if !ft.IsExported() || ft.Tag.Get("expandable") == "true" {
			continue
		}
		jsonName := jsonFieldName(ft)
		if jsonName == "" {
			continue
		}
		subPaths := pathsNestedUnderPrefix(includedPaths, jsonName+".")
		if len(subPaths) == 0 {
			continue
		}
		fv := rv.Field(i)
		if fv.Kind() == reflect.Pointer {
			if fv.IsNil() {
				continue
			}
			fv = fv.Elem()
		}
		if fv.Kind() != reflect.Struct {
			continue
		}
		if err := validateExpandableOnStruct(fv, subPaths); err != nil {
			errs = append(errs, fmt.Sprintf("%s.%s", jsonName, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("expandable field validation failed: %s", strings.Join(errs, "; "))
	}
	return nil
}

var timeType = reflect.TypeOf((*time.Time)(nil)).Elem()

// validateIncludedExpandableStub enforces validate:"required" on exported
// string-kind and time.Time fields (aligned with resourcetest.validateStubFields).
func validateIncludedExpandableStub(rv reflect.Value) []string {
	rt := rv.Type()
	var errs []string

	for i := 0; i < rt.NumField(); i++ {
		fv := rv.Field(i)
		ft := rt.Field(i)
		if !ft.IsExported() {
			continue
		}
		validateTag := ft.Tag.Get("validate")
		if !strings.Contains(validateTag, "required") {
			continue
		}
		jsonName := jsonFieldName(ft)
		if jsonName == "" {
			continue
		}

		switch {
		case ft.Type == timeType:
			if fv.IsZero() {
				errs = append(errs, fmt.Sprintf("%s (json:%q) is zero time", ft.Name, jsonName))
			}
		case fv.Kind() == reflect.String:
			if fv.IsZero() {
				errs = append(errs, fmt.Sprintf("%s (json:%q) is empty", ft.Name, jsonName))
			}
		}
	}
	return errs
}

// validateExpandableListDataElements validates each element of List[T].Data when
// the expandable field points at a list wrapper struct.
func validateExpandableListDataElements(listStruct reflect.Value, jsonField string) []string {
	rt := listStruct.Type()
	dataField, ok := rt.FieldByName("Data")
	if !ok || !dataField.IsExported() {
		return nil
	}
	fv := listStruct.FieldByName("Data")
	if fv.Kind() != reflect.Slice {
		return nil
	}
	var errs []string
	for j := 0; j < fv.Len(); j++ {
		elem := fv.Index(j)
		if elem.Kind() == reflect.Pointer {
			if elem.IsNil() {
				continue
			}
			elem = elem.Elem()
		}
		if elem.Kind() != reflect.Struct {
			continue
		}
		for _, e := range validateIncludedExpandableStub(elem) {
			errs = append(errs, fmt.Sprintf("%s.data[%d].%s", jsonField, j, e))
		}
	}
	return errs
}
