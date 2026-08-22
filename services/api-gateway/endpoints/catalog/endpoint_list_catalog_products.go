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

// Request to list products in a catalog product line.
type ListCatalogProductsRequest struct {
	// Product line ID.
	ProductLineID string `path:"id" validate:"required"`
	apiresource.PaginationRequest
}

// Returns the products in a product line, grouped by item category.
//
// Each category lists the properties its products vary along and the products themselves, with categories ordered by name and products ordered by SKU. Only products whose `portal_visibility` is `visible` are included, and a customer user additionally only sees product lines they have been granted access to.
//
// Pagination and the `q` search term apply to the categories — `q` is matched against the category name, and a page returns whole categories with all of their products.
type ListCatalogProductsEndpoint struct{}

func (e *ListCatalogProductsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListCatalogProductsRequest, *apiresource.List[apiresource.CatalogCategory]] {
	return (&apiendpoint.APIEndpoint[*ListCatalogProductsRequest, *apiresource.List[apiresource.CatalogCategory]]{
		Title:             "List Catalog Products",
		Method:            http.MethodGet,
		Route:             "/v1/catalog/catalog/product-lines/{id}/products",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainProducts, Action: types.ActionRead},
			{Domain: types.PermissionDomainCustomers, Action: types.ActionRead},
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionRead},
		},
		ObjectType: constants.ObjectTypeCatalogCategory,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListCatalogProductsRequest) (*apiresource.List[apiresource.CatalogCategory], *apierror.APIError) {
			return svc.(CatalogSvc).ListCatalogProducts
		},
	})
}
