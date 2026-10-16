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

// Request to list a manufacturing method's materials.
type ListMethodMaterialsRequest struct {
	object.ListRequest
	// Manufacturing method ID.
	ManufacturingMethodID string `path:"manufacturing_method_id" validate:"required"`
}

// Returns a list of a manufacturing method's materials.
//
// Sorted in the order they were added, first to last.
type ListMethodMaterialsEndpoint struct{}

func (e *ListMethodMaterialsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListMethodMaterialsRequest, *apiresource.List[apiresource.MethodMaterial]] {
	return &apiendpoint.APIEndpoint[*ListMethodMaterialsRequest, *apiresource.List[apiresource.MethodMaterial]]{
		Title:             "List Method Materials",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             materialsRoute,
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainManufacturingMethods, Action: types.ActionRead}}},
		ObjectType:        constants.ObjectTypeMethodMaterial,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListMethodMaterialsRequest) (*apiresource.List[apiresource.MethodMaterial], *apierror.APIError) {
			return svc.(ManufacturingMethodSvc).ListMethodMaterials
		},
		IncludeConfig: materialIncludes(),
	}
}
