package apiendpoint

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

// ValidateExpandableFields checks that non-nil expandable sub-resource fields
// included in the API response have their required string-typed fields populated.
// It returns an error describing all violations, or nil if valid.
//
// Only fields in the requested include set (plus defaults) are validated,
// since non-requested expandable fields are collapsed to null before reaching
// the client.
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
	if rv.Kind() == reflect.Ptr {
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
			if elem.Kind() == reflect.Ptr {
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

		// Get the JSON field name to check against included paths.
		jsonName := ft.Tag.Get("json")
		if jsonName == "" {
			continue
		}
		jsonName = strings.Split(jsonName, ",")[0]

		if !includedPaths[jsonName] {
			// Check for nested paths (e.g., "freight_preferences.carrier").
			// If this field is a parent of a nested include, we need to walk into it.
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
			// Walk into this struct to find nested expandable fields.
			fv := rv.Field(i)
			if fv.Kind() == reflect.Ptr {
				if fv.IsNil() {
					continue
				}
				fv = fv.Elem()
			}
			if fv.Kind() == reflect.Struct {
				// Build sub-paths relative to this field.
				subPaths := make(map[string]bool)
				prefix := jsonName + "."
				for p := range includedPaths {
					if strings.HasPrefix(p, prefix) {
						subPaths[strings.TrimPrefix(p, prefix)] = true
					}
				}
				if err := validateExpandableOnStruct(fv, subPaths); err != nil {
					errs = append(errs, fmt.Sprintf("%s.%s", jsonName, err))
				}
			}
			continue
		}

		fv := rv.Field(i)
		switch fv.Kind() {
		case reflect.Ptr:
			if fv.IsNil() || fv.Elem().Kind() != reflect.Struct {
				continue
			}
			if fieldErrs := validateStubStringFields(fv.Elem()); len(fieldErrs) > 0 {
				for _, e := range fieldErrs {
					errs = append(errs, fmt.Sprintf("%s.%s", jsonName, e))
				}
			}
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
					if fieldErrs := validateStubStringFields(elem); len(fieldErrs) > 0 {
						for _, e := range fieldErrs {
							errs = append(errs, fmt.Sprintf("%s[%d].%s", jsonName, j, e))
						}
					}
				}
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("expandable field validation failed: %s", strings.Join(errs, "; "))
	}
	return nil
}

// validateStubStringFields checks that required string-kind fields on a struct
// are non-zero. Returns a list of field-level error descriptions.
func validateStubStringFields(rv reflect.Value) []string {
	rt := rv.Type()
	timeType := reflect.TypeOf((*time.Time)(nil)).Elem()
	var errs []string

	for i := 0; i < rt.NumField(); i++ {
		fv := rv.Field(i)
		ft := rt.Field(i)
		if !ft.IsExported() {
			continue
		}
		if ft.Type == timeType || fv.Kind() == reflect.Bool ||
			fv.Kind() == reflect.Ptr || fv.Kind() == reflect.Slice ||
			fv.Kind() == reflect.Map {
			continue
		}
		if fv.Kind() != reflect.String {
			continue
		}
		if !strings.Contains(ft.Tag.Get("validate"), "required") {
			continue
		}
		if fv.IsZero() {
			jsonTag := ft.Tag.Get("json")
			if jsonTag != "" {
				jsonTag = strings.Split(jsonTag, ",")[0]
			} else {
				jsonTag = ft.Name
			}
			errs = append(errs, fmt.Sprintf("%s (json:%q) is empty", ft.Name, jsonTag))
		}
	}
	return errs
}
