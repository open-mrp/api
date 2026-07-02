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

// ListSuppliersRequest is the request to list suppliers.
type ListSuppliersRequest struct {
	apiresource.PaginationRequest
	// Filter by item IDs.
	//
	// Returns only suppliers that provide at least one material linked to any of the given items.
	ItemIDs []string `query:"item_ids"`
	// Only return suppliers created at or after this timestamp.
	StartDate *time.Time `query:"start_date"`
	// Only return suppliers created at or before this timestamp.
	EndDate *time.Time `query:"end_date"`
}

// TODO: stop returning SupplierSummary; return the full Supplier apiresource and use proper includes values to control expansion.

// Returns a paginated list of suppliers for the current account.
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
