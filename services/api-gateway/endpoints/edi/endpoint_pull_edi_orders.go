package ediep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to trigger an EDI pull-orders operation.
type PullEDIOrdersRequest struct{}

var examplePullEDIOrdersRequest = &PullEDIOrdersRequest{}

func (*PullEDIOrdersRequest) SchemaExample() any {
	return examplePullEDIOrdersRequest
}

// Triggers an EDI pull-orders operation, pulling orders from FTP and processing invoices via Stedi.
type PullEDIOrdersEndpoint struct{}

func (e *PullEDIOrdersEndpoint) Materialize() *apiendpoint.APIEndpoint[*PullEDIOrdersRequest, *apiresource.MessageResource] {
	return (&apiendpoint.APIEndpoint[*PullEDIOrdersRequest, *apiresource.MessageResource]{
		Title:             "Pull EDI Orders",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/operations/edi/actions/pull-orders",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *PullEDIOrdersRequest) (*apiresource.MessageResource, *apierror.APIError) {
			return svc.(EDISvc).PullOrders
		},
	})
}
