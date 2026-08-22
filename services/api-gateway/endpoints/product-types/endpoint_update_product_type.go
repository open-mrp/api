package producttypeep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to partially update a product type.
type UpdateProductTypeRequest struct {
	// Product type ID.
	ProductTypeID string `path:"id" validate:"required"`
	// New display name for the product type.
	//
	// Must be unique across all product types.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// New machine-readable code for the product type.
	//
	// Must be unique across all product types. Existing products point at their product type by code, so changing it leaves them referencing a code that no longer exists; only rename a code no product uses.
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
//
// Product types are shared across all accounts, so a change here applies everywhere.
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
