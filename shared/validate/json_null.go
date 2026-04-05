package validate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	apierror "github.com/augno/api/shared/errors"
)

// RejectExplicitJSONNulls returns an invalid_format API error when the JSON body
// contains an explicit null for any pointer field tagged nullable:"false".
// Absent keys are allowed (PATCH semantics). It only inspects top-level object keys;
// nested objects are not walked.
//
// Tag nullable:"false" is shared with the OpenAPI generator; keep it in sync.
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
	if rv.Kind() == reflect.Ptr {
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
			case sf.Type.Kind() == reflect.Ptr && sf.Type.Elem().Kind() == reflect.Struct:
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

		if strings.ToLower(strings.TrimSpace(sf.Tag.Get("nullable"))) != "false" {
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
	}
	return nil
}

// ApplyExplicitNulls sets nullable:"true" *string fields to ptr("") when the
// JSON body contains an explicit null for that field. This lets downstream code
// distinguish "field absent" (nil) from "field explicitly null" (ptr("")).
// It must be called after JSON decoding and before RejectExplicitJSONNulls.
func ApplyExplicitNulls(body []byte, v any) {
	body = bytes.TrimSpace(body)
	if len(body) == 0 || body[0] != '{' {
		return
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return
	}
	applyExplicitNullsInStruct(rv, rv.Type(), raw)
}

func applyExplicitNullsInStruct(rv reflect.Value, rt reflect.Type, raw map[string]json.RawMessage) {
	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i)
		if sf.PkgPath != "" {
			continue
		}

		if sf.Anonymous {
			switch {
			case sf.Type.Kind() == reflect.Struct:
				applyExplicitNullsInStruct(rv.Field(i), sf.Type, raw)
			case sf.Type.Kind() == reflect.Ptr && sf.Type.Elem().Kind() == reflect.Struct:
				fv := rv.Field(i)
				if fv.IsNil() {
					continue
				}
				applyExplicitNullsInStruct(fv.Elem(), sf.Type.Elem(), raw)
			}
			continue
		}

		if strings.ToLower(strings.TrimSpace(sf.Tag.Get("nullable"))) != "true" {
			continue
		}

		// Only handle *string fields.
		if sf.Type.Kind() != reflect.Ptr || sf.Type.Elem().Kind() != reflect.String {
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
			empty := ""
			rv.Field(i).Set(reflect.ValueOf(&empty))
		}
	}
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
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return
	}
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i)
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

		// Look for a companion "Has" + field name boolean.
		hasFld, ok := rt.FieldByName("Has" + sf.Name)
		if !ok || hasFld.Type.Kind() != reflect.Bool {
			continue
		}
		rv.FieldByName("Has" + sf.Name).SetBool(true)
	}
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
