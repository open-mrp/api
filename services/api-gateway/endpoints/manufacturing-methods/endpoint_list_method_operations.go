package manufacturingmethodep

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

// Request to list a manufacturing method's operations.
type ListMethodOperationsRequest struct {
	object.ListRequest
	// Manufacturing method ID.
	ManufacturingMethodID string `path:"manufacturing_method_id" validate:"required"`
}

// Returns a list of a manufacturing method's operations.
//
// Sorted by `position`, first to last.
type ListMethodOperationsEndpoint struct{}

func (e *ListMethodOperationsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListMethodOperationsRequest, *apiresource.List[apiresource.MethodOperation]] {
	return &apiendpoint.APIEndpoint[*ListMethodOperationsRequest, *apiresource.List[apiresource.MethodOperation]]{
		Title:             "List Method Operations",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             operationsRoute,
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainManufacturingMethods, Action: types.ActionRead}}},
		ObjectType:        constants.ObjectTypeMethodOperation,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListMethodOperationsRequest) (*apiresource.List[apiresource.MethodOperation], *apierror.APIError) {
			return svc.(ManufacturingMethodSvc).ListMethodOperations
		},
		IncludeConfig: operationIncludes(),
	}
}
