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

// Request to retrieve a material.
type RetrieveMethodMaterialRequest struct {
	// Manufacturing method ID.
	ManufacturingMethodID string `path:"manufacturing_method_id" validate:"required"`
	// Material ID.
	ID string `path:"id" validate:"required"`
}

// Retrieves a material of a manufacturing method.
type RetrieveMethodMaterialEndpoint struct{}

func (e *RetrieveMethodMaterialEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveMethodMaterialRequest, *apiresource.MethodMaterial] {
	return &apiendpoint.APIEndpoint[*RetrieveMethodMaterialRequest, *apiresource.MethodMaterial]{
		Title:             "Retrieve Method Material",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             materialsRoute + "/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainManufacturingMethods, Action: types.ActionRead}}},
		ObjectType:        constants.ObjectTypeMethodMaterial,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveMethodMaterialRequest) (*apiresource.MethodMaterial, *apierror.APIError) {
			return svc.(ManufacturingMethodSvc).GetMethodMaterial
		},
		IncludeConfig: materialIncludes(),
	}
}
