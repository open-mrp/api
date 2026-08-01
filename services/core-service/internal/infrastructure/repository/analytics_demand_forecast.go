package repository

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

// GetDemandForecastMonthlyDemand returns order-based monthly demand and revenue rows per item within the given window.
func (r *analyticsRepoImpl) GetDemandForecastMonthlyDemand(ctx context.Context, params domain.GetDemandForecastWindowParams) ([]domain.DemandForecastMonthlyDemandRow, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_demand_forecast_monthly_demand")
	defer span.End()

	rows, err := r.queries.GetDemandForecastMonthlyDemand(ctx, sqlc.GetDemandForecastMonthlyDemandParams{
		OwnerAccountID: params.AccountID,
		StartDate:      params.StartDate,
		EndDate:        params.EndDate,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]domain.DemandForecastMonthlyDemandRow, len(rows))
	for i, row := range rows {
		out[i] = domain.DemandForecastMonthlyDemandRow{
			ItemID:             row.ItemID,
			ProductSku:         row.ProductSku,
			ProductDescription: nullStringPtr(row.ProductDescription),
			ProductLineID:      nullStringPtr(row.ProductLineID),
			Unit:               row.Unit,
			Currency:           row.Currency,
			DemandYear:         row.DemandYear,
			DemandMonth:        row.DemandMonth,
			MonthlyDemand:      decimalToFloat64(row.MonthlyDemand),
			MonthlyRevenue:     decimalToFloat64(row.MonthlyRevenue),
		}
	}
	return out, nil
}

// GetDemandForecastMonthlyRevenue returns invoice-based monthly revenue rows per item within the given window.
func (r *analyticsRepoImpl) GetDemandForecastMonthlyRevenue(ctx context.Context, params domain.GetDemandForecastWindowParams) ([]domain.DemandForecastMonthlyRevenueRow, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_demand_forecast_monthly_revenue")
	defer span.End()

	rows, err := r.queries.GetDemandForecastMonthlyRevenue(ctx, sqlc.GetDemandForecastMonthlyRevenueParams{
		OwnerAccountID: params.AccountID,
		StartDate:      params.StartDate,
		EndDate:        params.EndDate,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]domain.DemandForecastMonthlyRevenueRow, len(rows))
	for i, row := range rows {
		out[i] = domain.DemandForecastMonthlyRevenueRow{
			ItemID:         row.ItemID,
			RevenueYear:    row.RevenueYear,
			RevenueMonth:   row.RevenueMonth,
			MonthlyRevenue: decimalToFloat64(row.MonthlyRevenue),
		}
	}
	return out, nil
}
