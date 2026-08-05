// Package validate provides request-level input validation for the platform. It wraps the go-playground/validator library with custom validation tags and human-readable error formatting that maps directly to the API's error response contract ([apierror.APIError] with a Param field).
//
// Seven custom validator tags are registered at init time:
//
//   - "password":          8–72 characters, at least one lowercase letter, one uppercase letter, one digit, and one special character.
//   - "username":          3–255 characters, alphanumeric (upper and lower), underscores, and hyphens only ([a-zA-Z0-9_-]).
//   - "identifier":        accepts either a valid email address or a username (3–50 characters, alphanumeric, underscores, and hyphens).
//   - "custom_email":      stricter email validation than the built-in "email" tag, enforcing RFC length limits, TLD format, and no consecutive dots.
//   - "nonzero_decimal":   the field, parsed as a decimal string, must not equal zero.
//   - "max_days_ahead=N":  a time.Time (or field.Optional[time.Time]) no more than N days in the future. Past/zero values pass.
//   - "multiple_of=N":     a numeric field (or field.Optional[float64]) that is a whole multiple of N (e.g. multiple_of=0.1). Zero/unset values pass.
//
// All custom tags treat empty/zero values as valid — combine with "required" when the field must be present.
//
// The package also provides a lightweight [Validator] helper for imperative checks that can't be expressed with struct tags (e.g. cross-field constraints).
package validate

import (
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
	"github.com/go-playground/validator/v10"
	"github.com/shopspring/decimal"
)

var (
	// hasLowercase matches any string containing at least one ASCII lowercase letter.
	hasLowercase = regexp.MustCompile(`[a-z]`)

	// hasUppercase matches any string containing at least one ASCII uppercase letter.
	hasUppercase = regexp.MustCompile(`[A-Z]`)

	// hasNumber matches any string containing at least one ASCII digit.
	hasNumber = regexp.MustCompile(`[0-9]`)

	// hasSpecialChar matches any string containing at least one of the special characters required by the "password" validation tag. Note: tilde (~) and backtick (`) are intentionally excluded.
	hasSpecialChar = regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`)

	// emailRX is the primary regex used by isValidEmail to validate the overall email format. It is applied after the structural checks (length limits, no consecutive dots, TLD format) have passed. The pattern follows RFC 5321 local-part rules and requires at least a two-character alphabetic TLD.
	emailRX = regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*\.[a-zA-Z]{2,}$`)
)

// validate is the package-level validator instance with custom tags registered.
var validate = validator.New()

func init() {
	field.RegisterValidator(validate)
	_ = validate.RegisterValidation("password", validatePassword)
	_ = validate.RegisterValidation("username", validateUsername)
	_ = validate.RegisterValidation("identifier", validateUsernameOrEmail)
	_ = validate.RegisterValidation("custom_email", validateCustomEmail)
	_ = validate.RegisterValidation("nonzero_decimal", validateNonzeroDecimal)
	_ = validate.RegisterValidation("decimal", validateDecimal)
	_ = validate.RegisterValidation("max_days_ahead", validateMaxDaysAhead)
	_ = validate.RegisterValidation("multiple_of", validateMultipleOf)
	// "enum" is enforced by reflection in httptransport.ValidateEnumFields, which runs before this validator and knows the allowed values from the field's type. It is registered as a no-op only so the tag cannot panic: go-playground panics on an unknown tag while building its struct cache, so a single `validate:"enum"` on a request struct would 500 that endpoint on every request. The tag is common on response structs, where this validator never sees it.
	_ = validate.RegisterValidation("enum", func(validator.FieldLevel) bool { return true })
}

// validatePassword implements the "password" struct tag. A valid password is 8–72 bytes long and contains at least one lowercase letter, one uppercase letter, one ASCII digit, and one special character (from the hasSpecialChar set). Empty strings pass (combine with "required" to enforce presence). The 72-byte upper bound matches bcrypt's maximum input length.
const PasswordMaxLength int = 72

func validatePassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()
	if password == "" {
		return true
	}
	if len(password) < 8 || len(password) > PasswordMaxLength {
		return false
	}
	if !hasLowercase.MatchString(password) || !hasUppercase.MatchString(password) {
		return false
	}
	if !hasNumber.MatchString(password) || !hasSpecialChar.MatchString(password) {
		return false
	}
	return true
}

// usernameOnlyRegex matches strings containing only ASCII alphanumeric characters, underscores, and hyphens. Used by validateUsername and validateUsernameOrEmail.
var usernameOnlyRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// validateUsername implements the "username" struct tag. A valid username is 3–255 runes long and contains only ASCII alphanumeric characters (upper and lower case), underscores, and hyphens. Empty strings pass (combine with "required" to enforce presence).
func validateUsername(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true
	}
	usernameLen := len([]rune(value))
	if usernameLen < 3 || usernameLen > 255 {
		return false
	}
	return usernameOnlyRegex.MatchString(value)
}

// validateUsernameOrEmail implements the "identifier" struct tag. If the value contains an "@" it is validated as an email via isValidEmail. Otherwise it is treated as a username: 3–50 runes, alphanumeric, underscores, and hyphens only. Empty strings pass (combine with "required" to enforce presence).
func validateUsernameOrEmail(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true
	}

	if strings.Contains(value, "@") {
		return isValidEmail(value)
	}

	usernameLen := len([]rune(value))
	if usernameLen < 3 || usernameLen > 50 {
		return false
	}
	return usernameOnlyRegex.MatchString(value)
}

// isValidEmail performs multi-step email validation that is stricter than the built-in "email" tag:
//
//  1. Total length <= 254 characters (RFC 5321 path limit).
//  2. Exactly one "@" separating local-part and domain.
//  3. Local-part: 1–64 characters, no consecutive dots, no leading/trailing dots.
//  4. Domain: <= 253 characters, TLD >= 2 characters and alphabetic only (no numeric TLDs like ".c0m").
//  5. Final regex check against emailRX for character-level validity.
func isValidEmail(email string) bool {
	if len([]rune(email)) > 254 {
		return false
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}

	localPart := parts[0]
	domain := parts[1]

	if len([]rune(localPart)) == 0 || len([]rune(localPart)) > 64 {
		return false
	}
	if len([]rune(domain)) > 253 {
		return false
	}
	if strings.Contains(localPart, "..") || strings.HasPrefix(localPart, ".") || strings.HasSuffix(localPart, ".") {
		return false
	}

	domainParts := strings.Split(domain, ".")
	if len(domainParts) >= 2 {
		tld := domainParts[len(domainParts)-1]
		if len(tld) < 2 {
			return false
		}
		tldRegex := regexp.MustCompile(`^[a-zA-Z]+$`)
		if !tldRegex.MatchString(tld) {
			return false
		}
	}

	return emailRX.MatchString(email)
}

// validateCustomEmail implements the "custom_email" struct tag. It delegates to isValidEmail for the actual checks. Empty strings pass (combine with "required" to enforce presence). Use this instead of the built-in "email" tag when you need the stricter TLD and length enforcement.
func validateCustomEmail(fl validator.FieldLevel) bool {
	email := fl.Field().String()
	if email == "" {
		return true
	}
	return isValidEmail(email)
}

// validateNonzeroDecimal implements the "nonzero_decimal" struct tag. It parses the field value as a decimal string and returns false if the parsed value equals zero. Empty strings pass — combine with "required" to enforce presence. Supports both string and *string fields.
func validateNonzeroDecimal(fl validator.FieldLevel) bool {
	field := fl.Field()
	if field.Kind() == reflect.Pointer {
		if field.IsNil() {
			return true
		}
		field = field.Elem()
	}
	s := field.String()
	if s == "" {
		return true
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return false
	}
	return !d.IsZero()
}

// validateDecimal implements the "decimal" struct tag. It parses the field value as a decimal string and returns false only when it is present but not a parseable decimal. Empty strings and zero pass — combine with "required" to enforce presence. Unlike "nonzero_decimal", zero is a valid value. Supports both string and *string fields.
func validateDecimal(fl validator.FieldLevel) bool {
	field := fl.Field()
	if field.Kind() == reflect.Pointer {
		if field.IsNil() {
			return true
		}
		field = field.Elem()
	}
	s := field.String()
	if s == "" {
		return true
	}
	_, err := decimal.NewFromString(s)
	return err == nil
}

// validateMaxDaysAhead implements the "max_days_ahead" struct tag for time.Time fields (and field.Optional[time.Time], which the custom type func unwraps to the inner time.Time or nil). It fails only when the value is more than N days in the future, where N is the tag parameter (e.g. max_days_ahead=30). Unset/zero and past values pass — combine with "required" to enforce presence.
func validateMaxDaysAhead(fl validator.FieldLevel) bool {
	field := fl.Field()
	if !field.IsValid() {
		return true
	}
	if field.Kind() == reflect.Pointer {
		if field.IsNil() {
			return true
		}
		field = field.Elem()
	}
	t, ok := field.Interface().(time.Time)
	if !ok || t.IsZero() {
		return true
	}
	days, err := strconv.Atoi(fl.Param())
	if err != nil {
		return false
	}
	return !t.After(time.Now().Add(time.Duration(days) * 24 * time.Hour))
}

// validateMultipleOf implements the "multiple_of=N" struct tag for numeric fields (and field.Optional[float64], unwrapped by the custom type func). It passes when the value is a whole multiple of N within floating-point tolerance. Zero/unset values pass — combine with "required" to enforce presence.
func validateMultipleOf(fl validator.FieldLevel) bool {
	f := fl.Field()
	if !f.IsValid() {
		return true
	}
	if f.Kind() == reflect.Pointer {
		if f.IsNil() {
			return true
		}
		f = f.Elem()
	}
	var value float64
	switch f.Kind() {
	case reflect.Float32, reflect.Float64:
		value = f.Float()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value = float64(f.Int())
	default:
		return true
	}
	if value == 0 {
		return true
	}
	step, err := strconv.ParseFloat(fl.Param(), 64)
	if err != nil || step == 0 {
		return false
	}
	ratio := value / step
	return math.Abs(ratio-math.Round(ratio)) < 1e-9
}

// Validate runs all struct-tag validations on v and returns a user-facing [apierror.APIError] on failure (nil on success). When a single field fails, the error's Param is set to that field's JSON/form/query name so the client can highlight the offending input. When multiple fields fail, the error message lists all violations and Param is set to the first failing field.
func Validate(v any) *apierror.APIError {
	err := validate.Struct(v)
	if err != nil {
		return parseValidationErrors(err, v)
	}
	return nil
}

// parseValidationErrors converts a validator error into a user-facing APIError. If the error isn't a validator.ValidationErrors (shouldn't happen in practice), it falls back to a generic validation error. For a single-field failure it returns a targeted error with Param set. For multi-field failures it joins all messages into one error string with Param set to the first field.
func parseValidationErrors(err error, structValue any) *apierror.APIError {
	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return apierror.NewValidationError(err.Error())
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

	return apierror.NewValidationErrorWithParam(fmt.Sprintf("Validation failed for the following fields: %s", strings.Join(fieldErrors, " ")), firstField)
}

// createFieldValidationError builds an APIError for a single-field validation failure, resolving the field name from struct tags and formatting a human-readable message. It selects a specific error code based on the validation tag and the field's source (JSON body vs query/path parameter).
func createFieldValidationError(fieldErr validator.FieldError, structValue any) *apierror.APIError {
	metadata := getFieldMetadata(fieldErr, structValue)
	message := formatFieldError(fieldErr, structValue)

	return newFieldError(fieldErr.Tag(), metadata, message)
}

// newFieldError selects the appropriate error constructor based on the validation tag and whether the field comes from the request body ("field") or a parameter source ("query", "path", "header", etc.).
func newFieldError(tag string, metadata fieldMetadata, message string) *apierror.APIError {
	isBodyField := metadata.source == "field"

	switch tag {
	case "required":
		if isBodyField {
			return apierror.NewMissingFieldError(message, metadata.name)
		}
		return apierror.NewParameterMissingError(message, metadata.name)
	case "email", "custom_email", "password", "username", "identifier", "len":
		if isBodyField {
			return apierror.NewInvalidFormatError(message, metadata.name)
		}
		return apierror.NewParameterInvalidError(message, metadata.name)
	default:
		if isBodyField {
			return apierror.NewInvalidFormatError(message, metadata.name)
		}
		return apierror.NewParameterInvalidError(message, metadata.name)
	}
}

// formatFieldError produces a human-readable error message for a single field validation failure. It resolves the field's public name and source (JSON body, query parameter, path parameter, header, cookie) from struct tags, then formats a message appropriate to the validation tag. Supported tags have dedicated templates; unrecognized tags fall back to a generic "'<field>' is invalid (<tag>)" message.
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
		return fmt.Sprintf("%s '%s' validation failed.", source, fieldName)
	case "password":
		return fmt.Sprintf("%s '%s' must be 8-72 characters and contain at least one lowercase letter, one uppercase letter, one number, and one special character.", source, fieldName)
	case "username":
		return fmt.Sprintf("%s '%s' must be 3-255 characters and contain only letters, numbers, underscores, and hyphens.", source, fieldName)
	case "identifier":
		return fmt.Sprintf("%s '%s' must be a valid email address or username (3-50 characters, alphanumeric, underscores, and hyphens only).", source, fieldName)
	case "custom_email":
		return fmt.Sprintf("%s '%s' must be a valid email address.", source, fieldName)
	case "nonzero_decimal":
		return fmt.Sprintf("%s '%s' must not be zero.", source, fieldName)
	case "decimal":
		return fmt.Sprintf("%s '%s' must be a valid decimal number.", source, fieldName)
	case "max_days_ahead":
		return fmt.Sprintf("%s '%s' must be no more than %s days in the future.", source, fieldName, fieldErr.Param())
	case "multiple_of":
		return fmt.Sprintf("%s '%s' must be a multiple of %s.", source, fieldName, fieldErr.Param())
	default:
		return fmt.Sprintf("%s '%s' is invalid (%s).", source, fieldName, fieldErr.Tag())
	}
}

// formatMinMaxError formats "min" and "max" tag failures with type-aware phrasing. Slices produce "must have at least/at most N item(s)", strings produce "must be at least/at most N characters long", and other types produce a plain numeric comparison.
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

// formatGteLteError formats "gte" and "lte" tag failures with the same type-aware phrasing as formatMinMaxError, using "greater/less than or equal to" wording.
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

// getItemWord returns "item" when param is "1" and "items" otherwise, for grammatically correct slice/array constraint messages.
func getItemWord(param string) string {
	if value, err := strconv.ParseFloat(param, 64); err == nil {
		if value == 1.0 {
			return "item"
		}
	}
	return "items"
}

// fieldMetadata holds the resolved public name and source location of a struct field for error message formatting.
type fieldMetadata struct {
	// name is the user-facing field name resolved from struct tags (e.g. "email", "page_size"). Defaults to the Go field name if no tag is found.
	name string
	// source identifies where the field came from: "field" (JSON body), "query", "path", "header", "form", or "cookie". Used to produce context-aware error prefixes like "Query parameter 'page_size' is required."
	source string
}

// getFieldMetadata resolves a struct field's public name and source from its struct tags. It checks tags in priority order: json first (since most requests are JSON bodies), then form, query, path, header, cookie. The first non-empty, non-"-" tag value wins. If no tag is found, the Go field name is returned with source "field".
func getFieldMetadata(fieldErr validator.FieldError, structValue any) fieldMetadata {
	fieldName := fieldErr.Field()
	if structValue == nil {
		return fieldMetadata{name: fieldName, source: "field"}
	}

	rv := reflect.ValueOf(structValue)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}

	rt := rv.Type()

	for field := range rt.Fields() {
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

// formatSource converts an internal source identifier ("query", "path", etc.) into the human-readable prefix used in error messages (e.g. "Query parameter", "Path parameter"). The default "field" source maps to "Field".
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

// getFieldName is a convenience wrapper that returns only the resolved name from getFieldMetadata, discarding the source. Used when only the Param value is needed.
func getFieldName(fieldErr validator.FieldError, structValue any) string {
	return getFieldMetadata(fieldErr, structValue).name
}

// Validator is a lightweight imperative validation helper for checks that cannot be expressed with struct tags (e.g. cross-field constraints, conditional logic). It collects named errors via AddError or Check and reports validity via Valid.
//
//	v := validate.New()
//	v.Check(req.EndDate.After(req.StartDate), "ends_at", "must be after starts_at")
//	if !v.Valid() { ... }
type Validator struct {
	// Errors maps field names to their first error message. Only the first error per field is stored to keep messages concise.
	Errors map[string]string
}
