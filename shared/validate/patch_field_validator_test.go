package validate

import (
	"strings"
	"testing"

	"github.com/augno/api/shared/patch"
)

type patchValidateReq struct {
	Email *patch.Field[string] `json:"email" validate:"omitempty,max=255"`
}

func TestPatchFieldValidator_unsetPasses(t *testing.T) {
	t.Parallel()
	if err := Validate(&patchValidateReq{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPatchFieldValidator_setValidates(t *testing.T) {
	t.Parallel()
	req := patchValidateReq{Email: new(patch.Set(strings.Repeat("x", 300)))}
	if err := Validate(&req); err == nil {
		t.Fatal("expected max validation error")
	}
}

func TestPatchFieldValidator_clearPasses(t *testing.T) {
	t.Parallel()
	req := patchValidateReq{Email: new(patch.Clear[string]())}
	if err := Validate(&req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
