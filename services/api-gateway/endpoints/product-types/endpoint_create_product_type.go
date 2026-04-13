package producttypeep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to create a product type.
type CreateProductTypeRequest struct {
	// Display name.
	Name string `json:"name" validate:"required,max=255"`
	// Unique code.
	Code string `json:"code" validate:"required,max=255"`
}

var sampleCreateProductTypeRequest = &CreateProductTypeRequest{
	Name: "Sale",
	Code: "sale",
}

func (*CreateProductTypeRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateProductTypeRequest)
}

type CreateProductTypeEndpoint struct{}

func (e *CreateProductTypeEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateProductTypeRequest, *apiresource.ProductType] {
	return &apiendpoint.APIEndpoint[*CreateProductTypeRequest, *apiresource.ProductType]{
		Title:             "Create Product Type",
		Description:       "Creates a product type.",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/catalog/product-types",
		Request:           &CreateProductTypeRequest{},
		Response:          &apiresource.ProductType{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateProductTypeRequest) (*apiresource.ProductType, *apierror.APIError) {
			return svc.(ProductTypeSvc).CreateProductType
		},
		LocationFunc: func(resp *apiresource.ProductType) string {
			return "/v1/catalog/product-types/" + resp.ID
		},
	}
}
