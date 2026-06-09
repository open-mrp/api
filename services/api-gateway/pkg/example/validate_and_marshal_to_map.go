package apiexample

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

var jsonMarshalerType = reflect.TypeFor[json.Marshaler]()

// ValidateAndMarshalToMap marshals the JSON-body fields of a struct into a map
// whose shape matches the actual wire payload, so generated examples agree with
// the generated schema. It mirrors encoding/json with one deliberate exception:
// exported fields without a json tag (path/query/header-only request fields) are
// dropped so body examples stay payload-only.
//
// In particular, and unlike a naive "skip every empty value" marshaller:
//   - a non-omitempty pointer that is nil is emitted as JSON null — a nullable
//     "value or null" response field stays visible instead of vanishing;
//   - omitempty/omitzero fields are dropped only when empty, exactly like the wire;
//   - field.Optional[T]/field.Clearable[T] encode through their own MarshalJSON
//     (set value, or null for an explicit Clearable clear) and drop when unset.
func ValidateAndMarshalToMap(example any) map[string]any {
	out := make(map[string]any)
	appendJSONTaggedFields(out, reflect.ValueOf(example))
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
		parts := strings.Split(jsonTag, ",")
		name := parts[0]
		if name == "" {
			continue
		}
		if hasOmitempty(parts[1:]) && isEmptyForOmit(fv) {
			continue
		}
		enc, ok := encodeValue(fv)
		if !ok {
			continue
		}
		out[name] = enc
	}
}

func hasOmitempty(opts []string) bool {
	for _, o := range opts {
		// The generator treats omitempty and omitzero identically when deciding
		// whether a field is required/nullable, so the example must too.
		if o == "omitempty" || o == "omitzero" {
			return true
		}
	}
	return false
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

// encodeValue renders a field's value as a schema-consistent JSON example. It
// follows the wire form except where the wire would contradict the generated
// schema, which keys nullability off the Go type:
//
//   - a nil pointer becomes null — pointer fields (without omitempty) are the
//     only nullable fields, so null is the correct "value or null" example;
//   - a nil slice/map becomes [] / {} rather than null — composites are never
//     nullable in the schema (type: array/object), so an empty value, not null,
//     is the schema-valid example;
//   - field.Optional/Clearable, time.Time, and other json.Marshaler types encode
//     through their own MarshalJSON (set value, or null for a Clearable clear).
//
// The bool is false only when the value cannot be marshaled (e.g. an unset patch
// field with no omitzero tag), so the caller drops the field instead of emitting
// a broken example.
func encodeValue(v reflect.Value) (any, bool) {
	// Types with custom JSON (field.Optional/Clearable, time.Time, enums) encode
	// to their exact wire form — that is the authoritative example.
	if v.Type().Implements(jsonMarshalerType) {
		return marshalViaJSON(v)
	}

	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return nil, true // JSON null — nullable response field
		}
		return encodeValue(v.Elem())
	case reflect.Struct:
		nested := make(map[string]any)
		appendJSONTaggedFields(nested, v)
		return nested, true
	case reflect.Slice, reflect.Array:
		out := make([]any, 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			if ev, ok := encodeValue(v.Index(i)); ok {
				out = append(out, ev)
			}
		}
		return out, true
	case reflect.Map:
		out := make(map[string]any, v.Len())
		for _, k := range v.MapKeys() {
			if ev, ok := encodeValue(v.MapIndex(k)); ok {
				out[fmt.Sprint(k.Interface())] = ev
			}
		}
		return out, true
	default:
		return marshalViaJSON(v)
	}
}

func marshalViaJSON(v reflect.Value) (any, bool) {
	if !v.IsValid() {
		return nil, true
	}
	b, err := json.Marshal(v.Interface())
	if err != nil {
		return nil, false
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, false
	}
	return out, true
}

// isEmptyForOmit reports whether a value is "empty" for omitempty/omitzero,
// matching encoding/json's emptiness rule and honoring an IsZero method
// (field.Optional/Clearable and time.Time) for omitzero.
func isEmptyForOmit(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		return v.IsNil()
	}
	if z, ok := isZeroViaMethod(v); ok {
		return z
	}
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	}
	return false
}

func isZeroViaMethod(v reflect.Value) (zero bool, ok bool) {
	m := v.MethodByName("IsZero")
	if !m.IsValid() {
		return false, false
	}
	mt := m.Type()
	if mt.NumIn() != 0 || mt.NumOut() != 1 || mt.Out(0).Kind() != reflect.Bool {
		return false, false
	}
	return m.Call(nil)[0].Bool(), true
}
