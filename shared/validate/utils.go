package validate

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"

	contracts "github.com/augno/api/shared/contracts"
	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func Validate(v any) *contracts.APIError {
	err := validate.Struct(v)
	if err != nil {
		return parseValidationErrors(err, v)
	}
	return nil
}

func parseValidationErrors(err error, structValue any) *contracts.APIError {
	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return contracts.NewValidationError(err.Error())
	}

	if len(validationErrors) == 1 {
		fieldErr := validationErrors[0]
		return createFieldValidationError(fieldErr, structValue)
	}

	var fieldErrors []string
	for _, fieldErr := range validationErrors {
		fieldErrors = append(fieldErrors, formatFieldError(fieldErr, structValue))
	}

	var firstField string
	if len(validationErrors) > 0 {
		firstField = getFieldName(validationErrors[0], structValue)
	}

	return contracts.NewValidationErrorWithParam(fmt.Sprintf("Validation failed for the following fields: %s", strings.Join(fieldErrors, " ")), firstField)
}

func createFieldValidationError(fieldErr validator.FieldError, structValue any) *contracts.APIError {
	fieldName := getFieldName(fieldErr, structValue)
	message := formatFieldError(fieldErr, structValue)

	return contracts.NewValidationErrorWithParam(message, fieldName)
}

func formatFieldError(fieldErr validator.FieldError, structValue any) string {
	fieldName := getFieldName(fieldErr, structValue)

	switch fieldErr.Tag() {
	case "required":
		return fmt.Sprintf("Field '%s' is required.", fieldName)
	case "min":
		return formatMinMaxError(fieldName, fieldErr, "at least")
	case "max":
		return formatMinMaxError(fieldName, fieldErr, "at most")
	case "email":
		return fmt.Sprintf("Field '%s' must be a valid email address.", fieldName)
	case "len":
		return fmt.Sprintf("Field '%s' must be exactly %s characters long.", fieldName, fieldErr.Param())
	case "gte":
		return formatGteLteError(fieldName, fieldErr, "greater than or equal to")
	case "lte":
		return formatGteLteError(fieldName, fieldErr, "less than or equal to")
	case "gt":
		return fmt.Sprintf("Field '%s' must be greater than %s.", fieldName, fieldErr.Param())
	case "lt":
		return fmt.Sprintf("Field '%s' must be less than %s.", fieldName, fieldErr.Param())
	case "oneof":
		return fmt.Sprintf("Field '%s' must be one of: %s.", fieldName, fieldErr.Param())
	case "omitempty":
		// This shouldn't happen as omitempty means "skip validation if empty"
		return fmt.Sprintf("Field '%s' validation failed.", fieldName)
	default:
		return fmt.Sprintf("Field '%s' is invalid (%s).", fieldName, fieldErr.Tag())
	}
}

func formatMinMaxError(fieldName string, fieldErr validator.FieldError, comparison string) string {
	param := fieldErr.Param()
	fieldType := fieldErr.Type()

	if fieldType.Kind() == reflect.Slice || fieldType.Kind() == reflect.Array {
		itemWord := getItemWord(param)
		return fmt.Sprintf("Field '%s' must have %s %s %s.", fieldName, comparison, param, itemWord)
	}

	if fieldType.Kind() == reflect.String {
		return fmt.Sprintf("Field '%s' must be %s %s characters long.", fieldName, comparison, param)
	}

	return fmt.Sprintf("Field '%s' must be %s %s.", fieldName, comparison, param)
}

func formatGteLteError(fieldName string, fieldErr validator.FieldError, comparison string) string {
	param := fieldErr.Param()
	fieldType := fieldErr.Type()

	if fieldType.Kind() == reflect.Slice || fieldType.Kind() == reflect.Array {
		itemWord := getItemWord(param)
		return fmt.Sprintf("Field '%s' must have %s %s %s.", fieldName, comparison, param, itemWord)
	}

	if fieldType.Kind() == reflect.String {
		return fmt.Sprintf("Field '%s' must be %s %s characters long.", fieldName, comparison, param)
	}

	return fmt.Sprintf("Field '%s' must be %s %s.", fieldName, comparison, param)
}

func getItemWord(param string) string {
	if value, err := strconv.ParseFloat(param, 64); err == nil {
		if value == 1.0 {
			return "item"
		}
	}
	return "items"
}

func getFieldName(fieldErr validator.FieldError, structValue any) string {
	fieldName := fieldErr.Field()

	if structValue != nil {
		if tagName := getFieldTagFromReflection(fieldName, structValue); tagName != "" {
			return tagName
		}
	}

	return fieldName
}

func getFieldTagFromReflection(fieldName string, structValue any) string {
	rv := reflect.ValueOf(structValue)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	rt := rv.Type()

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		if field.Name == fieldName {
			if jsonTag := field.Tag.Get("json"); jsonTag != "" {
				jsonName := strings.Split(jsonTag, ",")[0]
				if jsonName != "" && jsonName != "-" {
					return jsonName
				}
			}

			tagPriority := []string{"form", "query", "path", "header"}
			for _, tagName := range tagPriority {
				if tagValue := field.Tag.Get(tagName); tagValue != "" {
					tagName := strings.Split(tagValue, ",")[0]
					if tagName != "" && tagName != "-" {
						return tagName
					}
				}
			}

			return fieldName
		}
	}

	return fieldName
}

func GetValidator() *validator.Validate {
	return validate
}

type Validator struct {
	Errors map[string]string
}

func New() *Validator {
	return &Validator{Errors: make(map[string]string)}
}

func (v *Validator) Valid() bool {
	return len(v.Errors) == 0
}

func (v *Validator) AddError(key, message string) {
	if _, exists := v.Errors[key]; !exists {
		v.Errors[key] = message
	}
}

func (v *Validator) Check(ok bool, key, message string) {
	if !ok {
		v.AddError(key, message)
	}
}

func PermittedValue[T comparable](value T, permittedValues ...T) bool {
	return slices.Contains(permittedValues, value)
}

func Matches(value string, rx *regexp.Regexp) bool {
	return rx.MatchString(value)
}

func Unique[T comparable](values []T) bool {
	uniqueValues := make(map[T]bool)

	for _, value := range values {
		uniqueValues[value] = true
	}

	return len(values) == len(uniqueValues)
}
