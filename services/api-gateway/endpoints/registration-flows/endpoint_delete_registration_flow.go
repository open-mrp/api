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
	"github.com/open-mrp/apikit/object"
)

// Request to delete a registration flow.
type DeleteRegistrationFlowRequest struct {
	// Registration flow ID.
	RegistrationFlowID string `path:"id" validate:"required"`
}

// Deletes a registration flow.
type DeleteRegistrationFlowEndpoint struct{}

func (e *DeleteRegistrationFlowEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteRegistrationFlowRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteRegistrationFlowRequest, *object.Deleted]{
		Title:             "Delete Registration Flow",
		Method:            http.MethodDelete,
		Route:             "/v1/sales/registration-flows/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeRegistrationFlow,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainAccount, Action: types.ActionUpdate},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteRegistrationFlowRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(RegistrationFlowSvc).DeleteRegistrationFlow, constants.ObjectTypeRegistrationFlow, "id")
		},
	})
}
