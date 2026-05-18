package apiendpoint

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/augno/api/services/api-gateway/internal/header"
	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
	"github.com/augno/api/shared/validate"
	"github.com/augno/api/shared/version"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type APIEndpointExtras struct {
	SkipRequestBodyParsing bool `json:"skip_request_body_parsing" yaml:"skip_request_body_parsing"`
	ShieldRequestBody      bool `json:"shield_request_body" yaml:"shield_request_body"`
	ShieldResponseBody     bool `json:"shield_response_body" yaml:"shield_response_body"`
	SkipRequestLogging     bool `json:"skip_request_logging" yaml:"skip_request_logging"`
}

type ServiceHandler[TReq, TResp any] func(ctx context.Context, req TReq) (TResp, *apierror.APIError)

/*
Defines the details of a specific API operation. This will be used to generate the OpenAPI spec.
Consequently, consider this public data.
*/
type APIEndpoint[TReq, TResp any] struct {
	Title             string                                                                        `json:"title" yaml:"title"`
	Method            string                                                                        `json:"method" yaml:"method"`
	Route             string                                                                        `json:"route" yaml:"route"`
	ContentType       string                                                                        `json:"content_type" yaml:"content_type"`
	Request           TReq                                                                          `json:"-" yaml:"-"`
	Response          TResp                                                                         `json:"-" yaml:"-"`
	SuccessStatusCode int                                                                           `json:"success_status_code" yaml:"success_status_code"`
	Public            bool                                                                          `json:"-" yaml:"-"`
	Preview           bool                                                                          `json:"-" yaml:"-"`
	ServiceHandler    func(svc any) func(ctx context.Context, req TReq) (TResp, *apierror.APIError) `json:"-" yaml:"-"`
	Extras            APIEndpointExtras                                                             `json:"-" yaml:"-"`
	MinVersion        *version.APIVersion                                                           `json:"-" yaml:"-"`
	// ObjectType identifies the API resource type this endpoint operates on.
	// Used for version transformations. Only endpoints with an ObjectType get transformations applied.
	ObjectType constants.ObjectType `json:"-" yaml:"-"`
	// LocationFunc returns the Location header value for 201 Created responses.
	LocationFunc func(TResp) string `json:"-" yaml:"-"`
	// IncludeConfig declares which sub-objects can be expanded via the include
	// query parameter. When nil, no include support is provided (zero overhead).
	IncludeConfig *IncludeConfig `json:"-" yaml:"-"`

	group               *APIEndpointGroup
	service             any
	middleware          func(http.HandlerFunc) http.HandlerFunc
	bindOnce            sync.Once
	httpHandler         http.HandlerFunc
	boundServiceHandler func(ctx context.Context, req TReq) (TResp, *apierror.APIError)
	// EndpointType is the reflect.Type of the concrete *XxxEndpoint struct that
	// produced this APIEndpoint. It is used by the OpenAPI generator to resolve
	// the operation description from the Go doc comment on that struct.
	EndpointType reflect.Type
}

func (e *APIEndpoint[TReq, TResp]) Materialize() APIEndpointer {
	return e
}

func (e *APIEndpoint[TReq, TResp]) GetMethod() string {
	return e.Method
}

func (e *APIEndpoint[TReq, TResp]) GetRoute() string {
	return e.Route
}

func (e *APIEndpoint[TReq, TResp]) IsPublic() bool {
	return e.Public
}

func (e *APIEndpoint[TReq, TResp]) WithService(g *APIEndpointGroup, svc any) *APIEndpoint[TReq, TResp] {
	e.group = g
	e.service = svc
	return e
}

func (e *APIEndpoint[TReq, TResp]) WithMiddleware(mw func(http.HandlerFunc) http.HandlerFunc) *APIEndpoint[TReq, TResp] {
	e.middleware = mw
	return e
}

// WithDocSource records the concrete endpoint struct type so the OpenAPI
// generator can resolve the operation description from its Go doc comment.
// Pass the receiver of Materialize() (e.g. `.WithDocSource(e)`).
func (e *APIEndpoint[TReq, TResp]) WithDocSource(source any) *APIEndpoint[TReq, TResp] {
	if source != nil {
		t := reflect.TypeOf(source)
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		e.EndpointType = t
	}
	return e
}

func (e *APIEndpoint[TReq, TResp]) GetHandler() http.HandlerFunc {
	e.bindOnce.Do(func() {
		e.boundServiceHandler = e.ServiceHandler(e.service)
		e.httpHandler = e.Execute
		if e.middleware != nil {
			e.httpHandler = e.middleware(e.httpHandler)
		}
	})
	return e.httpHandler
}

func (e *APIEndpoint[TReq, TResp]) Execute(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check minimum version requirement
	if e.MinVersion != nil {
		if requestVersion, ok := appctx.GetAPIVersionFromContext(ctx); ok {
			if requestVersion.Before(*e.MinVersion) {
				apiErr := apierror.NewAPIVersionTooOldError(requestVersion.String(), e.MinVersion.String())
				httptransport.RespondWithAPIError(ctx, w, apiErr)
				return
			}
		}
	}

	if rl, ok := appctx.GetRequestLog(ctx); ok {
		if e.Extras.SkipRequestLogging {
			rl.SkipSave = true
		}
		if e.Extras.ShieldResponseBody {
			rl.ShieldResponseBody = true
		}
	}

	span := trace.SpanFromContext(ctx)

	if idempotencyKey := r.Header.Get(header.IdempotencyKeyHeader); idempotencyKey != "" {
		ctx = appctx.WithIdempotencyKey(ctx, idempotencyKey)
	} else if rl, ok := appctx.GetRequestLog(ctx); ok && rl != nil && rl.ID != "" {
		ctx = appctx.WithIdempotencyKey(ctx, rl.ID)
	}

	var req TReq
	req = httptransport.AllocIfPtr(req)

	if err := httptransport.BindFromHeaders(r, any(req)); err != nil {
		apiErr, ok := err.(*apierror.APIError)
		if !ok {
			apiErr = apierror.NewValidationError(err.Error())
		}
		tracing.RecordControllerError(span, apiErr)
		if span.IsRecording() {
			span.SetAttributes(attribute.String(httptransport.AttrErrorType, "header_binding"))
		}
		httptransport.RespondWithAPIError(ctx, w, apiErr)
		return
	}
	if err := httptransport.BindFromPath(r, any(req)); err != nil {
		apiErr, ok := err.(*apierror.APIError)
		if !ok {
			apiErr = apierror.NewValidationError(err.Error())
		}
		tracing.RecordControllerError(span, apiErr)
		if span.IsRecording() {
			span.SetAttributes(attribute.String(httptransport.AttrErrorType, "path_binding"))
		}
		httptransport.RespondWithAPIError(ctx, w, apiErr)
		return
	}
	if err := httptransport.BindFromQuery(r.URL, any(req)); err != nil {
		apiErr, ok := err.(*apierror.APIError)
		if !ok {
			apiErr = apierror.NewValidationError(err.Error())
		}
		tracing.RecordControllerError(span, apiErr)
		if span.IsRecording() {
			span.SetAttributes(attribute.String(httptransport.AttrErrorType, "query_binding"))
		}
		httptransport.RespondWithAPIError(ctx, w, apiErr)
		return
	}

	if apiErr := httptransport.RejectUnknownQueryParams(r.URL, any(req), e.IncludeConfig != nil); apiErr != nil {
		tracing.RecordControllerError(span, apiErr)
		if span.IsRecording() {
			span.SetAttributes(attribute.String(httptransport.AttrErrorType, "unknown_query_parameter"))
		}
		httptransport.RespondWithAPIError(ctx, w, apiErr)
		return
	}

	// Parse and validate include parameters
	requestedIncludes, apiErr := e.parseIncludeParams(r)
	if apiErr != nil {
		tracing.RecordControllerError(span, apiErr)
		if span.IsRecording() {
			span.SetAttributes(attribute.String(httptransport.AttrErrorType, "include_validation"))
		}
		httptransport.RespondWithAPIError(ctx, w, apiErr)
		return
	}

	// Buffer JSON bodies once for decode, nullable:"false" validation, and optional request logging.
	var jsonBodyBytes []byte
	if !e.Extras.SkipRequestBodyParsing && httptransport.ShouldDecodeBody(r) {
		const maxJSONBodyBytes = 1 << 20 // 1 MiB
		bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, maxJSONBodyBytes))
		_ = r.Body.Close()
		if err != nil {
			apiErr := apierror.NewValidationError(fmt.Sprintf("Failed to read request body: %v", err))
			tracing.RecordControllerError(span, apiErr)
			if span.IsRecording() {
				span.SetAttributes(attribute.String(httptransport.AttrErrorType, "body_read"))
			}
			httptransport.RespondWithAPIError(ctx, w, apiErr)
			return
		}
		jsonBodyBytes = bodyBytes
		if !e.Extras.ShieldRequestBody {
			if rl, ok := appctx.GetRequestLog(ctx); ok && len(bodyBytes) > 0 {
				const maxBodyLogSize = 256 << 10 // 256 KB
				if len(bodyBytes) > maxBodyLogSize {
					s := fmt.Sprintf(`{"_truncated":true,"_original_size":%d}`, len(bodyBytes))
					rl.BodyJSON = &s
				} else {
					s := string(bodyBytes)
					rl.BodyJSON = &s
				}
			}
		}
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	if e.Extras.SkipRequestBodyParsing {
		if err := httptransport.BindRawBody(r, any(req)); err != nil {
			apiErr, ok := err.(*apierror.APIError)
			if !ok {
				apiErr = apierror.NewValidationError(err.Error())
			}
			tracing.RecordControllerError(span, apiErr)
			if span.IsRecording() {
				span.SetAttributes(attribute.String(httptransport.AttrErrorType, "raw_body_binding"))
			}
			httptransport.RespondWithAPIError(ctx, w, apiErr)
			return
		}
	} else if httptransport.ShouldDecodeBody(r) {
		bytesForNull := jsonBodyBytes

		// Transform request body if versioned and ObjectType is set
		if e.ObjectType != "" {
			if requestVersion, ok := appctx.GetAPIVersionFromContext(ctx); ok {
				if !requestVersion.Equal(version.Latest) {
					var err error
					r, err = e.transformRequestBody(r, requestVersion, version.Latest)
					if err != nil {
						apiErr := apierror.NewValidationError(err.Error())
						tracing.RecordControllerError(span, apiErr)
						if span.IsRecording() {
							span.SetAttributes(attribute.String(httptransport.AttrErrorType, "version_transform"))
						}
						httptransport.RespondWithAPIError(ctx, w, apiErr)
						return
					}
					tb, err := io.ReadAll(r.Body)
					_ = r.Body.Close()
					if err != nil {
						apiErr := apierror.NewValidationError(fmt.Sprintf("Failed to read request body: %v", err))
						tracing.RecordControllerError(span, apiErr)
						if span.IsRecording() {
							span.SetAttributes(attribute.String(httptransport.AttrErrorType, "body_read"))
						}
						httptransport.RespondWithAPIError(ctx, w, apiErr)
						return
					}
					bytesForNull = tb
					r.Body = io.NopCloser(bytes.NewReader(tb))
				}
			}
		}

		if err := httptransport.DecodeJSONInto(any(req), r, true); err != nil {
			var apiErr *apierror.APIError
			if !errors.As(err, &apiErr) {
				apiErr = apierror.NewValidationError(err.Error())
			}
			tracing.RecordControllerError(span, apiErr)
			if span.IsRecording() {
				span.SetAttributes(attribute.String(httptransport.AttrErrorType, "json_decode"))
			}
			httptransport.RespondWithAPIError(ctx, w, apiErr)
			return
		}

		validate.ApplyExplicitNulls(bytesForNull, any(req))
		validate.ApplySlicePresenceFlags(bytesForNull, any(req))

		if apiErr := validate.RejectExplicitJSONNulls(bytesForNull, any(req)); apiErr != nil {
			tracing.RecordControllerError(span, apiErr)
			if span.IsRecording() {
				span.SetAttributes(attribute.String(httptransport.AttrErrorType, "json_null_validation"))
			}
			httptransport.RespondWithAPIError(ctx, w, apiErr)
			return
		}
	}

	if r.Method == http.MethodPatch && len(jsonBodyBytes) > 0 {
		if apiErr := validate.RejectEmptyPatchBody(jsonBodyBytes, any(req)); apiErr != nil {
			tracing.RecordControllerError(span, apiErr)
			if span.IsRecording() {
				span.SetAttributes(attribute.String(httptransport.AttrErrorType, "empty_patch_validation"))
			}
			httptransport.RespondWithAPIError(ctx, w, apiErr)
			return
		}
	}

	if apiErr := httptransport.ValidateEnumFields(any(req)); apiErr != nil {
		tracing.RecordControllerError(span, apiErr)
		if span.IsRecording() {
			span.SetAttributes(attribute.String(httptransport.AttrErrorType, "enum_validation"))
		}
		httptransport.RespondWithAPIError(ctx, w, apiErr)
		return
	}

	if apiErr := validate.Validate(any(req)); apiErr != nil {
		tracing.RecordControllerError(span, apiErr)
		if span.IsRecording() {
			span.SetAttributes(attribute.String(httptransport.AttrErrorType, "validation"))
		}
		httptransport.RespondWithAPIError(ctx, w, apiErr)
		return
	}

	if requestedIncludes != nil {
		ctx = appctx.WithRequestedIncludes(ctx, requestedIncludes)
	}

	ctx, responseMeta := appctx.WithHTTPResponseMetadata(ctx)

	resp, err := e.boundServiceHandler(ctx, req)

	// If the client disconnected during processing, report a 499
	// ("the client closed the connection before the server could respond") regardless of
	// what error (if any) the handler returned. There's no point sending a
	// response body — the client is gone — but we still write the status so the
	// logging middleware records the correct code.
	if r.Context().Err() == context.Canceled {
		apiErr := apierror.NewClientClosedRequestError("Client closed the connection.")
		tracing.RecordControllerError(span, apiErr)
		if span.IsRecording() {
			span.SetAttributes(attribute.String(httptransport.AttrErrorType, "client_closed"))
		}
		httptransport.RespondWithAPIError(ctx, w, apiErr)
		return
	}

	if err != nil {
		tracing.RecordControllerError(span, err)
		if span.IsRecording() {
			span.SetAttributes(attribute.String(httptransport.AttrErrorType, "handler"))
		}
		httptransport.RespondWithAPIError(ctx, w, err)
		return
	}

	for _, cookie := range responseMeta.Cookies {
		http.SetCookie(w, cookie)
	}

	if responseMeta.Replayed {
		w.Header().Set(header.IdempotentReplayedHeader, "true")
	}

	if span.IsRecording() {
		span.SetAttributes(attribute.Int(httptransport.AttrHTTPStatusCode, e.SuccessStatusCode))
	}

	var respondOpts []httptransport.RespondOption
	if e.SuccessStatusCode == http.StatusCreated && e.LocationFunc != nil {
		respondOpts = append(respondOpts, httptransport.WithLocation(e.LocationFunc(resp)))
	}

	// File-download response (e.g. Excel export).
	if fd, ok := any(resp).(*httptransport.FileDownload); ok {
		httptransport.RespondWithFile(ctx, w, e.SuccessStatusCode, fd, respondOpts...)
		return
	}

	// Apply include transform: collapse unexpanded sub-objects to {id, object}.
	if e.IncludeConfig != nil {
		e.respondWithIncludeTransform(ctx, w, resp, requestedIncludes, respondOpts...)
		return
	}
	httptransport.RespondWithJSON(ctx, w, e.SuccessStatusCode, resp, respondOpts...)
}

// transformRequestBody reads the request body, applies version transformations to upgrade
// the request from an older version format to the latest format, and returns a new request
// with the transformed body.
func (e *APIEndpoint[TReq, TResp]) transformRequestBody(r *http.Request, from, to version.APIVersion) (*http.Request, error) {
	// Read the original body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	_ = r.Body.Close()

	// If body is empty, just restore and return
	if len(body) == 0 {
		r.Body = io.NopCloser(bytes.NewReader(body))
		return r, nil
	}

	// Parse as JSON
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		// If not valid JSON, restore original body and let downstream handle the error
		r.Body = io.NopCloser(bytes.NewReader(body))
		return r, nil
	}

	// Apply request transformers (upgrade from older version to newer)
	transformed := version.TransformRequest(from, to, e.ObjectType, data)

	// Re-encode to JSON
	transformedBody, err := json.Marshal(transformed)
	if err != nil {
		// On marshal error, restore original body
		r.Body = io.NopCloser(bytes.NewReader(body))
		return r, nil
	}

	// Replace the body with the transformed content
	r.Body = io.NopCloser(bytes.NewReader(transformedBody))
	r.ContentLength = int64(len(transformedBody))

	return r, nil
}

// parseIncludeParams extracts and validates the include query parameter from the
// request. It supports both array-style (?include[]=role) and comma-separated
// (?include=role,account) formats. Returns a set of requested include keys and
// an error if any values are invalid or include is used on an endpoint without
// IncludeConfig.
func (e *APIEndpoint[TReq, TResp]) parseIncludeParams(r *http.Request) (map[string]bool, *apierror.APIError) {
	q := r.URL.Query()
	var raw []string
	if vals, ok := q["include[]"]; ok {
		for _, v := range vals {
			for part := range strings.SplitSeq(v, ",") {
				if s := strings.TrimSpace(part); s != "" {
					raw = append(raw, s)
				}
			}
		}
	}
	if vals, ok := q["include"]; ok {
		for _, v := range vals {
			for part := range strings.SplitSeq(v, ",") {
				if s := strings.TrimSpace(part); s != "" {
					raw = append(raw, s)
				}
			}
		}
	}

	if len(raw) == 0 {
		return nil, nil
	}

	if e.IncludeConfig == nil {
		return nil, apierror.NewParameterInvalidError(
			"This endpoint does not support the include parameter.",
			"include[]",
		)
	}

	allowed := make(map[string]bool, len(e.IncludeConfig.Fields))
	for _, f := range e.IncludeConfig.Fields {
		allowed[f.Key] = true
	}

	requested := make(map[string]bool, len(raw))
	for _, v := range raw {
		if !allowed[v] {
			return nil, apierror.NewParameterInvalidError(
				fmt.Sprintf("Invalid include value '%s'. Allowed values: %s",
					v, strings.Join(e.IncludeConfig.AllowedKeys(), ", ")),
				"include[]",
			)
		}
		requested[v] = true
	}

	return requested, nil
}

// respondWithIncludeTransform marshals the response to a generic map, applies
// the include collapse transform, and writes the result as JSON.
func (e *APIEndpoint[TReq, TResp]) respondWithIncludeTransform(ctx context.Context, w http.ResponseWriter, resp TResp, requested map[string]bool, opts ...httptransport.RespondOption) {
	// Validate that included expandable stubs have required fields populated.
	// Returns 500 rather than serving invalid data to the client.
	if err := ValidateExpandableFields(resp, requested, e.IncludeConfig); err != nil {
		apiErr := apierror.NewInvariantViolationError(err.Error())
		httptransport.RespondWithAPIError(ctx, w, apiErr)
		return
	}

	raw, err := json.Marshal(resp)
	if err != nil {
		httptransport.RespondWithJSON(ctx, w, e.SuccessStatusCode, resp, opts...)
		return
	}

	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		httptransport.RespondWithJSON(ctx, w, e.SuccessStatusCode, resp, opts...)
		return
	}

	transformed := CollapseUnexpanded(data, e.IncludeConfig, requested)

	httptransport.RespondWithJSON(ctx, w, e.SuccessStatusCode, transformed, opts...)
}

type APIEndpointer interface {
	Materialize() APIEndpointer
	GetMethod() string
	GetRoute() string
	IsPublic() bool
	GetHandler() http.HandlerFunc
}
