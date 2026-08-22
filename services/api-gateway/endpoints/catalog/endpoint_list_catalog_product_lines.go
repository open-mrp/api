package catalogep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list catalog product lines.
type ListCatalogProductLinesRequest struct {
	apiresource.PaginationRequest
}

// Returns the product lines available in the catalog, ordered by name.
//
// A product line only appears once it holds at least one product whose `portal_visibility` is `visible`. When the caller is a customer user, the list is narrowed further to the product lines that customer has been granted access to, either directly, through an account group, or through the account group used as their price group. The `q` search term is matched against the product line name.
type ListCatalogProductLinesEndpoint struct{}

func (e *ListCatalogProductLinesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListCatalogProductLinesRequest, *apiresource.List[apiresource.CatalogProductLine]] {
	return (&apiendpoint.APIEndpoint[*ListCatalogProductLinesRequest, *apiresource.List[apiresource.CatalogProductLine]]{
		Title:             "List Catalog Product Lines",
		Method:            http.MethodGet,
		Route:             "/v1/catalog/catalog/product-lines",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProducts, Action: types.ActionRead},
			{Domain: types.PermissionDomainCustomers, Action: types.ActionRead},
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionRead},
		},
		ObjectType: constants.ObjectTypeCatalogProductLine,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListCatalogProductLinesRequest) (*apiresource.List[apiresource.CatalogProductLine], *apierror.APIError) {
			return svc.(CatalogSvc).ListCatalogProductLines
		},
	})
}
