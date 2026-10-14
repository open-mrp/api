package producttypeep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list product types.
type ListProductTypesRequest struct {
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
}

// Returns a paginated list of product types.
//
// Product types are global and not scoped to a specific account. The `q` search term is matched against the product type name.
type ListProductTypesEndpoint struct{}

func (e *ListProductTypesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListProductTypesRequest, *apiresource.List[apiresource.ProductType]] {
	return (&apiendpoint.APIEndpoint[*ListProductTypesRequest, *apiresource.List[apiresource.ProductType]]{
		Title:             "List Product Types",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/catalog/product-types",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainProductTypes, Action: types.ActionRead}}},
		ObjectType:        constants.ObjectTypeProductType,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListProductTypesRequest) (*apiresource.List[apiresource.ProductType], *apierror.APIError) {
			return svc.(ProductTypeSvc).ListProductTypes
		},
	})
}
