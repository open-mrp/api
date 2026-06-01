package apiendpoint

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/augno/api/services/api-gateway/internal/header"
	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/patch"
	"github.com/augno/api/shared/redact"
	"github.com/augno/api/shared/tracing"
	"github.com/augno/api/shared/validate"
	"github.com/augno/api/shared/version"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type APIEndpointExtras struct {
	SkipRequestBodyParsing bool `json:"skip_request_body_parsing" yaml:"skip_request_body_parsing"`
	SkipRequestLogging     bool `json:"skip_request_logging" yaml:"skip_request_logging"`
}

type ServiceHandler[TReq, TResp any] = func(ctx context.Context, req TReq) (TResp, *apierror.APIError)

/*
Defines the details of a specific API operation. This will be used to generate the OpenAPI spec.
Consequently, consider this public data.
*/
type APIEndpoint[TReq, TResp any] struct {
	Title  string `json:"title" yaml:"title"`
	Method string `json:"method" yaml:"method"`
	Route  string `json:"route" yaml:"route"`
	// SDKMethodKey overrides the generated Stainless method key for this endpoint. When empty, codegen derives the key from the route + method.
	SDKMethodKey      string                                    `json:"sdk_method_key,omitempty" yaml:"sdk_method_key,omitempty"`
	ContentType       string                                    `json:"content_type" yaml:"content_type"`
	SuccessStatusCode int                                       `json:"success_status_code" yaml:"success_status_code"`
	Public            bool                                      `json:"-" yaml:"-"`
	Preview           bool                                      `json:"-" yaml:"-"`
	ServiceHandler    func(svc any) ServiceHandler[TReq, TResp] `json:"-" yaml:"-"`
	Extras            APIEndpointExtras                         `json:"-" yaml:"-"`
	MinVersion        *version.APIVersion                       `json:"-" yaml:"-"`
	// ObjectType identifies the API resource type this endpoint operates on. Used for version transformations. Only endpoints with an ObjectType get transformations applied.
	ObjectType constants.ObjectType `json:"-" yaml:"-"`
	// LocationFunc returns the Location header value for 201 Created responses.
	LocationFunc func(TResp) string `json:"-" yaml:"-"`
	// IncludeConfig declares which sub-objects can be expanded via the include query parameter. When nil, no include support is provided (zero overhead).
	IncludeConfig *IncludeConfig `json:"-" yaml:"-"`

	group               *APIEndpointGroup
	service             any
	middleware          func(http.HandlerFunc) http.HandlerFunc
	bindOnce            sync.Once
	sensitiveOnce       sync.Once
	httpHandler         http.HandlerFunc
	boundServiceHandler ServiceHandler[TReq, TResp]
	// EndpointType is the reflect.Type of the concrete *XxxEndpoint struct that produced this APIEndpoint. It is used by the OpenAPI generator to resolve the operation description from the Go doc comment on that struct.
	EndpointType reflect.Type

	sensitiveReqPaths  map[string]bool
	sensitiveRespPaths map[string]bool
}

func coercePlainExecuteError(err error) *apierror.APIError {
	if apiErr, ok := errors.AsType[*apierror.APIError](err); ok {
		return apiErr
	}
	return apierror.NewValidationError(err.Error())
}

func recordAndRespondAPIError(ctx context.Context, w http.ResponseWriter, span trace.Span, errTypeAttr string, apiErr *apierror.APIError) {
	tracing.RecordControllerError(span, apiErr)
	if span.IsRecording() {
		span.SetAttributes(attribute.String(httptransport.AttrErrorType, errTypeAttr))
	}
	httptransport.RespondWithAPIError(ctx, w, apiErr)
}

func (e *APIEndpoint[TReq, TResp]) GetRequestType() reflect.Type {
	return reflect.TypeFor[TReq]()
}

func (e *APIEndpoint[TReq, TResp]) GetResponseType() reflect.Type {
	return reflect.TypeFor[TResp]()
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

func (e *APIEndpoint[TReq, TResp]) IsServiceBound() bool {
	return e.service != nil
}

func (e *APIEndpoint[TReq, TResp]) WithMiddleware(mw func(http.HandlerFunc) http.HandlerFunc) *APIEndpoint[TReq, TResp] {
	e.middleware = mw
	return e
}

// From calls source.Materialize() and sets EndpointType so the OpenAPI generator can attach the wrapper type's Go doc to the route.
func From[TReq, TResp any, T interface {
	Materialize() *APIEndpoint[TReq, TResp]
}](source T) *APIEndpoint[TReq, TResp] {
	ep := source.Materialize()
	t := reflect.TypeOf(source)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	ep.EndpointType = t
	return ep
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

func (e *APIEndpoint[TReq, TResp]) ensureSensitivePaths() {
	e.sensitiveOnce.Do(func() {
		e.sensitiveReqPaths = redact.SensitiveFields(reflect.TypeFor[TReq]())
		e.sensitiveRespPaths = redact.SensitiveFields(reflect.TypeFor[TResp]())
	})
}

func (e *APIEndpoint[TReq, TResp]) Execute(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	e.ensureSensitivePaths()

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
		if len(e.sensitiveRespPaths) > 0 {
			rl.SensitiveResponseFields = maps.Clone(e.sensitiveRespPaths)
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

	includesEnabled := e.IncludeConfig != nil
	if err := httptransport.BindIncomingRequest(r, any(req), includesEnabled); err != nil {
		recordAndRespondAPIError(ctx, w, span, "incoming_request_binding", coercePlainExecuteError(err))
		return
	}

	var (
		includeTree *resourcekit.IncludeNode
		apiErr      *apierror.APIError
	)
	if includesEnabled {
		includeTree, apiErr = e.parseIncludeTree(r)
		if apiErr != nil {
			recordAndRespondAPIError(ctx, w, span, "include_validation", apiErr)
			return
		}
	}

	// Buffer JSON bodies once for decode, null validation, and optional request logging.
	var jsonBodyBytes []byte
	if !e.Extras.SkipRequestBodyParsing && httptransport.ShouldDecodeBody(r) {
		const maxJSONBodyBytes = 1 << 20 // 1 MiB
		bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, maxJSONBodyBytes))
		_ = r.Body.Close()
		if err != nil {
			recordAndRespondAPIError(ctx, w, span, "body_read", apierror.NewValidationError(fmt.Sprintf("Failed to read request body: %v", err)))
			return
		}
		jsonBodyBytes = bodyBytes
		if rl, ok := appctx.GetRequestLog(ctx); ok && len(bodyBytes) > 0 {
			const maxBodyLogSize = 256 << 10 // 256 KB
			if len(bodyBytes) > maxBodyLogSize {
				s := fmt.Sprintf(`{"_truncated":true,"_original_size":%d}`, len(bodyBytes))
				rl.BodyJSON = &s
			} else if len(e.sensitiveReqPaths) > 0 {
				rb := redact.RedactJSON(bodyBytes, e.sensitiveReqPaths)
				if rb != nil {
					s := string(rb)
					rl.BodyJSON = &s
				}
			} else {
				s := string(bodyBytes)
				rl.BodyJSON = &s
			}
		}
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	if e.Extras.SkipRequestBodyParsing {
		if err := httptransport.BindRawBody(r, any(req)); err != nil {
			recordAndRespondAPIError(ctx, w, span, "raw_body_binding", coercePlainExecuteError(err))
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
						recordAndRespondAPIError(ctx, w, span, "version_transform", apierror.NewValidationError(err.Error()))
						return
					}
					tb, err := io.ReadAll(r.Body)
					_ = r.Body.Close()
					if err != nil {
						recordAndRespondAPIError(ctx, w, span, "body_read", apierror.NewValidationError(fmt.Sprintf("Failed to read request body: %v", err)))
						return
					}
					bytesForNull = tb
					r.Body = io.NopCloser(bytes.NewReader(tb))
				}
			}
		}

		if err := httptransport.DecodeJSONInto(any(req), r, true); err != nil {
			if errors.Is(err, patch.ErrExplicitNull) {
				if name, ok := patch.ExplicitNullField(bytesForNull, any(req)); ok {
					recordAndRespondAPIError(ctx, w, span, "json_decode", apierror.NewInvalidFormatError(fmt.Sprintf("Field '%s' cannot be null.", name), name))
					return
				}
			}
			recordAndRespondAPIError(ctx, w, span, "json_decode", coercePlainExecuteError(err))
			return
		}

		patch.ApplyPtrFieldNulls(bytesForNull, any(req))
		validate.ApplySlicePresenceFlags(bytesForNull, any(req))

		if apiErr := validate.RejectExplicitJSONNulls(bytesForNull, any(req)); apiErr != nil {
			recordAndRespondAPIError(ctx, w, span, "json_null_validation", apiErr)
			return
		}
	}

	if r.Method == http.MethodPatch && len(jsonBodyBytes) > 0 {
		if apiErr := validate.RejectEmptyPatchBody(jsonBodyBytes, any(req)); apiErr != nil {
			recordAndRespondAPIError(ctx, w, span, "empty_patch_validation", apiErr)
			return
		}
	}

	if apiErr := httptransport.ValidateEnumFields(any(req)); apiErr != nil {
		recordAndRespondAPIError(ctx, w, span, "enum_validation", apiErr)
		return
	}

	if apiErr := validate.Validate(any(req)); apiErr != nil {
		recordAndRespondAPIError(ctx, w, span, "validation", apiErr)
		return
	}

	if includesEnabled {
		ctx = resourcekit.WithLoadCache(ctx)
		ctx = resourcekit.WithLoadMeta(ctx)
	}

	ctx, responseMeta := appctx.WithHTTPResponseMetadata(ctx)
	ctx = appctx.WithRequestURL(ctx, r.URL)

	resp, err := e.boundServiceHandler(ctx, req)

	// If the client disconnected during processing, report a 499 ("the client closed the connection before the server could respond") regardless of what error (if any) the handler returned. There's no point sending a response body — the client is gone — but we still write the status so the logging middleware records the correct code.
	if r.Context().Err() == context.Canceled {
		recordAndRespondAPIError(ctx, w, span, "client_closed", apierror.NewClientClosedRequestError("Client closed the connection."))
		return
	}

	if err != nil {
		recordAndRespondAPIError(ctx, w, span, "handler", err)
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

	if includesEnabled && e.ObjectType != "" {
		var roots []any
		if e.IncludeConfig.ExtractRoots != nil {
			roots = e.IncludeConfig.ExtractRoots(any(resp))
		} else {
			roots = extractIncludeRoots(any(resp))
		}
		if len(roots) > 0 && includeTree.HasChildren() {
			if apiErr := resourcekit.ResolveIncludes(ctx, roots, e.ObjectType, includeTree); apiErr != nil {
				recordAndRespondAPIError(ctx, w, span, "include_resolution", apiErr)
				return
			}
		}
	}
	httptransport.RespondWithJSON(ctx, w, e.SuccessStatusCode, resp, respondOpts...)
}

// transformRequestBody reads the request body, applies version transformations to upgrade the request from an older version format to the latest format, and returns a new request with the transformed body.
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

// collectIncludeQueryValues extracts raw include keys from `?include[]=` and `?include=` query params. Both array-style and comma-separated values are supported.
func collectIncludeQueryValues(r *http.Request) []string {
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
	return raw
}

func (e *APIEndpoint[TReq, TResp]) parseIncludeTree(r *http.Request) (*resourcekit.IncludeNode, *apierror.APIError) {
	raw := collectIncludeQueryValues(r)
	if len(raw) == 0 {
		return resourcekit.NewIncludeTree(), nil
	}
	if e.IncludeConfig == nil {
		return nil, apierror.NewParameterInvalidError(
			"This endpoint does not support the include parameter.",
			"include[]",
		)
	}
	if e.ObjectType != "" {
		if def := resourcekit.Lookup(e.ObjectType); def == nil {
			return nil, apierror.NewInvariantViolationError(fmt.Sprintf(
				"No resourcekit.Definition registered for %s",
				e.ObjectType,
			))
		}
	}
	allowed := e.IncludeConfig.AllowedKeys()
	allowedSet := make(map[string]bool, len(allowed))
	for _, k := range allowed {
		allowedSet[k] = true
	}
	for _, v := range raw {
		if !allowedSet[v] {
			return nil, apierror.NewParameterInvalidError(
				fmt.Sprintf("Invalid include value '%s'. Allowed values: %s",
					v, strings.Join(allowed, ", ")),
				"include[]",
			)
		}
	}
	return resourcekit.ParseIncludeTree(raw), nil
}

// extractIncludeRoots reflects on `resp` to find the resource pointers the resolver should walk. Handles two shapes: a `*Resource` returned directly, or a `*List[Resource]` (or `*List[*Resource]`) whose `Data` slice holds the roots. Slice elements must be addressable so the resolver can mutate them in place; values returned by handlers via `apiresource.NewList` already meet that requirement.
func extractIncludeRoots(resp any) []any {
	if resp == nil {
		return nil
	}
	v := reflect.ValueOf(resp)
	if !v.IsValid() {
		return nil
	}
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return nil
	}
	elem := v.Elem()
	if elem.Kind() != reflect.Struct {
		return nil
	}
	if dataField := elem.FieldByName("Data"); dataField.IsValid() && dataField.Kind() == reflect.Slice {
		roots := make([]any, 0, dataField.Len())
		for i := 0; i < dataField.Len(); i++ {
			item := dataField.Index(i)
			switch item.Kind() {
			case reflect.Pointer:
				if !item.IsNil() {
					roots = append(roots, item.Interface())
				}
			default:
				if item.CanAddr() {
					roots = append(roots, item.Addr().Interface())
				}
			}
		}
		return roots
	}
	return []any{resp}
}

type APIEndpointer interface {
	GetMethod() string
	GetRoute() string
	IsPublic() bool
	GetHandler() http.HandlerFunc
	IsServiceBound() bool
	GetRequestType() reflect.Type
	GetResponseType() reflect.Type
}
