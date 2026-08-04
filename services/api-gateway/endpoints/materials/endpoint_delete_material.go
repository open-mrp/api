package materialep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a material.
type DeleteMaterialRequest struct {
	// ID of the material to delete.
	ItemID string `path:"id" validate:"required"`
}

// Deletes a material.
//
// This is a soft delete: the material and the catalog item behind it stop being returned by other endpoints, but the records are retained. The response is the material as it stood immediately before deletion, and deleting an already-deleted material returns an error.
type DeleteMaterialEndpoint struct{}

func (e *DeleteMaterialEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteMaterialRequest, *apiresource.Material] {
	return (&apiendpoint.APIEndpoint[*DeleteMaterialRequest, *apiresource.Material]{
		Title:               "Delete Material",
		Method:              http.MethodDelete,
		ContentType:         "application/json",
		Route:               "/v1/catalog/materials/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMaterials, Action: types.ActionDelete}, {Domain: types.PermissionDomainCustomers, Action: types.ActionUpdate}, {Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteMaterialRequest) (*apiresource.Material, *apierror.APIError) {
			return svc.(MaterialSvc).DeleteMaterial
		},
	})
}
