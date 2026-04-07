package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"

	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/version"
)

// responseCapture wraps http.ResponseWriter to capture the response body
type responseCapture struct {
	http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
}

func (rc *responseCapture) Write(b []byte) (int, error) {
	return rc.body.Write(b)
}

func (rc *responseCapture) WriteHeader(statusCode int) {
	rc.statusCode = statusCode
}

// VersionTransformMiddleware applies version transformers to transform responses
// from the current/latest API version format to the requested version format.
func VersionTransformMiddleware(objectType constants.ObjectType) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Get the requested version from context
			requestVersion, ok := appctx.GetAPIVersionFromContext(ctx)
			if !ok {
				// No version in context, skip transformation
				next.ServeHTTP(w, r)
				return
			}

			// If requesting latest version, no transformation needed
			if requestVersion.Equal(version.Latest) {
				next.ServeHTTP(w, r)
				return
			}

			// Capture the response
			capture := &responseCapture{
				ResponseWriter: w,
				body:           &bytes.Buffer{},
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(capture, r)

			// Only transform successful JSON responses
			if capture.statusCode >= 200 && capture.statusCode < 300 {
				contentType := w.Header().Get("Content-Type")
				if contentType == "application/json" || contentType == "" {
					if transformed, ok := transformResponseData(capture.body.Bytes(), version.Latest, requestVersion, objectType); ok {
						httptransport.RespondWithJSON(ctx, w, capture.statusCode, transformed)
						return
					}
				}
			}

			// Write the original response
			w.WriteHeader(capture.statusCode)
			_, _ = w.Write(capture.body.Bytes())
		}
	}
}

// transformResponseData applies the transformer chain to the response body,
// returning the transformed data structure for RespondWithJSON to marshal.
func transformResponseData(body []byte, from, to version.APIVersion, objectType constants.ObjectType) (any, bool) {
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, false
	}

	return version.Transform(from, to, objectType, data), true
}
