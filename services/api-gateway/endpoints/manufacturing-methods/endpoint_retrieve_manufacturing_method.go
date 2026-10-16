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

// Request to retrieve a manufacturing method.
type RetrieveManufacturingMethodRequest struct {
	// Manufacturing method ID.
	ID string `path:"id" validate:"required"`
}

// Retrieves a manufacturing method.
type RetrieveManufacturingMethodEndpoint struct{}

func (e *RetrieveManufacturingMethodEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveManufacturingMethodRequest, *apiresource.ManufacturingMethod] {
	return &apiendpoint.APIEndpoint[*RetrieveManufacturingMethodRequest, *apiresource.ManufacturingMethod]{
		Title:             "Retrieve Manufacturing Method",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             methodsRoute + "/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainManufacturingMethods, Action: types.ActionRead}}},
		ObjectType:        constants.ObjectTypeManufacturingMethod,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveManufacturingMethodRequest) (*apiresource.ManufacturingMethod, *apierror.APIError) {
			return svc.(ManufacturingMethodSvc).GetManufacturingMethod
		},
		IncludeConfig: methodIncludes(),
	}
}
