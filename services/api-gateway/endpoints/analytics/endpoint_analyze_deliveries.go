package analyticsep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// AnalyzeDeliveriesRequest is the request to analyze delivery performance.
type AnalyzeDeliveriesRequest struct {
	// The start date for the analysis period.
	StartDate time.Time `json:"starts_at" validate:"required"`
	// The end date for the analysis period.
	EndDate time.Time `json:"ends_at" validate:"required"`
	// Optional product line IDs to filter by.
	ProductLineIDs []string `json:"product_line_ids,omitempty"`
	// Optional customer IDs to filter by.
	CustomerIDs []string `json:"customer_ids,omitempty"`
	// Optional customer group IDs to filter by.
	CustomerGroupIDs []string `json:"customer_group_ids,omitempty"`
	// Optional sales rep IDs to filter by.
	SalesRepIDs []string `json:"sales_rep_ids,omitempty"`
	// Optional target delivery time in days.
	TargetDeliveryTimeDays *int64 `json:"target_delivery_time_days,omitempty"`
	// Whether to override promised dates with the target delivery time.
	OverridePromisedDates *bool `json:"override_promised_dates,omitempty"`
}

// Returns delivery performance statistics over a date range, including on-time rates, average delivery times, and time-to-first-shipment metrics.
type AnalyzeDeliveriesEndpoint struct{}

func (e *AnalyzeDeliveriesEndpoint) Materialize() *apiendpoint.APIEndpoint[*AnalyzeDeliveriesRequest, *apiresource.AnalyzeDeliveriesResponse] {
	return (&apiendpoint.APIEndpoint[*AnalyzeDeliveriesRequest, *apiresource.AnalyzeDeliveriesResponse]{
		Title:               "Analyze Deliveries",
		Method:              http.MethodPut,
		Route:               "/v1/core/analytics/deliveries",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainInvoices, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeDeliveriesRequest) (*apiresource.AnalyzeDeliveriesResponse, *apierror.APIError) {
			return svc.(AnalyticsSvc).AnalyzeDeliveries
		},
	})
}
