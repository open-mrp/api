package patch

import (
	"encoding/json"
	"testing"
)

type ptrFieldReq struct {
	Description *Field[string] `json:"description"`
}

func TestApplyPtrFieldNulls_explicitNullClears(t *testing.T) {
	t.Parallel()
	body := []byte(`{"description": null}`)
	var req ptrFieldReq
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatal(err)
	}
	if req.Description != nil {
		t.Fatal("expected nil before ApplyPtrFieldNulls")
	}
	ApplyPtrFieldNulls(body, &req)
	if req.Description == nil || !req.Description.IsClear() {
		t.Fatal("expected clear after ApplyPtrFieldNulls")
	}
}

func TestApplyPtrFieldNulls_absentKeyUnset(t *testing.T) {
	t.Parallel()
	body := []byte(`{}`)
	var req ptrFieldReq
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatal(err)
	}
	ApplyPtrFieldNulls(body, &req)
	if req.Description != nil {
		t.Fatal("expected nil for absent key")
	}
}

func TestApplyPtrFieldNulls_valueSets(t *testing.T) {
	t.Parallel()
	body := []byte(`{"description": "hello"}`)
	var req ptrFieldReq
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatal(err)
	}
	ApplyPtrFieldNulls(body, &req)
	if req.Description == nil || !req.Description.IsSet() {
		t.Fatal("expected set")
	}
}
