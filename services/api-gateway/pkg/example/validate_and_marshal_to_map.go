package apiexample

import (
	"encoding/json"
	"reflect"
	"strings"
	"time"
)

// ValidateAndMarshalToMap marshals struct fields that participate in JSON bodies into a map.
// Fields without a json tag (including path/query/header-only fields) are omitted so path
// examples do not pick up empty strings and body examples stay payload-only.
func ValidateAndMarshalToMap(example any) map[string]any {
	v := reflect.ValueOf(example)
	out := make(map[string]any)
	appendJSONTaggedFields(out, v)
	return out
}

func appendJSONTaggedFields(out map[string]any, v reflect.Value) {
	v = derefValue(v)
	if v.Kind() != reflect.Struct {
		return
	}
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		sf := t.Field(i)
		if !sf.IsExported() {
			continue
		}
		jsonTag := sf.Tag.Get("json")
		fv := v.Field(i)
		if sf.Anonymous && jsonTag == "" {
			appendJSONTaggedFields(out, fv)
			continue
		}
		if jsonTag == "" || jsonTag == "-" {
			continue
		}
		name := strings.Split(jsonTag, ",")[0]
		if name == "" {
			continue
		}
		if enc := jsonExampleFromValue(fv); enc != nil {
			out[name] = enc
		}
	}
}

func derefValue(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return v
		}
		v = v.Elem()
	}
	return v
}

func jsonExampleFromValue(v reflect.Value) any {
	v = derefValue(v)
	if !v.IsValid() {
		return nil
	}
	switch v.Kind() {
	case reflect.Struct:
		if v.Type() == reflect.TypeFor[time.Time]() {
			tm := v.Interface().(time.Time)
			if tm.IsZero() {
				return nil
			}
			bs, err := json.Marshal(tm)
			if err != nil {
				return nil
			}
			var s string
			if err := json.Unmarshal(bs, &s); err != nil {
				return nil
			}
			return s
		}
		nested := make(map[string]any)
		appendJSONTaggedFields(nested, v)
		if len(nested) == 0 {
			return nil
		}
		return nested
	case reflect.Slice, reflect.Array:
		if v.Kind() == reflect.Slice && v.IsNil() {
			return nil
		}
		n := v.Len()
		if n == 0 {
			return []any{}
		}
		out := make([]any, n)
		for i := 0; i < n; i++ {
			out[i] = jsonExampleFromValue(v.Index(i))
		}
		return out
	case reflect.Map:
		if v.IsNil() {
			return nil
		}
		if v.Type().Key().Kind() != reflect.String {
			b, err := json.Marshal(v.Interface())
			if err != nil {
				return nil
			}
			var m map[string]any
			if err := json.Unmarshal(b, &m); err != nil {
				return nil
			}
			return m
		}
		out := make(map[string]any, v.Len())
		for _, key := range v.MapKeys() {
			out[key.String()] = jsonExampleFromValue(v.MapIndex(key))
		}
		return out
	case reflect.String:
		s := v.String()
		if s == "" {
			return nil
		}
		return s
	case reflect.Bool:
		return v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint()
	case reflect.Float32, reflect.Float64:
		return v.Float()
	default:
		return v.Interface()
	}
}
