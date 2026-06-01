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

type NullableReq struct {
	Name  Nullable[string] `json:"name"`
	Phone Nullable[string] `json:"phone"`
}

type embeddedNullableReq struct {
	NullableReq
	Other *Field[string] `json:"other"`
}

func TestExplicitNullField(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		body     string
		v        any
		wantName string
		wantOK   bool
	}{
		{"explicit null", `{"phone": null}`, &NullableReq{}, "phone", true},
		{"first null of many", `{"name": null, "phone": null}`, &NullableReq{}, "name", true},
		{"value not null", `{"phone": "555"}`, &NullableReq{}, "", false},
		{"absent key", `{}`, &NullableReq{}, "", false},
		{"embedded field null", `{"name": null}`, &embeddedNullableReq{}, "name", true},
		{"not an object", `[]`, &NullableReq{}, "", false},
		{"non-nullable null ignored", `{"other": null}`, &embeddedNullableReq{}, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			name, ok := ExplicitNullField([]byte(tt.body), tt.v)
			if ok != tt.wantOK || name != tt.wantName {
				t.Fatalf("got (%q, %v), want (%q, %v)", name, ok, tt.wantName, tt.wantOK)
			}
		})
	}
}
