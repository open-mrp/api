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
	"github.com/augno/api/shared/field"
)

// Request to partially update a product type.
type UpdateProductTypeRequest struct {
	// Product type ID.
	ProductTypeID string `path:"id" validate:"required"`
	// New human-readable name.
	//
	// Must be unique across product types.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// New machine-readable code.
	//
	// Must be unique across product types.
	Code field.Optional[string] `json:"code,omitzero" validate:"omitempty,max=255"`
}

var sampleUpdateProductTypeRequest = &UpdateProductTypeRequest{
	Name: field.Some("Service"),
	Code: field.Some("service"),
}

func (*UpdateProductTypeRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateProductTypeRequest)
}

// Partially updates a product type.
type UpdateProductTypeEndpoint struct{}

func (e *UpdateProductTypeEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateProductTypeRequest, *apiresource.ProductType] {
	return (&apiendpoint.APIEndpoint[*UpdateProductTypeRequest, *apiresource.ProductType]{
		Title:               "Update Product Type",
		Method:              http.MethodPatch,
		Route:               "/v1/catalog/product-types/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainProductTypes, Action: types.ActionUpdate}},
		ObjectType:          constants.ObjectTypeProductType,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateProductTypeRequest) (*apiresource.ProductType, *apierror.APIError) {
			return svc.(ProductTypeSvc).UpdateProductType
		},
	})
}
