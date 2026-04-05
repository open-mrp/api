package producttypeep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// GetProductTypeRequest is the request to retrieve a single product type.
type GetProductTypeRequest struct {
	// The ID or code of the product type to retrieve.
	ProductTypeID string `path:"id" validate:"required"`
}

type GetProductTypeEndpoint struct{}

func (e *GetProductTypeEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetProductTypeRequest, *apiresource.ProductType] {
	return &apiendpoint.APIEndpoint[*GetProductTypeRequest, *apiresource.ProductType]{
		Title:             "Get Product Type",
		Description:       "Returns a single product type by its ID or code.",
		Method:            http.MethodGet,
		Route:             "/v1/catalog/product-types/{id}",
		ContentType:       "application/json",
		Request:           &GetProductTypeRequest{},
		Response:          &apiresource.ProductType{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetProductTypeRequest) (*apiresource.ProductType, *apierror.APIError) {
			return svc.(ProductTypeSvc).GetProductType
		},
	}
}
