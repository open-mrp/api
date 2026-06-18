package field

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
)

// ExplicitNullField scans body for the first Optional[T] struct field whose JSON key is present and explicitly null, returning that field's JSON name. It lets a caller that holds the raw request body turn the opaque ErrExplicitNull (which has no field context, since UnmarshalJSON cannot know its own key) into a parameter-specific error. The struct shape is read from v's type; v need not be populated. Returns false when no such field is found (e.g. the null is on a nested non-embedded struct), so callers should fall back to a generic message.
func ExplicitNullField(body []byte, v any) (string, bool) {
	body = bytes.TrimSpace(body)
	if len(body) == 0 || body[0] != '{' {
		return "", false
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return "", false
	}
	rt := reflect.TypeOf(v)
	for rt != nil && rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}
	if rt == nil || rt.Kind() != reflect.Struct {
		return "", false
	}
	return explicitNullFieldInStruct(rt, raw)
}

func explicitNullFieldInStruct(rt reflect.Type, raw map[string]json.RawMessage) (string, bool) {
	for sf := range rt.Fields() {
		if sf.PkgPath != "" {
			continue
		}
		if sf.Anonymous {
			et := sf.Type
			if et.Kind() == reflect.Pointer {
				et = et.Elem()
			}
			if et.Kind() == reflect.Struct {
				if name, ok := explicitNullFieldInStruct(et, raw); ok {
					return name, true
				}
			}
			continue
		}
		if !IsOptionalType(sf.Type) {
			continue
		}
		jsonName := jsonFieldName(sf.Tag.Get("json"))
		if jsonName == "" || jsonName == "-" {
			continue
		}
		rm, ok := raw[jsonName]
		if !ok {
			continue
		}
		if string(bytes.TrimSpace(rm)) == "null" {
			return jsonName, true
		}
	}
	return "", false
}

func jsonFieldName(tag string) string {
	if tag == "" || tag == "-" {
		return ""
	}
	name, _, _ := strings.Cut(tag, ",")
	return name
}
