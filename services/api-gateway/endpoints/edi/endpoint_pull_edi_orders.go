package ediep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to pull EDI orders for the target account.
type PullEDIOrdersRequest struct{}

var samplePullEDIOrdersRequest = &PullEDIOrdersRequest{}

func (*PullEDIOrdersRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(samplePullEDIOrdersRequest)
}

// Triggers EDI order intake for the target account.
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
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainSalesOrders, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *PullEDIOrdersRequest) (*apiresource.MessageResource, *apierror.APIError) {
			return svc.(EDISvc).PullOrders
		},
	})
}
