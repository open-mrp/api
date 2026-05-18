package producttypeep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to get a product type.
type RetrieveProductTypeRequest struct {
	// Product ID or code.
	ProductTypeID string `path:"id" validate:"required"`
}

// Returns a product type by ID or code.
type RetrieveProductTypeEndpoint struct{}

func (e *RetrieveProductTypeEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveProductTypeRequest, *apiresource.ProductType] {
	return (&apiendpoint.APIEndpoint[*RetrieveProductTypeRequest, *apiresource.ProductType]{
		Title:             "Retrieve Product Type",
		Method:            http.MethodGet,
		Route:             "/v1/catalog/product-types/{id}",
		ContentType:       "application/json",
		Request:           &RetrieveProductTypeRequest{},
		Response:          &apiresource.ProductType{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveProductTypeRequest) (*apiresource.ProductType, *apierror.APIError) {
			return svc.(ProductTypeSvc).GetProductType
		},
	}).WithDocSource(e)
}
