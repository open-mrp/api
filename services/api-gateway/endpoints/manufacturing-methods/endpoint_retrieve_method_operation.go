package manufacturingmethodep

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

// Request to retrieve an operation.
type RetrieveMethodOperationRequest struct {
	// Manufacturing method ID.
	ManufacturingMethodID string `path:"manufacturing_method_id" validate:"required"`
	// Operation ID.
	ID string `path:"id" validate:"required"`
}

// Retrieves an operation of a manufacturing method.
type RetrieveMethodOperationEndpoint struct{}

func (e *RetrieveMethodOperationEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveMethodOperationRequest, *apiresource.MethodOperation] {
	return &apiendpoint.APIEndpoint[*RetrieveMethodOperationRequest, *apiresource.MethodOperation]{
		Title:             "Retrieve Method Operation",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             operationsRoute + "/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainManufacturingMethods, Action: types.ActionRead}}},
		ObjectType:        constants.ObjectTypeMethodOperation,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveMethodOperationRequest) (*apiresource.MethodOperation, *apierror.APIError) {
			return svc.(ManufacturingMethodSvc).GetMethodOperation
		},
		IncludeConfig: operationIncludes(),
	}
}
