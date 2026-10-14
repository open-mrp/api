package registrationflowep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to retrieve a registration flow.
type RetrieveRegistrationFlowRequest struct {
	// Registration flow ID.
	RegistrationFlowID string `path:"id" validate:"required"`
}

// Returns a registration flow by ID.
type RetrieveRegistrationFlowEndpoint struct{}

func (e *RetrieveRegistrationFlowEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveRegistrationFlowRequest, *apiresource.RegistrationFlow] {
	return (&apiendpoint.APIEndpoint[*RetrieveRegistrationFlowRequest, *apiresource.RegistrationFlow]{
		Title:             "Retrieve Registration Flow",
		Method:            http.MethodGet,
		Route:             "/v1/sales/registration-flows/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainAccount, Action: types.ActionRead},
		}},
		ObjectType: constants.ObjectTypeRegistrationFlow,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveRegistrationFlowRequest) (*apiresource.RegistrationFlow, *apierror.APIError) {
			return svc.(RegistrationFlowSvc).GetRegistrationFlow
		},
	})
}
