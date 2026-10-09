package appctx

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCapBody(t *testing.T) {
	t.Parallel()

	if CapBody(nil, 10) != nil {
		t.Error("a nil body stays nil")
	}

	small := `{"a":1}`
	if got := CapBody(&small, 10); got != &small {
		t.Errorf("a body within the limit is returned as is, got %q", *got)
	}

	large := `"` + strings.Repeat("x", 20) + `"`
	got := CapBody(&large, 10)
	var meta struct {
		Truncated    bool `json:"_truncated"`
		OriginalSize int  `json:"_original_size"`
	}
	if err := json.Unmarshal([]byte(*got), &meta); err != nil {
		t.Fatalf("marker is not JSON: %v", err)
	}
	if !meta.Truncated || meta.OriginalSize != len(large) {
		t.Errorf("marker: got %+v", meta)
	}
}
