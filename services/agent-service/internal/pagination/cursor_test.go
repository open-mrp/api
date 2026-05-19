package pagination

import (
	"testing"
	"time"

	sharedpagination "github.com/augno/api/shared/pagination"
)

func TestParseOptionalStringCursor_empty(t *testing.T) {
	t.Parallel()
	id, dir, apiErr := ParseOptionalStringCursor(nil)
	if apiErr != nil || id != "" || dir != nil {
		t.Fatalf("got id=%q dir=%v err=%v", id, dir, apiErr)
	}
}

func TestParseOptionalStringCursor_invalid(t *testing.T) {
	t.Parallel()
	raw := "not_a_real_cursor_value"
	_, _, apiErr := ParseOptionalStringCursor(&raw)
	if apiErr == nil {
		t.Fatal("expected error for garbage cursor")
	}
}

func TestParseOptionalStringCursor_validRoundTrip(t *testing.T) {
	t.Parallel()
	sharedpagination.Init([]byte("test-hmac-key-for-agent-pagination"))

	encoded := sharedpagination.EncodeStringCursor(sharedpagination.StringCursor{
		OccurredAt: time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC),
		ID:         "agrn_01seede2e_run00001",
		Direction:  sharedpagination.DirectionForward,
	})

	id, dir, apiErr := ParseOptionalStringCursor(&encoded)
	if apiErr != nil {
		t.Fatalf("unexpected error: %v", apiErr)
	}
	if id != "agrn_01seede2e_run00001" {
		t.Fatalf("id: got %q", id)
	}
	if dir == nil || *dir != sharedpagination.DirectionForward {
		t.Fatalf("dir: got %v", dir)
	}
}
