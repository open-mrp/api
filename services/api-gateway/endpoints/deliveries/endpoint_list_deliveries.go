package deliveryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list deliveries.
type ListDeliveriesRequest struct {
	apiresource.PaginationRequest
	// Filter by delivery status.
	//
	// Deliveries where nothing was accepted into inventory are hidden unless you ask for `rejected` or `all`.
	Status *constants.DeliveryListStatus `query:"status" default:"accepted"`
	// Filter to deliveries with at least one line for any of the given item IDs.
	ItemIDs []string `query:"item_ids"`
	// Filter to deliveries whose purchase order is with any of the given supplier account IDs.
	SupplierIDs []string `query:"supplier_ids"`
	// Only include deliveries created on or after this date (`YYYY-MM-DD`).
	StartDate *string `query:"starts_at"`
	// Only include deliveries created on or before this date (`YYYY-MM-DD`), covering that whole day.
	EndDate *string `query:"ends_at"`
}

// Returns a paginated list of deliveries for the current account, newest first.
//
// Only deliveries where goods were accepted into inventory are returned by default; pass `status` to include fully rejected ones.
type ListDeliveriesEndpoint struct{}

func (e *ListDeliveriesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListDeliveriesRequest, *apiresource.List[apiresource.Delivery]] {
	return (&apiendpoint.APIEndpoint[*ListDeliveriesRequest, *apiresource.List[apiresource.Delivery]]{
		Title:             "List Deliveries",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/deliveries",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeDelivery,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainDeliveries, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListDeliveriesRequest) (*apiresource.List[apiresource.Delivery], *apierror.APIError) {
			return svc.(DeliverySvc).ListDeliveries
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeDelivery,
			Fields:     []string{"related", "related.purchase_order", "related.receiving_order", "lines", "lines.item", "lines.unit_cost", "lines.location", "lines.lot"},
		}),
	})
}
