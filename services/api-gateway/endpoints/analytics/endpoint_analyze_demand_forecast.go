package analyticsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// AnalyzeDemandForecastRequest is the request to generate a demand forecast.
type AnalyzeDemandForecastRequest struct {
	// Optional product line IDs to filter by.
	ProductLineIDs []string `json:"product_line_ids,omitempty"`
	// Optional item IDs to filter by.
	ItemIDs []string `json:"item_ids,omitempty"`
	// Optional number of months of historical data to use.
	HistoryMonths *int64 `json:"history_months,omitempty"`
	// Optional number of months to forecast.
	ForecastMonths *int64 `json:"forecast_months,omitempty"`
}

// Returns demand forecasts for items, including historical data and projected demand with confidence bounds.
type AnalyzeDemandForecastEndpoint struct{}

func (e *AnalyzeDemandForecastEndpoint) Materialize() *apiendpoint.APIEndpoint[*AnalyzeDemandForecastRequest, *apiresource.AnalyzeDemandForecastResponse] {
	return (&apiendpoint.APIEndpoint[*AnalyzeDemandForecastRequest, *apiresource.AnalyzeDemandForecastResponse]{
		Title:             "Analyze Demand Forecast",
		Method:            http.MethodPut,
		Route:             "/v1/core/analytics/demand-forecast",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeDemandForecastRequest) (*apiresource.AnalyzeDemandForecastResponse, *apierror.APIError) {
			return svc.(AnalyticsSvc).AnalyzeDemandForecast
		},
	})
}
