package suppliermaterialep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to get a supplier material.
type GetSupplierMaterialRequest struct {
	// Supplier ID.
	SupplierID string `path:"supplier_id" validate:"required"`
	// Supplier material ID.
	ItemID string `path:"id" validate:"required"`
}

type GetSupplierMaterialEndpoint struct{}

func (e *GetSupplierMaterialEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetSupplierMaterialRequest, *apiresource.SupplierMaterial] {
	return &apiendpoint.APIEndpoint[*GetSupplierMaterialRequest, *apiresource.SupplierMaterial]{
		Title:             "Get Supplier Material",
		Description:       "Returns a supplier material by ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/suppliers/{supplier_id}/materials/{id}",
		Request:           &GetSupplierMaterialRequest{},
		Response:          &apiresource.SupplierMaterial{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetSupplierMaterialRequest) (*apiresource.SupplierMaterial, *apierror.APIError) {
			return svc.(SupplierMaterialSvc).GetSupplierMaterial
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeSupplierMaterial,
			Fields:     []string{"material", "material.item"},
		}),
	}
}
