package shipmentep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ListShipmentsRequest is the request to list shipments.
type ListShipmentsRequest struct {
	apiresource.PaginationRequest
	// Filter by shipment status.
	Status *string `query:"status"`
	// Filter by item IDs.
	ItemIDs []string `query:"item_ids"`
	// Filter by customer IDs.
	CustomerIDs []string `query:"customer_ids"`
	// Filter by product line IDs.
	ProductLineIDs []string `query:"product_line_ids"`
	// Filter by customer group IDs.
	CustomerGroupIDs []string `query:"customer_group_ids"`
	// Filter by sales rep IDs.
	SalesRepIDs []string `query:"sales_rep_ids"`
	// Filter by start date (inclusive).
	StartDate *string `query:"start_date"`
	// Filter by end date (inclusive).
	EndDate *string `query:"end_date"`
}

type ListShipmentsEndpoint struct{}

func (e *ListShipmentsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListShipmentsRequest, *apiresource.List[apiresource.ShipmentSummary]] {
	return &apiendpoint.APIEndpoint[*ListShipmentsRequest, *apiresource.List[apiresource.ShipmentSummary]]{
		Title:             "List Shipments",
		Description:       "Returns a paginated list of shipments.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/shipments",
		Request:           &ListShipmentsRequest{},
		Response:          &apiresource.List[apiresource.ShipmentSummary]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListShipmentsRequest) (*apiresource.List[apiresource.ShipmentSummary], *apierror.APIError) {
			return svc.(ShipmentSvc).ListShipments
		},
	}
}
