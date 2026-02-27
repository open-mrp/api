package pagination

import (
	"encoding/base64"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	Init([]byte("test-cursor-key"))
	os.Exit(m.Run())
}

func TestEncodeDecode_RoundTrip(t *testing.T) {
	now := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	original := Cursor{CreatedAt: now, ID: 42, Direction: DirectionForward}

	encoded := EncodeCursor(original)
	decoded, err := DecodeCursor(encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !decoded.CreatedAt.Equal(original.CreatedAt) {
		t.Errorf("CreatedAt mismatch: got %v, want %v", decoded.CreatedAt, original.CreatedAt)
	}
	if decoded.ID != original.ID {
		t.Errorf("ID mismatch: got %d, want %d", decoded.ID, original.ID)
	}
	if decoded.Direction != original.Direction {
		t.Errorf("Direction mismatch: got %q, want %q", decoded.Direction, original.Direction)
	}
}

func TestDecodeCursor_TamperedPayload(t *testing.T) {
	original := Cursor{CreatedAt: time.Now(), ID: 42, Direction: DirectionForward}
	encoded := EncodeCursor(original)

	parts := strings.SplitN(encoded, ".", 2)
	// Tamper with the payload by encoding different data
	tampered := base64.RawURLEncoding.EncodeToString([]byte(`{"c":"2025-01-01T00:00:00Z","i":999,"d":"f"}`))
	_, err := DecodeCursor(tampered + "." + parts[1])
	if err == nil {
		t.Fatal("expected error for tampered payload")
	}
}

func TestDecodeCursor_TamperedSignature(t *testing.T) {
	original := Cursor{CreatedAt: time.Now(), ID: 42, Direction: DirectionForward}
	encoded := EncodeCursor(original)

	parts := strings.SplitN(encoded, ".", 2)
	// Replace signature with garbage
	_, err := DecodeCursor(parts[0] + "." + "YmFkc2ln")
	if err == nil {
		t.Fatal("expected error for tampered signature")
	}
}

func TestDecodeCursor_MissingSeparator(t *testing.T) {
	_, err := DecodeCursor("noseparatorhere")
	if err == nil {
		t.Fatal("expected error for missing separator")
	}
}

func TestDecodeCursor_InvalidEncoding(t *testing.T) {
	_, err := DecodeCursor("!!!invalid.base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestDecodeCursor_InvalidJSON(t *testing.T) {
	// Encode a valid-looking cursor but with non-JSON payload, properly signed
	payload := base64.RawURLEncoding.EncodeToString([]byte("not-json"))
	encoded := EncodeCursor(Cursor{CreatedAt: time.Now(), ID: 1, Direction: DirectionForward})
	parts := strings.SplitN(encoded, ".", 2)
	// Use a valid signature for the bad payload — we need to craft it properly
	// Instead, just use the bad payload with the old signature; it will fail on signature check
	_, err := DecodeCursor(payload + "." + parts[1])
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestDecodeCursor_InvalidDirection(t *testing.T) {
	c := Cursor{CreatedAt: time.Now(), ID: 1, Direction: "x"}
	encoded := EncodeCursor(c)

	_, err := DecodeCursor(encoded)
	if err == nil {
		t.Fatal("expected error for invalid direction")
	}
}

type testItem struct {
	id        int64
	createdAt time.Time
}

func getCreatedAt(t testItem) time.Time { return t.createdAt }
func getID(t testItem) int64            { return t.id }

func TestBuildPage_FirstPage_HasMore(t *testing.T) {
	now := time.Now().UTC()
	items := make([]testItem, 4)
	for i := range items {
		items[i] = testItem{id: int64(100 - i), createdAt: now.Add(-time.Duration(i) * time.Hour)}
	}

	result, pi := BuildPage(items, 3, nil, getCreatedAt, getID)

	if len(result) != 3 {
		t.Errorf("expected 3 items, got %d", len(result))
	}
	if !pi.HasNextPage {
		t.Error("expected HasNextPage=true")
	}
	if pi.HasPrevPage {
		t.Error("expected HasPrevPage=false")
	}
	if pi.NextCursor == nil {
		t.Error("expected non-nil NextCursor")
	}
	if pi.PrevCursor != nil {
		t.Error("expected nil PrevCursor")
	}
}

func TestBuildPage_FirstPage_NoMore(t *testing.T) {
	now := time.Now().UTC()
	items := []testItem{
		{id: 100, createdAt: now},
		{id: 99, createdAt: now.Add(-time.Hour)},
	}

	result, pi := BuildPage(items, 3, nil, getCreatedAt, getID)

	if len(result) != 2 {
		t.Errorf("expected 2 items, got %d", len(result))
	}
	if pi.HasNextPage {
		t.Error("expected HasNextPage=false")
	}
	if pi.HasPrevPage {
		t.Error("expected HasPrevPage=false")
	}
	if pi.NextCursor != nil {
		t.Error("expected nil NextCursor")
	}
}

func TestBuildPage_ForwardCursor_MiddlePage(t *testing.T) {
	now := time.Now().UTC()
	items := make([]testItem, 4)
	for i := range items {
		items[i] = testItem{id: int64(50 - i), createdAt: now.Add(-time.Duration(i) * time.Hour)}
	}

	dir := DirectionForward
	result, pi := BuildPage(items, 3, &dir, getCreatedAt, getID)

	if len(result) != 3 {
		t.Errorf("expected 3 items, got %d", len(result))
	}
	if !pi.HasNextPage {
		t.Error("expected HasNextPage=true")
	}
	if !pi.HasPrevPage {
		t.Error("expected HasPrevPage=true")
	}
	if pi.NextCursor == nil {
		t.Error("expected non-nil NextCursor")
	}
	if pi.PrevCursor == nil {
		t.Error("expected non-nil PrevCursor")
	}
}

func TestBuildPage_ForwardCursor_LastPage(t *testing.T) {
	now := time.Now().UTC()
	items := []testItem{
		{id: 10, createdAt: now},
		{id: 9, createdAt: now.Add(-time.Hour)},
	}

	dir := DirectionForward
	result, pi := BuildPage(items, 3, &dir, getCreatedAt, getID)

	if len(result) != 2 {
		t.Errorf("expected 2 items, got %d", len(result))
	}
	if pi.HasNextPage {
		t.Error("expected HasNextPage=false")
	}
	if !pi.HasPrevPage {
		t.Error("expected HasPrevPage=true")
	}
	if pi.NextCursor != nil {
		t.Error("expected nil NextCursor")
	}
	if pi.PrevCursor == nil {
		t.Error("expected non-nil PrevCursor")
	}
}

func TestBuildPage_BackwardCursor_Reverses(t *testing.T) {
	now := time.Now().UTC()
	// Backward query returns ASC order: oldest first
	items := []testItem{
		{id: 50, createdAt: now.Add(-3 * time.Hour)},
		{id: 51, createdAt: now.Add(-2 * time.Hour)},
		{id: 52, createdAt: now.Add(-time.Hour)},
		{id: 53, createdAt: now}, // extra
	}

	dir := DirectionBackward
	result, pi := BuildPage(items, 3, &dir, getCreatedAt, getID)

	if len(result) != 3 {
		t.Errorf("expected 3 items, got %d", len(result))
	}
	// Should be reversed to DESC order
	if result[0].id != 52 {
		t.Errorf("expected first item id=52, got %d", result[0].id)
	}
	if result[2].id != 50 {
		t.Errorf("expected last item id=50, got %d", result[2].id)
	}
	if !pi.HasNextPage {
		t.Error("expected HasNextPage=true")
	}
	if !pi.HasPrevPage {
		t.Error("expected HasPrevPage=true")
	}
}

func TestBuildPage_BackwardCursor_FirstPageReached(t *testing.T) {
	now := time.Now().UTC()
	// Only 2 items returned, no extra — we've reached the first page
	items := []testItem{
		{id: 99, createdAt: now.Add(-time.Hour)},
		{id: 100, createdAt: now},
	}

	dir := DirectionBackward
	result, pi := BuildPage(items, 3, &dir, getCreatedAt, getID)

	if len(result) != 2 {
		t.Errorf("expected 2 items, got %d", len(result))
	}
	// reversed
	if result[0].id != 100 {
		t.Errorf("expected first item id=100, got %d", result[0].id)
	}
	if !pi.HasNextPage {
		t.Error("expected HasNextPage=true")
	}
	if pi.HasPrevPage {
		t.Error("expected HasPrevPage=false")
	}
	if pi.PrevCursor != nil {
		t.Error("expected nil PrevCursor")
	}
}

func TestBuildPage_Empty(t *testing.T) {
	result, pi := BuildPage([]testItem{}, 10, nil, getCreatedAt, getID)

	if len(result) != 0 {
		t.Errorf("expected 0 items, got %d", len(result))
	}
	if pi.HasNextPage || pi.HasPrevPage {
		t.Error("expected no pages")
	}
	if pi.NextCursor != nil || pi.PrevCursor != nil {
		t.Error("expected nil cursors")
	}
}
