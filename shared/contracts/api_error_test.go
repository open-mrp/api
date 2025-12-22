package contracts

import (
	"testing"
)

func TestAPIErrorResponse_SchemaExample(t *testing.T) {
	resp := APIErrorResponse{}
	example := resp.SchemaExample()

	if example == nil {
		t.Fatal("SchemaExample returned nil")
	}

	errResp, ok := example.(APIErrorResponse)
	if !ok {
		t.Fatalf("expected APIErrorResponse, got %T", example)
	}

	if errResp.Error.Code == "" {
		t.Error("expected non-empty error code in example")
	}
}

func TestResponseError_SchemaExample(t *testing.T) {
	resp := ResponseError{}
	example := resp.SchemaExample()

	if example == nil {
		t.Fatal("SchemaExample returned nil")
	}

	errResp, ok := example.(ResponseError)
	if !ok {
		t.Fatalf("expected ResponseError, got %T", example)
	}

	if errResp.Code == "" {
		t.Error("expected non-empty error code in example")
	}
}
