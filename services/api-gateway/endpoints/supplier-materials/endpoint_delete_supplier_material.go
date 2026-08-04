package suppliermaterialep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a supplier material.
type DeleteSupplierMaterialRequest struct {
	// ID of the supplier the material is linked to.
	SupplierID string `path:"supplier_id" validate:"required"`
	// ID of the material whose supplier link to delete.
	//
	// Supplier materials are addressed by the combination of supplier and material, so this path parameter takes the material's ID.
	MaterialID string `path:"id" validate:"required"`
}

// Deletes a supplier material link.
//
// Returns the link as it looked immediately before deletion. Removing the link does not affect the underlying material or supplier.
type DeleteSupplierMaterialEndpoint struct{}

func (e *DeleteSupplierMaterialEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteSupplierMaterialRequest, *apiresource.SupplierMaterial] {
	return (&apiendpoint.APIEndpoint[*DeleteSupplierMaterialRequest, *apiresource.SupplierMaterial]{
		Title:             "Delete Supplier Material",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/suppliers/{supplier_id}/materials/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteSupplierMaterialRequest) (*apiresource.SupplierMaterial, *apierror.APIError) {
			return svc.(SupplierMaterialSvc).DeleteSupplierMaterial
		},
	})
}
