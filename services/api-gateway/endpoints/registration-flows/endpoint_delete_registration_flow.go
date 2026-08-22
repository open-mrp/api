package registrationflowep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to delete a registration flow.
type DeleteRegistrationFlowRequest struct {
	// Registration flow ID.
	RegistrationFlowID string `path:"id" validate:"required"`
}

// Deletes a registration flow.
type DeleteRegistrationFlowEndpoint struct{}

func (e *DeleteRegistrationFlowEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteRegistrationFlowRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteRegistrationFlowRequest, *apiresource.EmptyResource]{
		Title:             "Delete Registration Flow",
		Method:            http.MethodDelete,
		Route:             "/v1/sales/registration-flows/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainAccount, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteRegistrationFlowRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(RegistrationFlowSvc).DeleteRegistrationFlow
		},
	})
}
