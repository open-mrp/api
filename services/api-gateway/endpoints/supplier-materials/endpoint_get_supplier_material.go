package suppliermaterialep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

type GetSupplierMaterialRequest struct {
	SupplierID string `path:"supplier_id" validate:"required"`
	ItemID     string `path:"id" validate:"required"`
}

type GetSupplierMaterialEndpoint struct{}

func (e *GetSupplierMaterialEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetSupplierMaterialRequest, *apiresource.SupplierMaterial] {
	return &apiendpoint.APIEndpoint[*GetSupplierMaterialRequest, *apiresource.SupplierMaterial]{
		Title:             "Get Supplier Material",
		Description:       "Returns a single supplier material by supplier and item ID.",
		Method:            http.MethodGet,
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
