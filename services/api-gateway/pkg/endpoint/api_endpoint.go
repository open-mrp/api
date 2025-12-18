package apiendpoint

import (
	"context"
	"net/http"

	"github.com/augno/api/shared/contracts"
)

type APIEndpointExtras struct {
	AllowUnknownJSONFields bool `json:"allow_unknown_json_fields" yaml:"allow_unknown_json_fields"`
}

type HandlerFunc[TReq, TResp any] func(ctx context.Context, req TReq) (TResp, *contracts.APIError)

/*
Defines the details of a specific API operation. This will be used to generate the OpenAPI spec.
Consequently, consider this public data.
*/
type APIEndpoint[TReq, TResp any] struct {
	Title             string                                  `json:"title" yaml:"title"`
	Description       string                                  `json:"description" yaml:"description"`
	Method            string                                  `json:"method" yaml:"method"`
	Route             string                                  `json:"route" yaml:"route"`
	ContentType       string                                  `json:"content_type" yaml:"content_type"`
	Request           TReq                                    `json:"-" yaml:"-"`
	Response          TResp                                   `json:"-" yaml:"-"`
	SuccessStatusCode int                                     `json:"success_status_code" yaml:"success_status_code"`
	IsPublic          bool                                    `json:"-" yaml:"-"`
	Handler           func(ctrl any) HandlerFunc[TReq, TResp] `json:"-" yaml:"-"`
	Extras            APIEndpointExtras                       `json:"-" yaml:"-"`
}

type BoundEndpoint[TReq, TResp any] struct {
	Spec    APIEndpoint[TReq, TResp]
	Handler HandlerFunc[TReq, TResp]
}

func Bind[TReq, TResp any](spec APIEndpoint[TReq, TResp], ctrl any) BoundEndpoint[TReq, TResp] {
	return BoundEndpoint[TReq, TResp]{
		Spec:    spec,
		Handler: spec.Handler(ctrl),
	}
}

func (e *APIEndpoint[TReq, TResp]) GetMethod() string {
	return e.Method
}

func (e *APIEndpoint[TReq, TResp]) GetRoute() string {
	return e.Route
}

type APIEndpointer interface {
	Materialize() APIEndpointer
	GetMethod() string
	GetRoute() string
	GetHandler() http.HandlerFunc
}
