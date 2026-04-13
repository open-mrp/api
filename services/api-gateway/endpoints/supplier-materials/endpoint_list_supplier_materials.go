package suppliermaterialep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list supplier materials.
type ListSupplierMaterialsRequest struct {
	apiresource.PaginationRequest
	// Supplier ID.
	SupplierID string `path:"supplier_id" validate:"required"`
}

type ListSupplierMaterialsEndpoint struct{}

func (e *ListSupplierMaterialsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListSupplierMaterialsRequest, *apiresource.List[apiresource.SupplierMaterial]] {
	return &apiendpoint.APIEndpoint[*ListSupplierMaterialsRequest, *apiresource.List[apiresource.SupplierMaterial]]{
		Title:             "List Supplier Materials",
		Description:       "Returns a paginated list of supplier materials.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/suppliers/{supplier_id}/materials",
		Request:           &ListSupplierMaterialsRequest{},
		Response:          &apiresource.List[apiresource.SupplierMaterial]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListSupplierMaterialsRequest) (*apiresource.List[apiresource.SupplierMaterial], *apierror.APIError) {
			return svc.(SupplierMaterialSvc).ListSupplierMaterials
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeSupplierMaterial,
			Fields:     []string{"material", "material.item"},
		}),
	}
}
