package pagination

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestEncodeDocumentationStringCursor_ProductionShape(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	encoded := EncodeDocumentationStringCursor(at, "ml_014613b8f7959a091d8cc0cef4")

	parts := strings.SplitN(encoded, ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		t.Fatalf("encoded cursor = %q, want payload.signature", encoded)
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	var c StringCursor
	if err := json.Unmarshal(payload, &c); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if !c.OccurredAt.Equal(at) {
		t.Errorf("OccurredAt = %v, want %v", c.OccurredAt, at)
	}
	if c.ID != "ml_014613b8f7959a091d8cc0cef4" {
		t.Errorf("ID = %q", c.ID)
	}
	if c.Direction != DirectionForward {
		t.Errorf("Direction = %q, want %q", c.Direction, DirectionForward)
	}
}

func TestEncodeDocumentationCursor_ProductionShape(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	encoded := EncodeDocumentationCursor(at, 9000000000000001)

	parts := strings.SplitN(encoded, ".", 2)
	if len(parts) != 2 {
		t.Fatalf("encoded cursor = %q", encoded)
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	var c Cursor
	if err := json.Unmarshal(payload, &c); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if c.ID != 9000000000000001 {
		t.Errorf("ID = %d", c.ID)
	}
	if c.Direction != DirectionForward {
		t.Errorf("Direction = %q", c.Direction)
	}
}
