package analyticsep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// AnalyzeDeliveryPerformanceRequest is the request to measure promises against shipments.
type AnalyzeDeliveryPerformanceRequest struct {
	// The start date for the analysis period.
	StartDate time.Time `json:"starts_at" validate:"required"`
	// The end date for the analysis period.
	EndDate time.Time `json:"ends_at" validate:"required"`
	// The period to break the results down by. Defaults to `week`.
	Granularity field.Optional[constants.DeliveryGranularity] `json:"granularity,omitzero"`
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
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainSalesOrders, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeDeliveryPerformanceRequest) (*apiresource.AnalyzeDeliveryPerformanceResponse, *apierror.APIError) {
			return svc.(AnalyticsSvc).AnalyzeDeliveryPerformance
		},
	})
}
