package field

import (
	"reflect"
	"time"

	"github.com/go-playground/validator/v10"
)

// RegisterValidator teaches go-playground/validator how to validate struct tags on field.Clearable[T] and field.Optional[T]: unset (and clear for Clearable) are treated as empty for validate:"omitempty"; when set, the inner value is validated.
//
// Every concrete inner type T that carries a comparison validator (min, max, gte, lte, len, …) MUST be registered here. Without registration the validator sees the wrapper struct instead of the inner value and panics (e.g. min on a bare field.Optional[int32]). String/slice inner types only need registration so their tags are honored; the scalar numeric/bool/time types below are required to avoid that panic. Inner types defined outside this package (constants.*, request inputs) cannot be referenced here without an import cycle — those fields must rely on omitempty/required only, or register themselves from their own package.
func RegisterValidator(v *validator.Validate) {
	for _, typ := range []any{
		Clearable[string]{},
		Clearable[[]string]{},
		Clearable[int]{},
		Clearable[int32]{},
		Clearable[int64]{},
		Clearable[float64]{},
		Clearable[bool]{},
		Clearable[time.Time]{},
		Clearable[map[string]*string]{},
		Optional[string]{},
		Optional[[]string]{},
		Optional[int]{},
		Optional[int32]{},
		Optional[int64]{},
		Optional[float64]{},
		Optional[bool]{},
		Optional[time.Time]{},
	} {
		v.RegisterCustomTypeFunc(patchWrappedCustomType, typ)
	}
}

// RegisterWrappedTypes does what RegisterValidator does for wrappers whose T lives outside this package, such as a
// request's nested input struct, so the tags on T's fields are enforced once the wrapper is set.
func RegisterWrappedTypes(v *validator.Validate, wrappers ...any) {
	v.RegisterCustomTypeFunc(patchWrappedCustomType, wrappers...)
}

// InnerType is the T of a field.Optional[T] or field.Clearable[T]; ok is false for any other type.
func InnerType(typ reflect.Type) (reflect.Type, bool) {
	if patchWrapperStructType(typ) == nil || typ.Kind() == reflect.Pointer {
		return nil, false
	}
	inner, ok := reflect.Zero(typ).Interface().(interface{ OpenAPIInnerType() reflect.Type })
	if !ok {
		return nil, false
	}
	return inner.OpenAPIInnerType(), true
}

func patchWrappedCustomType(field reflect.Value) any {
	if field.Kind() == reflect.Pointer {
		if field.IsNil() {
			return nil
		}
		field = field.Elem()
	}
	if !IsClearableType(field.Type()) && !IsOptionalType(field.Type()) {
		return field.Interface()
	}
	// A wrapper inside a struct that was itself unwrapped is not addressable; read it through a copy.
	if !field.CanAddr() {
		addressable := reflect.New(field.Type()).Elem()
		addressable.Set(field)
		field = addressable
	}
	if field.CanAddr() {
		if IsClearableType(field.Type()) {
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
