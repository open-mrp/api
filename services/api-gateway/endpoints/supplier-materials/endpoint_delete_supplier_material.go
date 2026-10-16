package suppliermaterialep

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
// Returns a deleted stub. Removing the link does not affect the underlying material or supplier.
type DeleteSupplierMaterialEndpoint struct{}

func (e *DeleteSupplierMaterialEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteSupplierMaterialRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteSupplierMaterialRequest, *object.Deleted]{
		Title:             "Delete Supplier Material",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/suppliers/{supplier_id}/materials/{id}",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeSupplierMaterial,
		Public:            false,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteSupplierMaterialRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(SupplierMaterialSvc).DeleteSupplierMaterial, constants.ObjectTypeSupplierMaterial, "id")
		},
	})
}
