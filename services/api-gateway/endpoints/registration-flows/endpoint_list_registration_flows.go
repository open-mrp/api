package registrationflowep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list registration flows.
type ListRegistrationFlowsRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of registration flows for the current account.
type ListRegistrationFlowsEndpoint struct{}

func (e *ListRegistrationFlowsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListRegistrationFlowsRequest, *apiresource.List[apiresource.RegistrationFlow]] {
	return (&apiendpoint.APIEndpoint[*ListRegistrationFlowsRequest, *apiresource.List[apiresource.RegistrationFlow]]{
		Title:             "List Registration Flows",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/registration-flows",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListRegistrationFlowsRequest) (*apiresource.List[apiresource.RegistrationFlow], *apierror.APIError) {
			return svc.(RegistrationFlowSvc).ListRegistrationFlows
		},
	})
}
