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

// Request to list supplier materials.
type ListSupplierMaterialsRequest struct {
	apiresource.PaginationRequest
	// ID of the supplier whose material links to list.
	SupplierID string `path:"supplier_id" validate:"required"`
}

// Returns a paginated list of materials linked to the given supplier, newest first.
//
// Both active and inactive links are returned. The `q` search term matches the supplier part number and description as well as the underlying item's SKU and description.
type ListSupplierMaterialsEndpoint struct{}

func (e *ListSupplierMaterialsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListSupplierMaterialsRequest, *apiresource.List[apiresource.SupplierMaterial]] {
	return (&apiendpoint.APIEndpoint[*ListSupplierMaterialsRequest, *apiresource.List[apiresource.SupplierMaterial]]{
		Title:             "List Supplier Materials",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/suppliers/{supplier_id}/materials",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeSupplierMaterial,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListSupplierMaterialsRequest) (*apiresource.List[apiresource.SupplierMaterial], *apierror.APIError) {
			return svc.(SupplierMaterialSvc).ListSupplierMaterials
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeSupplierMaterial,
			Fields:     []string{"material", "material.item"},
		}),
	})
}
