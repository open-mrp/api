package audit

import (
	"encoding/json"
	"reflect"
	"strings"
)

// ComputeChanges compares two values (typically two versions of the same
// struct) and returns field-level before/after changes.
//
// Only struct fields with a non-empty `audit` struct tag are considered.
// When fields is empty, every exported field that has an `audit` tag is
// compared. When fields is non-empty, each name must be a Go struct field
// name that also has an `audit` tag; untagged fields are skipped.
func ComputeChanges(old, new any, fields ...string) []FieldChange {
	oldVal := reflect.ValueOf(old)
	newVal := reflect.ValueOf(new)

	if oldVal.IsValid() && oldVal.Kind() == reflect.Pointer {
		if oldVal.IsNil() {
			oldVal = reflect.Value{}
		} else {
			oldVal = oldVal.Elem()
		}
	}
	if newVal.IsValid() && newVal.Kind() == reflect.Pointer {
		if newVal.IsNil() {
			newVal = reflect.Value{}
		} else {
			newVal = newVal.Elem()
		}
	}

	var oldType reflect.Type
	if oldVal.IsValid() {
		oldType = oldVal.Type()
	} else {
		oldType = reflect.TypeOf(new)
	}
	if oldType != nil && oldType.Kind() == reflect.Pointer {
		oldType = oldType.Elem()
	}

	if len(fields) == 0 && oldType != nil && oldType.Kind() == reflect.Struct {
		for sf := range oldType.Fields() {
			if sf.PkgPath != "" { // unexported
				continue
			}
			if strings.TrimSpace(sf.Tag.Get("audit")) == "" {
				continue
			}
			fields = append(fields, sf.Name)
		}
	}

	changes := make([]FieldChange, 0, len(fields))
	for _, fieldName := range fields {
		fieldKey, ok := auditFieldKeyForStructField(oldType, fieldName)
		if !ok {
			continue
		}

		oldField, newField := fieldValues(oldVal, newVal, fieldName)
		if oldField == nil && newField == nil {
			continue
		}

		if jsonEqual(oldField, newField) {
			continue
		}

		oldJSON := marshalToRawJSON(oldField)
		newJSON := marshalToRawJSON(newField)

		changes = append(changes, FieldChange{
			Field:    fieldKey,
			OldValue: oldJSON,
			NewValue: newJSON,
		})
	}

	return changes
}

// auditFieldKeyForStructField returns the audit payload key for a struct field
// if the field exists and has a non-empty `audit` tag.
func auditFieldKeyForStructField(structType reflect.Type, fieldName string) (string, bool) {
	if structType == nil {
		return "", false
	}
	if structType.Kind() == reflect.Pointer {
		structType = structType.Elem()
	}
	if structType.Kind() != reflect.Struct {
		return "", false
	}

	sf, ok := structType.FieldByName(fieldName)
	if !ok {
		return "", false
	}

	tag := strings.TrimSpace(sf.Tag.Get("audit"))
	if tag == "" {
		return "", false
	}
	return strings.Split(tag, ",")[0], true
}

func fieldValues(oldVal, newVal reflect.Value, fieldName string) (any, any) {
	var oldField, newField any

	if oldVal.IsValid() && oldVal.Kind() == reflect.Struct {
		f := oldVal.FieldByName(fieldName)
		if f.IsValid() && f.CanInterface() {
			oldField = f.Interface()
		}
	}

	if newVal.IsValid() && newVal.Kind() == reflect.Struct {
		f := newVal.FieldByName(fieldName)
		if f.IsValid() && f.CanInterface() {
			newField = f.Interface()
		}
	}

	return oldField, newField
}

// jsonEqual compares two values. For json.RawMessage fields it performs a
// semantic comparison (unmarshal then DeepEqual) so that key ordering and
// whitespace differences do not produce false diffs.
func jsonEqual(a, b any) bool {
	aRaw, aIsJSON := a.(json.RawMessage)
	bRaw, bIsJSON := b.(json.RawMessage)
	if aIsJSON && bIsJSON {
		var av, bv any
		if err := json.Unmarshal(aRaw, &av); err != nil {
			return reflect.DeepEqual(a, b)
		}
		if err := json.Unmarshal(bRaw, &bv); err != nil {
			return reflect.DeepEqual(a, b)
		}
		return reflect.DeepEqual(av, bv)
	}
	return reflect.DeepEqual(a, b)
}

func marshalToRawJSON(v any) json.RawMessage {
	// Treat nil explicitly so callers get a deterministic "null".
	if v == nil {
		return json.RawMessage("null")
	}

	b, err := json.Marshal(v)
	if err != nil {
		// Fallback to a safe value to avoid publisher crashes. The audit event
		// will still be recorded, albeit with null values for this field.
		return json.RawMessage("null")
	}

	return json.RawMessage(b)
}
