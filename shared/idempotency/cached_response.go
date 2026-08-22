package idempotency

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/open-mrp/api/shared/appctx"
	apierror "github.com/open-mrp/api/shared/errors"
)

// CachedResult holds the outcome of attempting to deserialize a previously stored idempotency response. Service handlers use this to short-circuit execution when a cache hit is found.
//
// The type parameter T is the expected success response struct (e.g. a proto-generated message or a domain DTO). On a cache hit the result contains either the deserialized success data or a structured API error, depending on the original response's status code.
type CachedResult[T any] struct {
	// HasCache is true when a previously stored response was found for the idempotency key. When false, the handler should proceed normally (first-time request).
	HasCache bool
	// Data holds the deserialized success response body. Non-nil only when HasCache is true and the original response had a status code below 400.
	Data *T
	// Error holds the deserialized API error. Non-nil only when HasCache is true and the original response had a status code >= 400.
	Error *apierror.APIError
}

// UnmarshalCachedResponse deserializes a stored idempotency response into a typed CachedResult. It is called by service handlers after looking up an idempotency key to determine whether a cached response can be returned instead of re-executing the handler logic.
//
// Behavior by input:
//   - statusCode == nil: no cache entry exists; returns {HasCache: false}.
//   - statusCode >= 400: the original request produced an error; body is deserialized
//     as an APIError via apierror.APIErrorFromJSON.
//   - statusCode < 400: the original request succeeded; body is deserialized into T.
//
// In both cache-hit cases, appctx.MarkIdempotencyReplayed is called so the transport layer can set the appropriate idempotent-replayed header on the outgoing response.
//
// Returns an error if body is empty (when statusCode is non-nil) or if JSON deserialization fails. These are internal errors that indicate a corrupted cache entry rather than a client mistake.
func UnmarshalCachedResponse[T any](ctx context.Context, statusCode *int, body json.RawMessage) (CachedResult[T], error) {
	var zero CachedResult[T]
	if statusCode == nil {
		return CachedResult[T]{HasCache: false}, nil
	}

	if len(body) == 0 {
		return zero, errors.New("cached response body missing")
	}

	code := *statusCode
	if code >= http.StatusBadRequest {
		apiErr, err := apierror.APIErrorFromJSON(body)
		if err != nil {
			return zero, fmt.Errorf("unmarshal cached error response: %w", err)
		}
		appctx.MarkIdempotencyReplayed(ctx)
		return CachedResult[T]{HasCache: true, Data: nil, Error: apiErr}, nil

	} else {
		var result T
		if err := json.Unmarshal(body, &result); err != nil {
			return zero, fmt.Errorf("unmarshal cached response: %w", err)
		}
		appctx.MarkIdempotencyReplayed(ctx)
		return CachedResult[T]{HasCache: true, Data: &result, Error: nil}, nil
	}
}
