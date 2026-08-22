package suppliermaterialep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to retrieve a supplier material.
type RetrieveSupplierMaterialRequest struct {
	// ID of the supplier the material is linked to.
	SupplierID string `path:"supplier_id" validate:"required"`
	// ID of the material whose supplier link to retrieve.
	//
	// Supplier materials are addressed by the combination of supplier and material, so this path parameter takes the material's ID.
	MaterialID string `path:"id" validate:"required"`
}

// Returns the supplier material link for the given supplier and material.
type RetrieveSupplierMaterialEndpoint struct{}

func (e *RetrieveSupplierMaterialEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveSupplierMaterialRequest, *apiresource.SupplierMaterial] {
	return (&apiendpoint.APIEndpoint[*RetrieveSupplierMaterialRequest, *apiresource.SupplierMaterial]{
		Title:             "Retrieve Supplier Material",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/suppliers/{supplier_id}/materials/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeSupplierMaterial,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveSupplierMaterialRequest) (*apiresource.SupplierMaterial, *apierror.APIError) {
			return svc.(SupplierMaterialSvc).GetSupplierMaterial
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeSupplierMaterial,
			Fields:     []string{"material", "material.item"},
		}),
	})
}
