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
	metadata := getFieldMetadata(fieldErr, structValue)
	fieldName := metadata.name
	source := formatSource(metadata.source)

	switch fieldErr.Tag() {
	case "required":
		return fmt.Sprintf("%s '%s' is required.", source, fieldName)
	case "min":
		return formatMinMaxError(fieldName, source, fieldErr, "at least")
	case "max":
		return formatMinMaxError(fieldName, source, fieldErr, "at most")
	case "email":
		return fmt.Sprintf("%s '%s' must be a valid email address.", source, fieldName)
	case "len":
		return fmt.Sprintf("%s '%s' must be exactly %s characters long.", source, fieldName, fieldErr.Param())
	case "gte":
		return formatGteLteError(fieldName, source, fieldErr, "greater than or equal to")
	case "lte":
		return formatGteLteError(fieldName, source, fieldErr, "less than or equal to")
	case "gt":
		return fmt.Sprintf("%s '%s' must be greater than %s.", source, fieldName, fieldErr.Param())
	case "lt":
		return fmt.Sprintf("%s '%s' must be less than %s.", source, fieldName, fieldErr.Param())
	case "oneof":
		return fmt.Sprintf("%s '%s' must be one of: %s.", source, fieldName, fieldErr.Param())
	case "omitempty":
		// This shouldn't happen as omitempty means "skip validation if empty"
		return fmt.Sprintf("%s '%s' validation failed.", source, fieldName)
	default:
		return fmt.Sprintf("%s '%s' is invalid (%s).", source, fieldName, fieldErr.Tag())
	}
}

func formatMinMaxError(fieldName, source string, fieldErr validator.FieldError, comparison string) string {
	param := fieldErr.Param()
	fieldType := fieldErr.Type()

	if fieldType.Kind() == reflect.Slice || fieldType.Kind() == reflect.Array {
		itemWord := getItemWord(param)
		return fmt.Sprintf("%s '%s' must have %s %s %s.", source, fieldName, comparison, param, itemWord)
	}

	if fieldType.Kind() == reflect.String {
		return fmt.Sprintf("%s '%s' must be %s %s characters long.", source, fieldName, comparison, param)
	}

	return fmt.Sprintf("%s '%s' must be %s %s.", source, fieldName, comparison, param)
}

func formatGteLteError(fieldName, source string, fieldErr validator.FieldError, comparison string) string {
	param := fieldErr.Param()
	fieldType := fieldErr.Type()

	if fieldType.Kind() == reflect.Slice || fieldType.Kind() == reflect.Array {
		itemWord := getItemWord(param)
		return fmt.Sprintf("%s '%s' must have %s %s %s.", source, fieldName, comparison, param, itemWord)
	}

	if fieldType.Kind() == reflect.String {
		return fmt.Sprintf("%s '%s' must be %s %s characters long.", source, fieldName, comparison, param)
	}

	return fmt.Sprintf("%s '%s' must be %s %s.", source, fieldName, comparison, param)
}

func getItemWord(param string) string {
	if value, err := strconv.ParseFloat(param, 64); err == nil {
		if value == 1.0 {
			return "item"
		}
	}
	return "items"
}

type fieldMetadata struct {
	name   string
	source string
}

func getFieldMetadata(fieldErr validator.FieldError, structValue any) fieldMetadata {
	fieldName := fieldErr.Field()
	if structValue == nil {
		return fieldMetadata{name: fieldName, source: "field"}
	}

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
					return fieldMetadata{name: jsonName, source: "field"}
				}
			}

			tagPriority := []string{"form", "query", "path", "header", "cookie"}
			for _, tagName := range tagPriority {
				if tagValue := field.Tag.Get(tagName); tagValue != "" {
					tagNameValue := strings.Split(tagValue, ",")[0]
					if tagNameValue != "" && tagNameValue != "-" {
						return fieldMetadata{name: tagNameValue, source: tagName}
					}
				}
			}

			return fieldMetadata{name: fieldName, source: "field"}
		}
	}

	return fieldMetadata{name: fieldName, source: "field"}
}

func formatSource(source string) string {
	switch source {
	case "header":
		return "Header"
	case "query":
		return "Query parameter"
	case "path":
		return "Path parameter"
	case "form":
		return "Form field"
	case "cookie":
		return "Cookie"
	default:
		return "Field"
	}
}

func getFieldName(fieldErr validator.FieldError, structValue any) string {
	return getFieldMetadata(fieldErr, structValue).name
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
