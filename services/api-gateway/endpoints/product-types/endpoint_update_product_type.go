package producttypeep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to partially update a product type.
type UpdateProductTypeRequest struct {
	// Product type ID.
	ProductTypeID string `path:"id" validate:"required"`
	// Display name.
	Name *string `json:"name,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Unique code.
	Code *string `json:"code,omitempty" nullable:"false" validate:"omitempty,max=255"`
}

var sampleUpdateProductTypeRequest = &UpdateProductTypeRequest{
	Name: new("Service"),
	Code: new("service"),
}

func (*UpdateProductTypeRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateProductTypeRequest)
}

type UpdateProductTypeEndpoint struct{}

func (e *UpdateProductTypeEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateProductTypeRequest, *apiresource.ProductType] {
	return &apiendpoint.APIEndpoint[*UpdateProductTypeRequest, *apiresource.ProductType]{
		Title:             "Update Product Type",
		Description:       "Partially updates a product type.",
		Method:            http.MethodPatch,
		Route:             "/v1/catalog/product-types/{id}",
		ContentType:       "application/json",
		Request:           &UpdateProductTypeRequest{},
		Response:          &apiresource.ProductType{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateProductTypeRequest) (*apiresource.ProductType, *apierror.APIError) {
			return svc.(ProductTypeSvc).UpdateProductType
		},
	}
}
