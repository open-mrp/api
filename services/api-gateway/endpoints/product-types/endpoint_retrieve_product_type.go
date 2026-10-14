package producttypeep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to retrieve a product type.
type RetrieveProductTypeRequest struct {
	// Product type ID or code.
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
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainProductTypes, Action: types.ActionRead}}},
		ObjectType:        constants.ObjectTypeProductType,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveProductTypeRequest) (*apiresource.ProductType, *apierror.APIError) {
			return svc.(ProductTypeSvc).GetProductType
		},
	})
}
