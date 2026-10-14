package registrationflowep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list registration flows.
type ListRegistrationFlowsRequest struct {
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
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
		AgentTool:         true,
		Preview:           true,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainAccount, Action: types.ActionRead},
		}},
		ObjectType: constants.ObjectTypeRegistrationFlow,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListRegistrationFlowsRequest) (*apiresource.List[apiresource.RegistrationFlow], *apierror.APIError) {
			return svc.(RegistrationFlowSvc).ListRegistrationFlows
		},
	})
}
