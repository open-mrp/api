package idempotency

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

// separator is the ASCII Unit Separator (0x1F), used to delimit fields in the
// hash input string. Using a non-printable character prevents accidental collisions
// when field values contain common delimiters like commas or colons.
const separator = "\x1f"

// ComputeHTTPScopeHash produces a deterministic SHA-256 hex digest that uniquely
// identifies the "scope" of an idempotent HTTP request. The scope binds the
// idempotency key to a specific actor, target account, HTTP method, and route so
// that the same key used by a different actor or against a different endpoint is
// treated as a distinct request.
//
// Fields are concatenated with a unit-separator delimiter and hashed:
//
//	SHA256(actorID + \x1f + targetAccountID + \x1f + method + \x1f + normalizedRoute + \x1f + idempotencyKey)
//
// Nil pointer fields (actorID, targetAccountID) are coerced to empty strings,
// meaning nil and "" produce the same hash. This is intentional — unauthenticated
// requests have no actor ID, and some endpoints have no target account.
//
// The returned hash is stored in the idempotency_keys table as scope_hash and
// compared on subsequent requests to detect key reuse across different scopes.
func ComputeHTTPScopeHash(actorID, targetAccountID *string, method, normalizedRoute, idempotencyKey string) string {
	emptyString := ""
	if actorID == nil {
		actorID = &emptyString
	}
	if targetAccountID == nil {
		targetAccountID = &emptyString
	}
	input := *actorID + separator + *targetAccountID + separator + method + separator + normalizedRoute + separator + idempotencyKey
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

// ComputeServiceScopeHash produces a deterministic SHA-256 hex digest that uniquely
// identifies the "scope" of an idempotent gRPC service call. This is the service-layer
// counterpart of ComputeHTTPScopeHash, used by idempotency mediators in backend
// services (e.g. auth-service) rather than the HTTP gateway.
//
// Fields are concatenated with a unit-separator delimiter and hashed:
//
//	SHA256(actorID + \x1f + service + \x1f + handler + \x1f + idempotencyKey)
//
// Note that the field count differs from ComputeHTTPScopeHash (4 vs 5), so even
// identical field values will produce different hashes, preventing cross-layer
// collisions.
func ComputeServiceScopeHash(actorID *string, service, handler, idempotencyKey string) string {
	emptyString := ""
	if actorID == nil {
		actorID = &emptyString
	}
	input := *actorID + separator + service + separator + handler + separator + idempotencyKey
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

// ComputeRequestBodyHash produces a deterministic SHA-256 hex digest of the request
// payload. It is stored alongside the scope hash and compared on retries to detect
// when a client reuses the same idempotency key with a different request body (which
// is an error — the key should be unique per intended mutation).
//
// The body is first canonicalized via CanonicalizeJSON so that semantically identical
// payloads with different key ordering or whitespace produce the same hash. If the
// body is not valid JSON (e.g. form-encoded), the raw bytes are used as-is.
//
// URL query parameters are appended to the canonical body in sorted key order,
// separated by the unit-separator character. This ensures that requests like
// POST /api?page=1&limit=10 and POST /api?limit=10&page=1 hash identically.
func ComputeRequestBodyHash(body []byte, params map[string]string) string {
	canonicalized, err := CanonicalizeJSON(body)
	if err != nil {
		canonicalized = body
	}

	var buf bytes.Buffer
	buf.Write(canonicalized)

	if len(params) > 0 {
		keys := make([]string, 0, len(params))
		for k := range params {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			buf.WriteString(separator)
			buf.WriteString(k)
			buf.WriteString("=")
			buf.WriteString(params[k])
		}
	}

	hash := sha256.Sum256(buf.Bytes())
	return hex.EncodeToString(hash[:])
}

// CanonicalizeJSON re-encodes a JSON value into a deterministic byte representation.
// It round-trips the data through json.Unmarshal/json.Marshal, which normalizes
// whitespace and (via Go's map iteration + encoding/json's sorted-key output)
// produces consistent key ordering for objects at all nesting levels.
//
// Array element order is preserved — only object key order is normalized.
//
// Returns ([]byte{}, nil) for empty input and (nil, error) for malformed JSON.
func CanonicalizeJSON(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return []byte{}, nil
	}

	var parsed any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, err
	}

	canonical := canonicalizeValue(parsed)

	return json.Marshal(canonical)
}

// canonicalizeValue recursively walks a parsed JSON value, rebuilding maps and
// slices so that encoding/json will serialize them with sorted keys.
func canonicalizeValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		return canonicalizeMap(val)
	case []any:
		return canonicalizeSlice(val)
	default:
		return val
	}
}

// canonicalizeMap copies a JSON object into a fresh map, recursively canonicalizing
// nested values. The new map ensures encoding/json produces sorted-key output.
func canonicalizeMap(m map[string]any) map[string]any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = canonicalizeValue(v)
	}
	return result
}

// canonicalizeSlice copies a JSON array, recursively canonicalizing each element.
// Element order is preserved since JSON arrays are ordered.
func canonicalizeSlice(s []any) []any {
	result := make([]any, len(s))
	for i, v := range s {
		result[i] = canonicalizeValue(v)
	}
	return result
}
