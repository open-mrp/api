package supplierep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// RetrieveSupplierRequest is the request to get a supplier by ID.
type RetrieveSupplierRequest struct {
	// Supplier ID.
	SupplierID string `path:"id" validate:"required"`
}

// Returns a supplier by ID.
type RetrieveSupplierEndpoint struct{}

func (e *RetrieveSupplierEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveSupplierRequest, *apiresource.SupplierDetail] {
	return (&apiendpoint.APIEndpoint[*RetrieveSupplierRequest, *apiresource.SupplierDetail]{
		Title:             "Retrieve Supplier",
		Method:            http.MethodGet,
		Route:             "/v1/operations/suppliers/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveSupplierRequest) (*apiresource.SupplierDetail, *apierror.APIError) {
			return svc.(SupplierSvc).GetSupplier
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeSupplier,
			Fields:     []string{"bill_to_address", "ship_to_address"},
		}),
	})
}
