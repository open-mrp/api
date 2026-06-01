package patch

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
)

// ApplyPtrFieldNulls sets *patch.Field[T] struct fields to clear when the JSON body
// contains an explicit null for that key. encoding/json leaves such fields as nil
// pointers; this restores the clear state after Unmarshal.
func ApplyPtrFieldNulls(body []byte, v any) {
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
	applyPtrFieldNullsInStruct(rv, rv.Type(), raw)
}

func applyPtrFieldNullsInStruct(rv reflect.Value, rt reflect.Type, raw map[string]json.RawMessage) {
	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i)
		if sf.PkgPath != "" {
			continue
		}
		if sf.Anonymous {
			switch {
			case sf.Type.Kind() == reflect.Struct:
				applyPtrFieldNullsInStruct(rv.Field(i), sf.Type, raw)
			case sf.Type.Kind() == reflect.Pointer && sf.Type.Elem().Kind() == reflect.Struct:
				fv := rv.Field(i)
				if fv.IsNil() {
					continue
				}
				applyPtrFieldNullsInStruct(fv.Elem(), sf.Type.Elem(), raw)
			}
			continue
		}
		if sf.Type.Kind() != reflect.Pointer || !IsFieldType(sf.Type.Elem()) {
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
		if string(bytes.TrimSpace(rm)) != "null" {
			continue
		}
		fv := rv.Field(i)
		if !fv.IsNil() {
			continue
		}
		cleared := reflect.New(sf.Type.Elem()).Elem()
		clearMethod := cleared.Addr().MethodByName("UnmarshalJSON")
		if clearMethod.IsValid() {
			clearMethod.Call([]reflect.Value{reflect.ValueOf([]byte("null"))})
		}
		fv.Set(cleared.Addr())
	}
}

// ExplicitNullField scans body for the first Nullable[T] struct field whose JSON
// key is present and explicitly null, returning that field's JSON name. It lets a
// caller that holds the raw request body turn the opaque ErrExplicitNull (which
// has no field context, since UnmarshalJSON cannot know its own key) into a
// parameter-specific error. The struct shape is read from v's type; v need not be
// populated. Returns false when no such field is found (e.g. the null is on a
// nested non-embedded struct), so callers should fall back to a generic message.
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
		if !IsNullableType(sf.Type) {
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
