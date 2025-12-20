package apiexample

import (
	"reflect"
	"strings"

	"github.com/augno/api/shared/validate"
)

func ValidateAndMarshalToMap(example any) map[string]any {
	// Validate the example first
	if err := validate.Validate(example); err != nil {
		// Return empty map if validation fails
		return make(map[string]any)
	}

	// Convert to map using JSON field names
	result := make(map[string]any)

	val := reflect.ValueOf(example)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return result
	}

	t := val.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Only include fields with json tags
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Parse json tag to get field name
		jsonName := strings.Split(jsonTag, ",")[0]
		if jsonName == "" {
			jsonName = field.Name
		}

		fieldValue := val.Field(i)
		if !fieldValue.CanInterface() {
			continue
		}

		// Skip nil pointers to avoid "null" in OpenAPI examples which vacuum dislikes for non-nullable types
		if fieldValue.Kind() == reflect.Ptr && fieldValue.IsNil() {
			continue
		}

		result[jsonName] = fieldValue.Interface()
	}

	return result
}
