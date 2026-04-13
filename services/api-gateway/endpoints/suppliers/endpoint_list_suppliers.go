package supplierep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ListSuppliersRequest is the request to list suppliers.
type ListSuppliersRequest struct {
	apiresource.PaginationRequest
	// Item IDs filter. Returns suppliers with materials for these items.
	ItemIDs []string `query:"item_ids"`
	// Start date filter (created after).
	StartDate *time.Time `query:"start_date"`
	// End date filter (created before).
	EndDate *time.Time `query:"end_date"`
}

type ListSuppliersEndpoint struct{}

func (e *ListSuppliersEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListSuppliersRequest, *apiresource.List[apiresource.SupplierSummary]] {
	return &apiendpoint.APIEndpoint[*ListSuppliersRequest, *apiresource.List[apiresource.SupplierSummary]]{
		Title:             "List Suppliers",
		Description:       "Returns a paginated list of suppliers for the current account.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/suppliers",
		Request:           &ListSuppliersRequest{},
		Response:          &apiresource.List[apiresource.SupplierSummary]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListSuppliersRequest) (*apiresource.List[apiresource.SupplierSummary], *apierror.APIError) {
			return svc.(SupplierSvc).ListSuppliers
		},
	}
}
