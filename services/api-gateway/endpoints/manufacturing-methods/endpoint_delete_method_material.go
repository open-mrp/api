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

// Request to delete a material.
type DeleteMethodMaterialRequest struct {
	// Manufacturing method ID.
	ManufacturingMethodID string `path:"manufacturing_method_id" validate:"required"`
	// Material ID.
	ID string `path:"id" validate:"required"`
}

// Deletes a material of a draft manufacturing method.
//
// Only a `draft` method can change (409 `invalid_state_transition`).
type DeleteMethodMaterialEndpoint struct{}

func (e *DeleteMethodMaterialEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteMethodMaterialRequest, *object.Deleted] {
	return &apiendpoint.APIEndpoint[*DeleteMethodMaterialRequest, *object.Deleted]{
		Title:             "Delete Method Material",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             materialsRoute + "/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainManufacturingMethods, Action: types.ActionUpdate}}},
		ObjectType:        constants.ObjectTypeMethodMaterial,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteMethodMaterialRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(ManufacturingMethodSvc).DeleteMethodMaterial, constants.ObjectTypeMethodMaterial, "id")
		},
	}
}
