package validate

import (
	"bytes"
	"encoding/json"
	"reflect"

	apierror "github.com/open-mrp/api/shared/errors"
)

// RejectEmptyPatchBody returns a validation error when the JSON body does not contain at least one field that maps to a body-bound struct field. Fields bound from non-body sources (path, query, header tags) are excluded.
//
// This prevents PATCH requests with empty bodies ({}) from silently succeeding as no-op updates.
func RejectEmptyPatchBody(body []byte, v any) *apierror.APIError {
	body = bytes.TrimSpace(body)
	if len(body) == 0 || body[0] != '{' {
		return nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil
	}

	bodyFieldNames := collectBodyFieldNames(rv.Type())
	for key := range raw {
		if bodyFieldNames[key] {
			return nil
		}
	}

	return apierror.NewValidationError("Request body must contain at least one field to update.")
}

// collectBodyFieldNames returns the set of JSON field names that are bound from the request body (i.e. have a json tag but no path, query, or header tag).
func collectBodyFieldNames(rt reflect.Type) map[string]bool {
	names := make(map[string]bool)
	collectBodyFieldNamesFromType(rt, names)
	return names
}

func collectBodyFieldNamesFromType(rt reflect.Type, names map[string]bool) {
	for sf := range rt.Fields() {
		if sf.PkgPath != "" {
			continue
		}

		if sf.Anonymous {
			t := sf.Type
			if t.Kind() == reflect.Pointer {
				t = t.Elem()
			}
			if t.Kind() == reflect.Struct {
				collectBodyFieldNamesFromType(t, names)
			}
			continue
		}

		// Skip fields bound from non-body sources.
		if sf.Tag.Get("path") != "" || sf.Tag.Get("query") != "" || sf.Tag.Get("header") != "" {
			continue
		}

		jsonName := jsonFieldNameFromTag(sf.Tag.Get("json"))
		if jsonName == "" || jsonName == "-" {
			continue
		}

		names[jsonName] = true
	}
}
