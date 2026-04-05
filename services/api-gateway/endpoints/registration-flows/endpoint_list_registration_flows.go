package registrationflowep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ListRegistrationFlowsRequest is the request to list registration flows with optional filters.
type ListRegistrationFlowsRequest struct {
	apiresource.PaginationRequest
}

type ListRegistrationFlowsEndpoint struct{}

func (e *ListRegistrationFlowsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListRegistrationFlowsRequest, *apiresource.List[apiresource.RegistrationFlow]] {
	return &apiendpoint.APIEndpoint[*ListRegistrationFlowsRequest, *apiresource.List[apiresource.RegistrationFlow]]{
		Title:             "List Registration Flows",
		Description:       "Returns a paginated list of registration flows for the current account.",
		Method:            http.MethodGet,
		Route:             "/v1/sales/registration-flows",
		Request:           &ListRegistrationFlowsRequest{},
		Response:          &apiresource.List[apiresource.RegistrationFlow]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListRegistrationFlowsRequest) (*apiresource.List[apiresource.RegistrationFlow], *apierror.APIError) {
			return svc.(RegistrationFlowSvc).ListRegistrationFlows
		},
	}
}
