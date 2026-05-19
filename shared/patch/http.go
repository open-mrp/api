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

func jsonFieldName(tag string) string {
	if tag == "" || tag == "-" {
		return ""
	}
	name, _, _ := strings.Cut(tag, ",")
	return name
}
