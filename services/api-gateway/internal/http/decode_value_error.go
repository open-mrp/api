package httptransport

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// valueDecodeError turns an error decoding a value of body into an invalid_format error naming the
// field. The decoder leaves the field off an error raised inside an UnmarshalJSON method
// (field.Optional, field.Clearable, time.Time), so the field is found by re-reading body.
func valueDecodeError(body []byte, dst any, err error) error {
	path, cause := field.BadValuePath(body, dst)
	if cause == nil {
		path, cause = "", err
		if uterr, ok := errors.AsType[*json.UnmarshalTypeError](err); ok {
			path = uterr.Field
		}
	}

	if uterr, ok := errors.AsType[*json.UnmarshalTypeError](cause); ok {
		if path == "" {
			return apierror.NewInvalidFormatError(fmt.Sprintf("Invalid request body: expected %s, got %s.", jsonTypeName(uterr.Type), uterr.Value), "")
		}
		return apierror.NewInvalidFormatError(fmt.Sprintf("Invalid type for field '%s': expected %s, got %s.", path, jsonTypeName(uterr.Type), uterr.Value), path)
	}
	if path == "" {
		return err
	}
	if _, ok := errors.AsType[*time.ParseError](cause); ok {
		return apierror.NewInvalidFormatError(fmt.Sprintf("Invalid format for field '%s': expected an RFC 3339 timestamp, such as 2024-01-15T10:30:00Z.", path), path)
	}
	if errors.Is(cause, field.ErrExplicitNull) {
		return apierror.NewInvalidFormatError(fmt.Sprintf("Field '%s' cannot be null.", path), path)
	}
	return apierror.NewInvalidFormatError(fmt.Sprintf("Invalid value for field '%s'.", path), path)
}

var timeType = reflect.TypeFor[time.Time]()

// jsonTypeName names the JSON a Go type accepts, so a type error does not leak Go type names.
func jsonTypeName(t reflect.Type) string {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == timeType {
		return "an RFC 3339 timestamp"
	}
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "boolean"
	case reflect.Int8, reflect.Int16, reflect.Int32:
		return fmt.Sprintf("%d-bit integer", t.Bits())
	case reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return fmt.Sprintf("%d-bit unsigned integer", t.Bits())
	case reflect.Int, reflect.Int64:
		return "integer"
	case reflect.Uint, reflect.Uint64:
		return "unsigned integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Slice, reflect.Array:
		if t.Elem().Kind() == reflect.Uint8 {
			return "base64 string"
		}
		return "array"
	case reflect.Struct, reflect.Map:
		return "object"
	default:
		return "JSON value"
	}
}
