package analyticsep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// AnalyzeDeliveryPerformanceRequest is the request to measure promises against shipments.
type AnalyzeDeliveryPerformanceRequest struct {
	// The start date for the analysis period.
	StartDate time.Time `json:"starts_at" validate:"required"`
	// The end date for the analysis period.
	EndDate time.Time `json:"ends_at" validate:"required"`
	// The period to break the results down by. Defaults to `week`.
	Granularity field.Optional[constants.DeliveryGranularity] `json:"granularity,omitzero"`
	// Only measure orders bought by these customers. Their child accounts are included, matching how the sales analytics resolve a customer.
	CustomerIDs []string `json:"customer_ids,omitempty"`
	// Only measure orders whose customer sits in these groups.
	CustomerGroupIDs []string `json:"customer_group_ids,omitempty"`
	// Only measure orders containing at least one line in these product lines.
	ProductLineIDs []string `json:"product_line_ids,omitempty"`
	// Only measure orders owned by these sales reps.
	SalesRepIDs []string `json:"sales_rep_ids,omitempty"`
}

// Returns how reliably promised delivery dates were met.
//
// Orders are counted in the period their promise came due, not the period they shipped — an order promised in March and shipped in May is March's miss. On time means the first shipment left on or before the promised date, because the promise is that the order starts moving by then; judging on the last shipment would fail an order the customer received on time in two boxes. On time in full adds that the whole ordered quantity was packed.
//
// The denominator is orders that were due, not orders that shipped, so an order past its date and still unshipped counts against the rate rather than being held back until it moves. Excluding open orders would let a plant with a growing late backlog report perfect delivery.
//
// Only orders carrying a ship-by commitment participate. An order with no commitment cannot be late, and counting it as on time would inflate the rate with orders nobody promised anything about — `uncommitted_order_count` says how many were excluded, so the gap is visible rather than silent.
//
// Every rate is null rather than zero when nothing was due, and average lateness is measured over late orders only.
//
// The same window is also returned sliced by customer, customer group, product line, and the rule each ship-by date came from — each ordered worst-first, and each derived from the same set of orders as the headline so a drilldown always adds up to it. `by_product_line` is the one exception to that: an order spanning two lines is counted under both, because a late order is late for every line on it.
//
// Every filter is empty-means-all and they combine with AND. They narrow `uncommitted_order_count` too, so the excluded count always describes the same slice of the order book the rates do.
type AnalyzeDeliveryPerformanceEndpoint struct{}

func (e *AnalyzeDeliveryPerformanceEndpoint) Materialize() *apiendpoint.APIEndpoint[*AnalyzeDeliveryPerformanceRequest, *apiresource.AnalyzeDeliveryPerformanceResponse] {
	return (&apiendpoint.APIEndpoint[*AnalyzeDeliveryPerformanceRequest, *apiresource.AnalyzeDeliveryPerformanceResponse]{
		Title:               "Analyze Delivery Performance",
		Method:              http.MethodPut,
		Route:               "/v1/core/analytics/delivery-performance",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		AgentTool:           true,
		ReadOnly:            true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainSalesOrders, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeDeliveryPerformanceRequest) (*apiresource.AnalyzeDeliveryPerformanceResponse, *apierror.APIError) {
			return svc.(AnalyticsSvc).AnalyzeDeliveryPerformance
		},
	})
}
