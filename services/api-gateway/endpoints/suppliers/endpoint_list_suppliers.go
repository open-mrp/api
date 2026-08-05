package supplierep

import (
	"context"
	"net/http"
	"time"

	"github.com/augno/api/services/auth-service/pkg/types"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list suppliers.
type ListSuppliersRequest struct {
	apiresource.PaginationRequest
	// Filter to suppliers that can source any of these items.
	//
	// A supplier matches when it provides a material for one of the items, whether or not that material link is active.
	ItemIDs []string `query:"item_ids"`
	// Only return suppliers created at or after this timestamp.
	StartDate *time.Time `query:"starts_at"`
	// Only return suppliers created at or before this timestamp.
	EndDate *time.Time `query:"ends_at"`
}

// TODO: stop returning SupplierSummary; return the full Supplier apiresource and use proper includes values to control expansion.

// Returns a paginated list of suppliers for the current account, newest first.
//
// Filters combine with AND, so an item filter and a date range narrow the list together. The `q` search term matches the supplier name and number.
type ListSuppliersEndpoint struct{}

func (e *ListSuppliersEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListSuppliersRequest, *apiresource.List[apiresource.SupplierSummary]] {
	return (&apiendpoint.APIEndpoint[*ListSuppliersRequest, *apiresource.List[apiresource.SupplierSummary]]{
		Title:             "List Suppliers",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/suppliers",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListSuppliersRequest) (*apiresource.List[apiresource.SupplierSummary], *apierror.APIError) {
			return svc.(SupplierSvc).ListSuppliers
		},
	})
}
