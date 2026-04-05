package supplierep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// GetSupplierRequest is the request to retrieve a single supplier by ID.
type GetSupplierRequest struct {
	// The ID of the supplier to retrieve.
	SupplierID string `path:"id" validate:"required"`
}

type GetSupplierEndpoint struct{}

func (e *GetSupplierEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetSupplierRequest, *apiresource.SupplierDetail] {
	return &apiendpoint.APIEndpoint[*GetSupplierRequest, *apiresource.SupplierDetail]{
		Title:             "Get Supplier",
		Description:       "Returns a single supplier by its ID.",
		Method:            http.MethodGet,
		Route:             "/v1/operations/suppliers/{id}",
		ContentType:       "application/json",
		Request:           &GetSupplierRequest{},
		Response:          &apiresource.SupplierDetail{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetSupplierRequest) (*apiresource.SupplierDetail, *apierror.APIError) {
			return svc.(SupplierSvc).GetSupplier
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeSupplier,
			Fields:     []string{"bill_to_address", "ship_to_address"},
		}),
	}
}
