package patch

import (
	"reflect"

	"github.com/go-playground/validator/v10"
)

// RegisterValidator teaches go-playground/validator how to validate struct tags on
// patch.Field[T] (and *patch.Field[T]) and patch.Nullable[T]: unset (and clear for Field)
// are treated as empty for validate:"omitempty"; when set, the inner value is validated.
func RegisterValidator(v *validator.Validate) {
	for _, typ := range []any{
		Field[string]{}, (*Field[string])(nil),
		Field[[]string]{}, (*Field[[]string])(nil),
		Nullable[string]{},
		Nullable[[]string]{},
	} {
		v.RegisterCustomTypeFunc(patchWrappedCustomType, typ)
	}
}

func patchWrappedCustomType(field reflect.Value) any {
	if field.Kind() == reflect.Pointer {
		if field.IsNil() {
			return nil
		}
		field = field.Elem()
	}
	if !IsFieldType(field.Type()) && !IsNullableType(field.Type()) {
		return field.Interface()
	}
	if field.CanAddr() {
		if IsFieldType(field.Type()) {
			if isClear := field.Addr().MethodByName("IsClear"); isClear.IsValid() {
				out := isClear.Call(nil)
				if len(out) == 1 && out[0].Kind() == reflect.Bool && out[0].Bool() {
					return nil
				}
			}
		}
		if isSet := field.Addr().MethodByName("IsSet"); isSet.IsValid() {
			out := isSet.Call(nil)
			if len(out) == 1 && out[0].Kind() == reflect.Bool && !out[0].Bool() {
				return nil
			}
		}
		if val := field.Addr().MethodByName("Value"); val.IsValid() {
			out := val.Call(nil)
			if len(out) == 2 && out[1].Kind() == reflect.Bool && out[1].Bool() {
				return out[0].Interface()
			}
		}
	}
	return nil
}
