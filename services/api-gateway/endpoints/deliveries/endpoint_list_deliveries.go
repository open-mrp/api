package deliveryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list deliveries.
type ListDeliveriesRequest struct {
	apiresource.PaginationRequest
	// Filter by status: all, accepted, or rejected. Defaults to accepted.
	Status *string `query:"status" default:"accepted" validate:"omitempty,oneof=all accepted rejected"`
	// Filter by item IDs present in delivery lines.
	ItemIDs []string `query:"item_ids"`
	// Filter by supplier account IDs.
	SupplierIDs []string `query:"supplier_ids"`
	// Filter by start date (inclusive).
	StartDate *string `query:"start_date"`
	// Filter by end date (inclusive).
	EndDate *string `query:"end_date"`
}

// Returns a paginated list of deliveries for the caller's account.
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
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListDeliveriesRequest) (*apiresource.List[apiresource.Delivery], *apierror.APIError) {
			return svc.(DeliverySvc).ListDeliveries
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeDelivery,
			Fields:     []string{"purchase_order"},
		}),
	})
}
