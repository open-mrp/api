package validate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"

	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// RejectExplicitJSONNulls returns an invalid_format API error when the JSON body
// contains an explicit null or a blank string for optional pointer fields
// (json omitempty) or a blank string for field.Optional fields. Absent keys are
// allowed (PATCH semantics).
//
// field.Clearable values accept null (clear) and are not checked here.
// field.Optional values reject explicit null at unmarshal time; this pass only
// rejects a present-but-blank string for them. Response-style pointers without
// omitempty are not checked here.
func RejectExplicitJSONNulls(body []byte, v any) *apierror.APIError {
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
	return rejectExplicitNullsInStruct(rv, rv.Type(), raw)
}

func rejectExplicitNullsInStruct(rv reflect.Value, rt reflect.Type, raw map[string]json.RawMessage) *apierror.APIError {
	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i)
		if sf.PkgPath != "" {
			continue
		}

		if sf.Anonymous {
			switch {
			case sf.Type.Kind() == reflect.Struct:
				if apiErr := rejectExplicitNullsInStruct(rv.Field(i), sf.Type, raw); apiErr != nil {
					return apiErr
				}
			case sf.Type.Kind() == reflect.Pointer && sf.Type.Elem().Kind() == reflect.Struct:
				fv := rv.Field(i)
				if fv.IsNil() {
					continue
				}
				if apiErr := rejectExplicitNullsInStruct(fv.Elem(), sf.Type.Elem(), raw); apiErr != nil {
					return apiErr
				}
			}
			continue
		}

		if field.IsClearableType(sf.Type) {
			// Clearable accepts null (clear) by design.
			continue
		}
		if field.IsOptionalType(sf.Type) {
			// Optional rejects an explicit null at unmarshal time; here we additionally
			// reject a present-but-blank string so an empty value is a 400 rather than
			// silently set to "". Non-string Optionals never carry a blank string here
			// (it would have failed to unmarshal into the inner type).
			if apiErr := rejectBlankString(sf, raw); apiErr != nil {
				return apiErr
			}
			continue
		}

		if sf.Type.Kind() != reflect.Pointer || !jsonTagHasOmitempty(sf.Tag.Get("json")) {
			continue
		}

		jsonName := jsonFieldNameFromTag(sf.Tag.Get("json"))
		if jsonName == "" || jsonName == "-" {
			continue
		}

		rm, ok := raw[jsonName]
		if !ok {
			continue
		}
		var parsed any
		if err := json.Unmarshal(rm, &parsed); err != nil {
			continue
		}
		if parsed == nil {
			return apierror.NewInvalidFormatError(fmt.Sprintf("Field '%s' cannot be null.", jsonName), jsonName)
		}
		if sf.Type.Elem().Kind() == reflect.String {
			if str, ok := parsed.(string); ok && strings.TrimSpace(str) == "" {
				return apierror.NewInvalidFormatError(fmt.Sprintf("Field '%s' must not be blank.", jsonName), jsonName)
			}
		}
	}
	return nil
}

// ApplySlicePresenceFlags sets boolean "Has" companion fields to true when the
// corresponding slice field's JSON key is present in the raw body. This lets
// downstream code distinguish "field absent" (Has=false) from "field explicitly
// sent" (Has=true), including empty arrays to clear the collection.
//
// Convention: a slice field `FooIDs []string` with json tag "foo_ids" has a
// companion `HasFooIDs bool` with json:"-". When "foo_ids" appears in the JSON
// body, HasFooIDs is set to true.
func ApplySlicePresenceFlags(body []byte, v any) {
	body = bytes.TrimSpace(body)
	if len(body) == 0 || body[0] != '{' {
		return
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return
	}
	rv := reflect.ValueOf(v)
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
	for sf := range rt.Fields() {
		if sf.PkgPath != "" || sf.Type.Kind() != reflect.Slice {
			continue
		}

		jsonName := jsonFieldNameFromTag(sf.Tag.Get("json"))
		if jsonName == "" || jsonName == "-" {
			continue
		}

		if _, ok := raw[jsonName]; !ok {
			continue
		}

		hasFld, ok := rt.FieldByName("Has" + sf.Name)
		if !ok || hasFld.Type.Kind() != reflect.Bool {
			continue
		}
		rv.FieldByName("Has" + sf.Name).SetBool(true)
	}
}

// rejectBlankString returns an invalid_format error when sf's JSON key is present
// and its value is a blank string. Absent keys and non-string values yield nil.
func rejectBlankString(sf reflect.StructField, raw map[string]json.RawMessage) *apierror.APIError {
	jsonName := jsonFieldNameFromTag(sf.Tag.Get("json"))
	if jsonName == "" || jsonName == "-" {
		return nil
	}
	rm, ok := raw[jsonName]
	if !ok {
		return nil
	}
	var parsed any
	if err := json.Unmarshal(rm, &parsed); err != nil {
		return nil
	}
	if str, ok := parsed.(string); ok && strings.TrimSpace(str) == "" {
		return apierror.NewInvalidFormatError(fmt.Sprintf("Field '%s' must not be blank.", jsonName), jsonName)
	}
	return nil
}

func jsonTagHasOmitempty(tag string) bool {
	if tag == "" || tag == "-" {
		return false
	}
	return slices.Contains(strings.Split(tag, ",")[1:], "omitempty")
}

func jsonFieldNameFromTag(tag string) string {
	if tag == "" || tag == "-" {
		return ""
	}
	before, _, ok := strings.Cut(tag, ",")
	if ok {
		return before
	}
	return tag
}
