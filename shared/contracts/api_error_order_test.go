package contracts

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAPIError_ToResponseMap_Order(t *testing.T) {
	apiErr := &APIError{
		Code:          ErrorCodeValidationFailed,
		Type:          ErrorTypeInvalidRequest,
		PublicMessage: "Invalid input",
		Param:         "email",
		DocURL:        "https://docs.example.com/errors",
	}

	resp := apiErr.ToResponseMap()
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	jsonStr := string(data)

	// Current implementation uses map[string]any, which sorts keys alphabetically during marshal.
	// Expected alphabetical order: code, doc_url, message, param, type
	// Desired order (as defined in ToResponseMap): code, type, message, param, doc_url

	expectedOrder := []string{
		`"code":"validation_failed"`,
		`"type":"invalid_request_error"`,
		`"message":"Invalid input"`,
		`"param":"email"`,
		`"doc_url":"https://docs.example.com/errors"`,
	}

	lastIdx := -1
	for _, expected := range expectedOrder {
		idx := strings.Index(jsonStr, expected)
		if idx == -1 {
			t.Errorf("expected %s to be in JSON, but not found in %s", expected, jsonStr)
			continue
		}
		if idx < lastIdx {
			t.Errorf("expected %s to appear after previous field, but it appeared before in %s", expected, jsonStr)
		}
		lastIdx = idx
	}
}
