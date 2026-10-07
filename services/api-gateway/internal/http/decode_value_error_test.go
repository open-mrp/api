package httptransport

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

type decodeQuantity struct {
	Value field.Optional[int32] `json:"value,omitzero"`
}

type decodeValueReq struct {
	Count    field.Optional[int32]           `json:"count,omitzero"`
	Name     field.Optional[string]          `json:"name,omitzero"`
	Verified field.Clearable[time.Time]      `json:"verified,omitzero"`
	Due      time.Time                       `json:"due"`
	Plain    int32                           `json:"plain"`
	Quantity field.Optional[decodeQuantity]  `json:"quantity,omitzero"`
	Lines    []decodeQuantity                `json:"lines"`
	Patch    field.Clearable[decodeQuantity] `json:"patch,omitzero"`
}

func decodeValueBody(t *testing.T, body string) error {
	t.Helper()
	req := httptest.NewRequest(http.MethodPatch, "/test", bytes.NewBufferString(body))
	return DecodeJSONInto(&decodeValueReq{}, req, true)
}

// A value of the wrong type is refused naming the field it was sent for, however deep it sits and
// whichever UnmarshalJSON decoded it.
func TestDecodeJSONInto_valueErrorNamesTheField(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		body        string
		wantParam   string
		wantMessage string
	}{
		{"optional int given text", `{"count":"abc"}`, "count", "Invalid type for field 'count': expected 32-bit integer, got string."},
		{"optional int given a fraction", `{"count":1.5}`, "count", "Invalid type for field 'count': expected 32-bit integer, got number 1.5."},
		{"optional int out of range", `{"count":2147483648}`, "count", "Invalid type for field 'count': expected 32-bit integer, got number 2147483648."},
		{"optional int given an object", `{"count":{"value":1}}`, "count", "Invalid type for field 'count': expected 32-bit integer, got object."},
		{"optional string given a number", `{"name":12}`, "name", "Invalid type for field 'name': expected string, got number."},
		{"clearable time given a number", `{"verified":12}`, "verified", "Invalid type for field 'verified': expected an RFC 3339 timestamp, got number."},
		{"clearable time not a time", `{"verified":"yesterday"}`, "verified", "Invalid format for field 'verified': expected an RFC 3339 timestamp, such as 2024-01-15T10:30:00Z."},
		{"plain time not a time", `{"due":"soon"}`, "due", "Invalid format for field 'due': expected an RFC 3339 timestamp, such as 2024-01-15T10:30:00Z."},
		{"plain int given text", `{"plain":"x"}`, "plain", "Invalid type for field 'plain': expected 32-bit integer, got string."},
		{"field inside an optional object", `{"quantity":{"value":"x"}}`, "quantity.value", "Invalid type for field 'quantity.value': expected 32-bit integer, got string."},
		{"optional object given a string", `{"quantity":"x"}`, "quantity", "Invalid type for field 'quantity': expected object, got string."},
		{"field of a list item", `{"lines":[{"value":1},{"value":"x"}]}`, "lines[1].value", "Invalid type for field 'lines[1].value': expected 32-bit integer, got string."},
		{"null optional", `{"count":null}`, "count", "Field 'count' cannot be null."},
		{"null optional inside a clearable object", `{"patch":{"value":null}}`, "patch.value", "Field 'patch.value' cannot be null."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := decodeValueBody(t, tt.body)
			apiErr, ok := errors.AsType[*apierror.APIError](err)
			if !ok {
				t.Fatalf("got %T %v, want *apierror.APIError", err, err)
			}
			if apiErr.Code != apierror.ErrorCodeInvalidFormat {
				t.Errorf("code = %s, want %s", apiErr.Code, apierror.ErrorCodeInvalidFormat)
			}
			if apiErr.Param != tt.wantParam {
				t.Errorf("param = %q, want %q", apiErr.Param, tt.wantParam)
			}
			if apiErr.PublicMessage != tt.wantMessage {
				t.Errorf("message = %q, want %q", apiErr.PublicMessage, tt.wantMessage)
			}
		})
	}
}

func TestDecodeJSONInto_bodyOfTheWrongType(t *testing.T) {
	t.Parallel()
	apiErr, ok := errors.AsType[*apierror.APIError](decodeValueBody(t, `[1]`))
	if !ok {
		t.Fatal("want an APIError")
	}
	if apiErr.Param != "" || !strings.Contains(apiErr.PublicMessage, "expected object, got array") {
		t.Fatalf("got param %q message %q", apiErr.Param, apiErr.PublicMessage)
	}
}
