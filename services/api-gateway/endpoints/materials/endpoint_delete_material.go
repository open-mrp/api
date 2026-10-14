package materialep

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

// Request to delete a material.
type DeleteMaterialRequest struct {
	// ID of the material to delete.
	ItemID string `path:"id" validate:"required"`
}

// Deletes a material.
//
// This is a soft delete: the material and the catalog item behind it stop being returned by other endpoints, but the records are retained. The response is the material as it stood immediately before deletion, and deleting an already-deleted material returns an error.
type DeleteMaterialEndpoint struct{}

func (e *DeleteMaterialEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteMaterialRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteMaterialRequest, *object.Deleted]{
		Title:             "Delete Material",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/catalog/materials/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMaterials, Action: types.ActionDelete}}, Counterparty: authz.Counterparties(types.ActionUpdate)},
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteMaterialRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(MaterialSvc).DeleteMaterial, constants.ObjectTypeMaterial, "id")
		},
	})
}
