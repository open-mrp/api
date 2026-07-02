package producttypeep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to create a product type.
type CreateProductTypeRequest struct {
	// Human-readable name of the product type.
	//
	// Must be unique across product types.
	Name string `json:"name" validate:"required,max=255"`
	// Stable machine-readable code for the product type.
	//
	// Must be unique across product types. Products reference their product type by this code, and the code can be used in place of the ID when retrieving a product type.
	Code string `json:"code" validate:"required,max=255"`
}

var sampleCreateProductTypeRequest = &CreateProductTypeRequest{
	Name: "Sale",
	Code: "sale",
}

func (*CreateProductTypeRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateProductTypeRequest)
}

// Creates a product type.
type CreateProductTypeEndpoint struct{}

func (e *CreateProductTypeEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateProductTypeRequest, *apiresource.ProductType] {
	return (&apiendpoint.APIEndpoint[*CreateProductTypeRequest, *apiresource.ProductType]{
		Title:               "Create Product Type",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/catalog/product-types",
		SuccessStatusCode:   http.StatusCreated,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainProductTypes, Action: types.ActionCreate}},
		ObjectType:          constants.ObjectTypeProductType,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateProductTypeRequest) (*apiresource.ProductType, *apierror.APIError) {
			return svc.(ProductTypeSvc).CreateProductType
		},
		LocationFunc: func(resp *apiresource.ProductType) string {
			return "/v1/catalog/product-types/" + resp.ID
		},
	})
}
