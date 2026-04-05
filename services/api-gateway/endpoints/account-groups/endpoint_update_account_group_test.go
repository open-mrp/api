package accountgroupep

import (
	"encoding/json"
	"testing"

	"github.com/augno/api/shared/validate"
)

func TestRejectExplicitJSONNulls_updateAccountGroup_nameNull(t *testing.T) {
	t.Parallel()
	body := []byte(`{"name": null}`)
	var req UpdateAccountGroupRequest
	if err := validate.RejectExplicitJSONNulls(body, &req); err == nil {
		t.Fatal("expected error for name: null")
	}
}

func TestUpdateAccountGroupRequest_JSON_omittedNameOK(t *testing.T) {
	t.Parallel()
	body := `{}`
	var req UpdateAccountGroupRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Name != nil {
		t.Fatalf("expected Name nil, got %v", req.Name)
	}
}

func TestUpdateAccountGroupRequest_JSON_nameStringOK(t *testing.T) {
	t.Parallel()
	body := `{"name": "Retail"}`
	var req UpdateAccountGroupRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Name == nil || *req.Name != "Retail" {
		t.Fatalf("unexpected Name: %+v", req.Name)
	}
}
