package pagination

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/augno/api/shared/crypto"
)

var cursorKey []byte

// Init sets the HMAC key used to sign and verify cursors. Must be called once at service startup before any cursor operations.
func Init(key []byte) {
	cursorKey = key
}

type Direction string

const (
	DirectionForward  Direction = "f"
	DirectionBackward Direction = "b"
)

type Cursor struct {
	CreatedAt time.Time `json:"c"`
	ID        int64     `json:"i"`
	Direction Direction `json:"d"`
}

// StringCursor is like Cursor but uses a string ID instead of int64, for tables whose primary key is a varchar (e.g. request_log).
type StringCursor struct {
	OccurredAt time.Time `json:"c"`
	ID         string    `json:"s"`
	Direction  Direction `json:"d"`
	// MatchTier is set for ranked catalog search pagination (exact/prefix/substring SKU tiers).
	MatchTier *int `json:"t,omitempty"`
}

func encodeCursorPayload(v any) string {
	return signCursorPayload(cursorKey, v)
}

func decodeCursorPayload(s string) ([]byte, error) {
	parts := strings.SplitN(s, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid cursor format")
	}

	payload, sigStr := parts[0], parts[1]

	sig, err := base64.RawURLEncoding.DecodeString(sigStr)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor encoding")
	}

	if !crypto.VerifyHMACSHA256(cursorKey, []byte(payload), sig) {
		return nil, fmt.Errorf("invalid cursor signature")
	}

	b, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor encoding")
	}

	return b, nil
}

func EncodeCursor(c Cursor) string {
	return encodeCursorPayload(c)
}

func DecodeCursor(s string) (Cursor, error) {
	b, err := decodeCursorPayload(s)
	if err != nil {
		return Cursor{}, err
	}

	var c Cursor
	if err := json.Unmarshal(b, &c); err != nil {
		return Cursor{}, fmt.Errorf("invalid cursor format")
	}

	if c.Direction != DirectionForward && c.Direction != DirectionBackward {
		return Cursor{}, fmt.Errorf("invalid cursor direction")
	}

	return c, nil
}

func EncodeStringCursor(c StringCursor) string {
	return encodeCursorPayload(c)
}

func DecodeStringCursor(s string) (StringCursor, error) {
	b, err := decodeCursorPayload(s)
	if err != nil {
		return StringCursor{}, err
	}

	var c StringCursor
	if err := json.Unmarshal(b, &c); err != nil {
		return StringCursor{}, fmt.Errorf("invalid cursor format")
	}

	if c.Direction != DirectionForward && c.Direction != DirectionBackward {
		return StringCursor{}, fmt.Errorf("invalid cursor direction")
	}

	return c, nil
}
