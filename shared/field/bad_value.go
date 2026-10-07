package field

import (
	"encoding"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
)

var (
	jsonUnmarshalerType = reflect.TypeFor[json.Unmarshaler]()
	textUnmarshalerType = reflect.TypeFor[encoding.TextUnmarshaler]()
)

// BadValuePath returns the JSON path (e.g. "lines[0].quantity") of the first value in body that
// does not decode into its field of v's type, with the error decoding that value on its own. An
// error raised inside an UnmarshalJSON method (Optional, Clearable, time.Time) reaches the caller
// with no field, so a caller holding the body uses this to name it. It returns ("", nil) when no
// single field is at fault, such as a body that is not an object.
func BadValuePath(body []byte, v any) (string, error) {
	rt := reflect.TypeOf(v)
	if rt == nil {
		return "", nil
	}
	return badValue(body, rt, "")
}

func badValue(raw []byte, rt reflect.Type, path string) (string, error) {
	err := json.Unmarshal(raw, reflect.New(rt).Interface())
	if err == nil {
		return "", nil
	}
	if p, childErr := badChild(raw, rt, path); childErr != nil {
		return p, childErr
	}
	if path == "" {
		return "", nil
	}
	return path, err
}

// badChild narrows a value that failed to decode to the member at fault, if it has members.
func badChild(raw []byte, rt reflect.Type, path string) (string, error) {
	for rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}
	if w, ok := reflect.Zero(rt).Interface().(interface{ OpenAPIInnerType() reflect.Type }); ok {
		return badValue(raw, w.OpenAPIInnerType(), path)
	}
	if reflect.PointerTo(rt).Implements(jsonUnmarshalerType) || reflect.PointerTo(rt).Implements(textUnmarshalerType) {
		return "", nil
	}

	switch rt.Kind() {
	case reflect.Struct:
		var obj map[string]json.RawMessage
		if json.Unmarshal(raw, &obj) != nil {
			return "", nil
		}
		return badField(obj, rt, path)

	case reflect.Slice, reflect.Array:
		if rt.Elem().Kind() == reflect.Uint8 {
			return "", nil
		}
		var items []json.RawMessage
		if json.Unmarshal(raw, &items) != nil {
			return "", nil
		}
		for i, item := range items {
			if p, err := badValue(item, rt.Elem(), fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return p, err
			}
		}

	case reflect.Map:
		var obj map[string]json.RawMessage
		if json.Unmarshal(raw, &obj) != nil {
			return "", nil
		}
		for _, k := range slices.Sorted(maps.Keys(obj)) {
			if p, err := badValue(obj[k], rt.Elem(), joinPath(path, k)); err != nil {
				return p, err
			}
		}
	}
	return "", nil
}

// badField checks rt's fields in declaration order, flattening embedded structs as encoding/json does.
func badField(obj map[string]json.RawMessage, rt reflect.Type, path string) (string, error) {
	for sf := range rt.Fields() {
		tag := sf.Tag.Get("json")
		if tag == "-" {
			continue
		}
		name := jsonFieldName(tag)
		if sf.Anonymous && name == "" {
			et := sf.Type
			if et.Kind() == reflect.Pointer {
				et = et.Elem()
			}
			if et.Kind() == reflect.Struct {
				if p, err := badField(obj, et, path); err != nil {
					return p, err
				}
				continue
			}
		}
		if !sf.IsExported() {
			continue
		}
		if name == "" {
			name = sf.Name
		}
		raw, ok := lookupJSONKey(obj, name)
		if !ok {
			continue
		}
		if p, err := badValue(raw, sf.Type, joinPath(path, name)); err != nil {
			return p, err
		}
	}
	return "", nil
}

// lookupJSONKey matches keys the way encoding/json does: exactly, else ignoring case.
func lookupJSONKey(obj map[string]json.RawMessage, name string) (json.RawMessage, bool) {
	if raw, ok := obj[name]; ok {
		return raw, true
	}
	for k, raw := range obj {
		if strings.EqualFold(k, name) {
			return raw, true
		}
	}
	return nil, false
}

func joinPath(path, name string) string {
	if path == "" {
		return name
	}
	return path + "." + name
}
