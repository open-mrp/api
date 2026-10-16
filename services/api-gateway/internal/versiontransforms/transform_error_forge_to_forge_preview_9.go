package versiontransforms

import (
	"net/http"
	"strings"

	"github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/object"
	"github.com/open-mrp/apikit/version"
)

// forge reshaped the error object (an errors list on every error, no request_log_url, quota only on
// limit_exceeded), made validation failures 422 with the failing fields listed under validation_failed, renamed
// rate_limit_exceeded to rate_limited, and gave idempotency key reuse and in-use deletes their own codes. preview.9
// callers get the error as preview.9 sent it: 400 for invalid input, a single failing field's own code at the top,
// and the old code names.
type errorForgeToPreview9 struct {
	forgeToPreview9
	noRequestChange
}

func init() {
	version.RegisterTransformer(errorForgeToPreview9{})
}

// It changes no successful response; errors go through TransformError.
func (errorForgeToPreview9) ObjectTypes() []object.Type { return nil }

func (errorForgeToPreview9) Transform(_ object.Type, data map[string]any) map[string]any {
	return data
}

const badCursorMessage = "Invalid pagination cursor."

var preview9ErrorCodes = map[string]string{
	"rate_limited":           "rate_limit_exceeded",
	"idempotency_key_reused": "validation_failed",
	"resource_in_use":        "resource_conflict",
}

func (errorForgeToPreview9) TransformError(status int, body map[string]any) (int, map[string]any) {
	e, ok := body["error"].(map[string]any)
	if !ok {
		return status, body
	}
	code, _ := e["code"].(string)
	fields, _ := e["errors"].([]any)
	switch {
	case code == "validation_failed" && len(fields) == 1:
		// preview.9 reported a single failing field under its own code, missing_field or invalid_format.
		if field, ok := fields[0].(map[string]any); ok {
			if fieldCode, ok := field["code"].(string); ok && fieldCode != "" {
				e["code"] = fieldCode
			}
			// preview.9 rejected a body enum outside its set as an invalid parameter.
			if message, _ := field["message"].(string); field["code"] == "invalid_format" && strings.HasPrefix(message, "Field '") && strings.Contains(message, "' must be one of: ") {
				e["code"] = "parameter_invalid"
			}
		}
	case code == "parameter_invalid" && e["param"] == "cursor" && e["message"] == badCursorMessage:
		// forge reports a bad cursor as an invalid query parameter; preview.9 reported it as a field in the wrong format.
		e["code"] = "invalid_format"
	case code == "parameter_invalid" && e["param"] == nil:
		// A body that does not parse was a validation failure in preview.9.
		e["code"] = "validation_failed"
	default:
		if old, ok := preview9ErrorCodes[code]; ok {
			e["code"] = old
		}
	}
	if newCode, _ := e["code"].(string); newCode != code {
		if _, ok := e["doc_url"].(string); ok {
			if url := apierror.Code(newCode).DocURL(); url != "" {
				e["doc_url"] = url
			}
		}
	}
	if status == http.StatusUnprocessableEntity {
		status = http.StatusBadRequest
	}
	delete(e, "errors")
	if _, ok := e["quota"]; !ok {
		e["quota"] = nil
	}
	e["request_log_url"] = nil
	return status, body
}
