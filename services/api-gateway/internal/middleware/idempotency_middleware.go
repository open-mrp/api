package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	"github.com/augno/api/services/api-gateway/internal/header"
	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/idempotency"
	pb "github.com/augno/api/shared/proto/platform"
	"github.com/augno/api/shared/tracing"
)

type SerializedCookie struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Path     string `json:"path,omitempty"`
	Domain   string `json:"domain,omitempty"`
	MaxAge   int    `json:"max_age,omitempty"`
	Secure   bool   `json:"secure,omitempty"`
	HttpOnly bool   `json:"http_only,omitempty"`
	SameSite string `json:"same_site,omitempty"`
}

const cookieTTLSeconds = 300 // 5 minutes

var idempotencyMiddlewareTracer = tracing.GetTracer("api-gateway.idempotency_middleware")

const maxIdempotencyResponseSize = 1024 * 64

type responseRecorder struct {
	http.ResponseWriter
	statusCode      int
	body            bytes.Buffer
	written         bool
	capturedCookies []*http.Cookie
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter:  w,
		statusCode:      http.StatusOK,
		capturedCookies: make([]*http.Cookie, 0),
	}
}

func (r *responseRecorder) WriteHeader(code int) {
	if !r.written {
		r.statusCode = code
		r.written = true
		// Capture cookies before writing header
		r.captureCookies()
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if !r.written {
		r.written = true
		// Capture cookies if WriteHeader wasn't called explicitly
		r.captureCookies()
	}
	if r.body.Len() < maxIdempotencyResponseSize {
		remaining := maxIdempotencyResponseSize - r.body.Len()
		if len(b) <= remaining {
			r.body.Write(b)
		} else {
			r.body.Write(b[:remaining])
		}
	}
	return r.ResponseWriter.Write(b)
}

func (r *responseRecorder) captureCookies() {
	for _, cookieStr := range r.ResponseWriter.Header()["Set-Cookie"] {
		h := http.Header{"Set-Cookie": {cookieStr}}
		resp := &http.Response{Header: h}
		cookies := resp.Cookies()
		if len(cookies) > 0 {
			r.capturedCookies = append(r.capturedCookies, cookies[0])
		}
	}
}

func (r *responseRecorder) GetSerializedCookies() ([]byte, error) {
	if len(r.capturedCookies) == 0 {
		return nil, nil
	}

	serialized := make([]SerializedCookie, 0, len(r.capturedCookies))
	for _, c := range r.capturedCookies {
		sc := SerializedCookie{
			Name:     c.Name,
			Value:    c.Value,
			Path:     c.Path,
			Domain:   c.Domain,
			MaxAge:   c.MaxAge,
			Secure:   c.Secure,
			HttpOnly: c.HttpOnly,
		}
		switch c.SameSite {
		case http.SameSiteLaxMode:
			sc.SameSite = "Lax"
		case http.SameSiteStrictMode:
			sc.SameSite = "Strict"
		case http.SameSiteNoneMode:
			sc.SameSite = "None"
		}
		serialized = append(serialized, sc)
	}

	return json.Marshal(serialized)
}

func (r *responseRecorder) HasCookies() bool {
	return len(r.capturedCookies) > 0
}

type IdempotencyMiddlewareConfig struct {
	PlatformClient *grpcclient.PlatformServiceClient
}

func (c *IdempotencyMiddlewareConfig) validate() error {
	return nil
}

func IdempotencyMiddleware(config *IdempotencyMiddlewareConfig) func(http.HandlerFunc) http.HandlerFunc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			idempotencyKey := r.Header.Get(header.IdempotencyKeyHeader)
			if idempotencyKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			if r.Method != http.MethodPost && r.Method != http.MethodPatch && r.Method != http.MethodDelete {
				next.ServeHTTP(w, r)
				return
			}

			if config.PlatformClient == nil {
				ctx := appctx.WithIdempotencyKey(r.Context(), idempotencyKey)
				r = r.WithContext(ctx)
				next.ServeHTTP(w, r)
				return
			}

			// Identity may not be set depending on the endpoint, so we will handle unset identities as unauthorized
			identity, _ := appctx.GetIdentityFromContext(r.Context())

			bodyBytes, r := readAndRestoreBody(r)
			params := readAndRestoreParams(r)

			normalizedRoute := r.URL.Path
			if routePattern, ok := appctx.GetRoutePattern(r.Context()); ok && routePattern != "" {
				normalizedRoute = routePattern
			}

			// For unauthenticated requests, use nil actorID and "unauthenticated" identity type
			var actorID *string
			var targetAccountID *string
			var identityType string
			if identity != nil && identity.Actor != nil {
				actorID = &identity.Actor.ID
				identityType = string(identity.Type)
			} else {
				actorID = nil
				identityType = string(types.IdentityTypeUnauthenticated)
			}
			if identity != nil {
				targetAccountID = identity.TargetAccountID
			}

			rl, ok := appctx.GetRequestLog(r.Context())
			if !ok {
				apiErr := apierror.NewInvariantViolationError("Request log not found in context.")
				httptransport.RespondWithAPIError(r.Context(), w, apiErr)
				return
			}

			scopeHash := idempotency.ComputeHTTPScopeHash(actorID, targetAccountID, r.Method, normalizedRoute, idempotencyKey)
			requestBodyHash := idempotency.ComputeRequestBodyHash(bodyBytes, params)

			var paramsBytes []byte
			if len(params) > 0 {
				paramsBytes, _ = json.Marshal(params)
			}

			rpcCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			resp, err := config.PlatformClient.Client.ProcessIdempotencyKey(rpcCtx, &pb.ProcessIdempotencyKeyRequest{
				ActorId:         actorID,
				IdentityType:    identityType,
				RequestId:       rl.ID,
				RequestMethod:   r.Method,
				NormalizedRoute: normalizedRoute,
				IdempotencyKey:  idempotencyKey,
				RequestParams:   paramsBytes,
				ScopeHash:       scopeHash,
				RequestBodyHash: requestBodyHash,
				TargetAccountId: targetAccountID,
			})

			if err != nil {
				apiErr := contracts.ConvertGRPCError(ctx, err, "platform-service")
				httptransport.RespondWithAPIError(ctx, w, apiErr)
				return
			}

			switch resp.Result {
			case pb.ProcessIdempotencyKeyResult_PROCESS_RESULT_HASH_MISMATCH:
				apiErr := apierror.NewIdempotencyHashMismatchError(idempotencyKey)
				httptransport.RespondWithAPIError(ctx, w, apiErr)
				return

			case pb.ProcessIdempotencyKeyResult_PROCESS_RESULT_REPLAY:
				if rl, ok := appctx.GetRequestLog(ctx); ok && rl != nil {
					rl.IdempotencyKeyID = &resp.IdempotencyKeyId
				}

				// Restore cookies from cached headers
				if len(resp.ResponseHeaders) > 0 {
					var cookies []SerializedCookie
					if err := json.Unmarshal(resp.ResponseHeaders, &cookies); err == nil {
						for _, sc := range cookies {
							cookie := &http.Cookie{
								Name:     sc.Name,
								Value:    sc.Value,
								Path:     sc.Path,
								Domain:   sc.Domain,
								MaxAge:   sc.MaxAge,
								Secure:   sc.Secure,
								HttpOnly: sc.HttpOnly,
							}
							switch sc.SameSite {
							case "Lax":
								cookie.SameSite = http.SameSiteLaxMode
							case "Strict":
								cookie.SameSite = http.SameSiteStrictMode
							case "None":
								cookie.SameSite = http.SameSiteNoneMode
							}
							http.SetCookie(w, cookie)
						}
					}
				}

				w.Header().Set(header.IdempotentReplayedHeader, "true")

				if r.Method == http.MethodDelete && resp.ResponseCode != nil {
					originalCode := int(*resp.ResponseCode)
					if originalCode >= 200 && originalCode < 300 {
						apiErr := apierror.NewResourceGoneError("Resource was deleted.")
						httptransport.RespondWithAPIError(ctx, w, apiErr)
						return
					}
				}

				code := http.StatusOK
				if resp.ResponseCode != nil {
					code = int(*resp.ResponseCode)
				}
				body := resp.ResponseBody
				if body == nil {
					body = []byte("{}")
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(code)
				_, _ = w.Write(body) // #nosec G705 - Cached response body from prior request, not user-controlled
				return

			case pb.ProcessIdempotencyKeyResult_PROCESS_RESULT_IN_PROGRESS:
				apiErr := apierror.NewIdempotencyInProgressError(idempotencyKey)
				httptransport.RespondWithAPIError(ctx, w, apiErr)
				return

			case pb.ProcessIdempotencyKeyResult_PROCESS_RESULT_NEW:
				ctx = appctx.WithIdempotencyKey(ctx, idempotencyKey)
				ctx = appctx.WithIdempotencyKeyID(ctx, resp.IdempotencyKeyId)

				if rl, ok := appctx.GetRequestLog(ctx); ok && rl != nil {
					rl.IdempotencyKeyID = &resp.IdempotencyKeyId
				}

				recorder := newResponseRecorder(w)
				r = r.WithContext(ctx)
				next.ServeHTTP(recorder, r)

				// Capture cookies for caching
				var headers []byte
				var ttl *int32
				if recorder.HasCookies() {
					headers, _ = recorder.GetSerializedCookies()
					ttlVal := int32(cookieTTLSeconds)
					ttl = &ttlVal
				}

				go storeIdempotencyResponse(
					context.WithoutCancel(ctx),
					config.PlatformClient,
					resp.IdempotencyKeyId,
					recorder.statusCode,
					recorder.body.Bytes(),
					headers,
					ttl,
				)
				return

			default:
				ctx = appctx.WithIdempotencyKey(ctx, idempotencyKey)
				r = r.WithContext(ctx)
				next.ServeHTTP(w, r)
			}
		}
	}
}

func readAndRestoreBody(r *http.Request) ([]byte, *http.Request) {
	if r.Body == nil {
		return nil, r
	}

	bodyBytes, err := io.ReadAll(r.Body)
	_ = r.Body.Close()

	if err != nil || len(bodyBytes) == 0 {
		r.Body = io.NopCloser(bytes.NewReader([]byte{}))
		return nil, r
	}

	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	return bodyBytes, r
}

func readAndRestoreParams(r *http.Request) map[string]string {
	params := make(map[string]string)
	for key, values := range r.URL.Query() {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}
	return params
}

func isTransientStatusCode(code int) bool {
	switch code {
	case http.StatusTooManyRequests, // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout:      // 504
		return true
	default:
		return false
	}
}

func storeIdempotencyResponse(
	ctx context.Context,
	client *grpcclient.PlatformServiceClient,
	keyID string,
	statusCode int,
	body []byte,
	headers []byte,
	ttlSeconds *int32,
) {
	if client == nil || keyID == "" {
		return
	}

	ctx, span := idempotencyMiddlewareTracer.Start(ctx, "idempotency.store_response")
	defer span.End()

	if isTransientStatusCode(statusCode) {
		rpcCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		_, err := client.Client.ReleaseIdempotencyKey(rpcCtx, &pb.ReleaseIdempotencyKeyRequest{
			IdempotencyKeyId: keyID,
		})
		if err != nil {
			apiErr := contracts.ConvertGRPCError(ctx, err, "platform-service")
			tracing.RecordControllerError(span, apiErr)
		}
		return
	}

	rpcCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := client.Client.SetIdempotencyKeyResponse(rpcCtx, &pb.SetIdempotencyKeyResponseRequest{
		IdempotencyKeyId: keyID,
		ResponseCode:     int32(statusCode), // #nosec G115 - HTTP status code, always fits int32
		ResponseBody:     body,
		ResponseHeaders:  headers,
		TtlSeconds:       ttlSeconds,
	})
	if err != nil {
		apiErr := contracts.ConvertGRPCError(ctx, err, "platform-service")
		tracing.RecordControllerError(span, apiErr)
	}
}
