package apiendpoint

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
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
	AllowUnknownJSONFields bool `json:"allow_unknown_json_fields" yaml:"allow_unknown_json_fields"`
	SkipRequestBodyParsing bool `json:"skip_request_body_parsing" yaml:"skip_request_body_parsing"`
}

type ServiceHandler[TReq, TResp any] func(ctx context.Context, req TReq) (TResp, *apierror.APIError)

/*
Defines the details of a specific API operation. This will be used to generate the OpenAPI spec.
Consequently, consider this public data.
*/
type APIEndpoint[TReq, TResp any] struct {
	Title             string                                                                        `json:"title" yaml:"title"`
	Description       string                                                                        `json:"description" yaml:"description"`
	Method            string                                                                        `json:"method" yaml:"method"`
	Route             string                                                                        `json:"route" yaml:"route"`
	ContentType       string                                                                        `json:"content_type" yaml:"content_type"`
	Request           TReq                                                                          `json:"-" yaml:"-"`
	Response          TResp                                                                         `json:"-" yaml:"-"`
	SuccessStatusCode int                                                                           `json:"success_status_code" yaml:"success_status_code"`
	Public            bool                                                                          `json:"-" yaml:"-"`
	ServiceHandler    func(svc any) func(ctx context.Context, req TReq) (TResp, *apierror.APIError) `json:"-" yaml:"-"`
	Extras            APIEndpointExtras                                                             `json:"-" yaml:"-"`
	MinVersion        *version.APIVersion                                                           `json:"-" yaml:"-"`
	// ObjectType identifies the API resource type this endpoint operates on.
	// Used for version transformations. Only endpoints with an ObjectType get transformations applied.
	ObjectType constants.ObjectType `json:"-" yaml:"-"`

	group               *APIEndpointGroup
	service             any
	middleware          func(http.HandlerFunc) http.HandlerFunc
	bindOnce            sync.Once
	httpHandler         http.HandlerFunc
	boundServiceHandler func(ctx context.Context, req TReq) (TResp, *apierror.APIError)
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

func (e *APIEndpoint[TReq, TResp]) WithService(g *APIEndpointGroup, svc any) *APIEndpoint[TReq, TResp] {
	e.group = g
	e.service = svc
	return e
}

func (e *APIEndpoint[TReq, TResp]) WithMiddleware(mw func(http.HandlerFunc) http.HandlerFunc) *APIEndpoint[TReq, TResp] {
	e.middleware = mw
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

	span := trace.SpanFromContext(ctx)
	paginationParams, err := httptransport.ApplyPagination(r)
	if err != nil {
		tracing.RecordControllerError(span, err)
		if span.IsRecording() {
			span.SetAttributes(attribute.String(httptransport.AttrErrorType, "pagination_parsing"))
		}
		httptransport.RespondWithAPIError(ctx, w, err)
		return
	}

	ctx = appctx.WithPagination(ctx, paginationParams)

	if idempotencyKey := r.Header.Get(header.IdempotencyKeyHeader); idempotencyKey != "" {
		ctx = appctx.WithIdempotencyKey(ctx, idempotencyKey)
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
				}
			}
		}

		if err := httptransport.DecodeJSONInto(any(req), r, !e.Extras.AllowUnknownJSONFields); err != nil {
			apiErr, ok := err.(*apierror.APIError)
			if !ok {
				apiErr = apierror.NewValidationError(err.Error())
			}
			tracing.RecordControllerError(span, apiErr)
			if span.IsRecording() {
				span.SetAttributes(attribute.String(httptransport.AttrErrorType, "json_decode"))
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

	ctx, responseMeta := appctx.WithHTTPResponseMetadata(ctx)

	resp, err := e.boundServiceHandler(ctx, req)

	// If the client disconnected during processing, report a 499 regardless of
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
	httptransport.RespondWithJSON(ctx, w, e.SuccessStatusCode, resp)
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

type APIEndpointer interface {
	Materialize() APIEndpointer
	GetMethod() string
	GetRoute() string
	GetHandler() http.HandlerFunc
}
