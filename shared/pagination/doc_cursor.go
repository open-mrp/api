package pagination

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/augno/api/shared/crypto"
)

// DocumentationHMACKey signs OpenAPI example cursors. It is not used at runtime.
const DocumentationHMACKey = "augno-openapi-documentation-cursor-key-v1"

var documentationHMACKey = []byte(DocumentationHMACKey)

func signCursorPayload(key []byte, v any) string {
	b, _ := json.Marshal(v)
	payload := base64.RawURLEncoding.EncodeToString(b)
	sig := base64.RawURLEncoding.EncodeToString(crypto.HMACSHA256(key, []byte(payload)))
	return payload + "." + sig
}

// EncodeDocumentationCursor returns a signed forward cursor for OpenAPI list examples
// using pagination.Cursor (int64 internal IDs).
func EncodeDocumentationCursor(createdAt time.Time, internalID int64) string {
	return signCursorPayload(documentationHMACKey, Cursor{
		CreatedAt: createdAt,
		ID:        internalID,
		Direction: DirectionForward,
	})
}

// EncodeDocumentationStringCursor returns a signed forward cursor for OpenAPI list examples
// using pagination.StringCursor (varchar primary keys / type IDs).
func EncodeDocumentationStringCursor(occurredAt time.Time, id string) string {
	return signCursorPayload(documentationHMACKey, StringCursor{
		OccurredAt: occurredAt,
		ID:         id,
		Direction:  DirectionForward,
	})
}
