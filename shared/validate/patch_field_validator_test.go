package validate

import (
	"strings"
	"testing"

	"github.com/open-mrp/api/shared/field"
)

type patchValidateReq struct {
	Email field.Clearable[string] `json:"email,omitzero" validate:"omitempty,max=255"`
}

func TestPatchFieldValidator_unsetPasses(t *testing.T) {
	t.Parallel()
	if err := Validate(&patchValidateReq{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPatchFieldValidator_setValidates(t *testing.T) {
	t.Parallel()
	req := patchValidateReq{Email: field.Set(strings.Repeat("x", 300))}
	if err := Validate(&req); err == nil {
		t.Fatal("expected max validation error")
	}
}

func TestPatchFieldValidator_clearPasses(t *testing.T) {
	t.Parallel()
	req := patchValidateReq{Email: field.Clear[string]()}
	if err := Validate(&req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
