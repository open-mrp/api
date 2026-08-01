package service

import (
	"context"
	"slices"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/forecast"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

// forecastMonth holds filled month data used by the demand forecast algorithm.
type forecastMonth struct {
	monthStart time.Time
	demand     float64
	revenue    float64
}

// buildDemandForecast composes the raw monthly demand/revenue reads into per-item history and seasonal-EMA forecasts.
func (s *analyticsSvcImpl) buildDemandForecast(ctx context.Context, params domain.GetDemandForecastParams) (*domain.DemandForecastResult, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.build_demand_forecast")
	defer span.End()

	// Defaults and bounds matching dashboard behavior.
	const maxHistoryMonths = 60
	const maxForecastMonths = 24
	safeHistoryMonths := 24
	safeForecastMonths := 4
	if params.HistoryMonths != nil && *params.HistoryMonths > 0 {
		safeHistoryMonths = min(int(*params.HistoryMonths), maxHistoryMonths)
	}
	if params.ForecastMonths != nil && *params.ForecastMonths > 0 {
		safeForecastMonths = min(int(*params.ForecastMonths), maxForecastMonths)
	}

	now := time.Now().UTC()
	currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	nextMonthStart := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	historyStart := time.Date(now.Year(), now.Month()-time.Month(safeHistoryMonths), 1, 0, 0, 0, 0, time.UTC)
	lastCompleteMonthStart := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, time.UTC)

	// Current month fraction for proration.
	daysInCurrentMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	currentMonthFraction := float64(now.Day()) / float64(daysInCurrentMonth)

	repo := s.repos.NewAnalyticsRepo()
	window := domain.GetDemandForecastWindowParams{
		AccountID: params.AccountID,
		StartDate: historyStart,
		EndDate:   nextMonthStart,
	}

	// Fetch monthly demand data (order-based).
	demandRows, apiErr := repo.GetDemandForecastMonthlyDemand(ctx, window)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Fetch monthly sales data (invoice-based).
	salesRows, apiErr := repo.GetDemandForecastMonthlyRevenue(ctx, window)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Build sales lookup: itemID -> [{year, month, revenue}].
	type salesEntry struct {
		year    int
		month   int
		revenue float64
	}
	salesByItem := make(map[string][]salesEntry)
	for _, sr := range salesRows {
		salesByItem[sr.ItemID] = append(salesByItem[sr.ItemID], salesEntry{
			year:    int(sr.RevenueYear),
			month:   int(sr.RevenueMonth),
			revenue: sr.MonthlyRevenue,
		})
	}

	if len(demandRows) == 0 {
		return &domain.DemandForecastResult{
			Items:                []domain.DemandForecastItem{},
			CurrentMonthFraction: currentMonthFraction,
		}, nil
	}

	// Group demand rows by item.
	type monthData struct {
		year    int
		month   int
		demand  float64
		revenue float64
	}
	type itemAccumulator struct {
		itemID             string
		productLineID      *string
		productSku         string
		productDescription *string
		unit               string
		currency           string
		months             []monthData
	}

	itemsMap := make(map[string]*itemAccumulator)
	var itemOrder []string
	for _, row := range demandRows {
		// Apply filters (sqlc doesn't support optional IN clauses).
		if len(params.ProductLineIDs) > 0 {
			if row.ProductLineID == nil {
				continue
			}
			found := slices.Contains(params.ProductLineIDs, *row.ProductLineID)
			if !found {
				continue
			}
		}
		if len(params.ItemIDs) > 0 {
			found := slices.Contains(params.ItemIDs, row.ItemID)
			if !found {
				continue
			}
		}

		acc, ok := itemsMap[row.ItemID]
		if !ok {
			acc = &itemAccumulator{
				itemID:             row.ItemID,
				productLineID:      row.ProductLineID,
				productSku:         row.ProductSku,
				productDescription: row.ProductDescription,
				unit:               row.Unit,
				currency:           row.Currency,
			}
			itemsMap[row.ItemID] = acc
			itemOrder = append(itemOrder, row.ItemID)
		}
		acc.months = append(acc.months, monthData{
			year:    int(row.DemandYear),
			month:   int(row.DemandMonth),
			demand:  row.MonthlyDemand,
			revenue: row.MonthlyRevenue,
		})
	}

	if len(itemsMap) == 0 {
		return &domain.DemandForecastResult{
			Items:                []domain.DemandForecastItem{},
			CurrentMonthFraction: currentMonthFraction,
		}, nil
	}

	// Find global last history month.
	var globalLastHistoryMonth time.Time
	for _, acc := range itemsMap {
		for _, m := range acc.months {
			ms := time.Date(m.year, time.Month(m.month), 1, 0, 0, 0, 0, time.UTC)
			if ms.After(globalLastHistoryMonth) {
				globalLastHistoryMonth = ms
			}
		}
	}
	if globalLastHistoryMonth.IsZero() {
		globalLastHistoryMonth = lastCompleteMonthStart
	}
	fillEndMonthStart := globalLastHistoryMonth
	if fillEndMonthStart.Before(currentMonthStart) {
		fillEndMonthStart = currentMonthStart
	}

	// Confidence band z-score (~68% CI, one standard deviation).
	const zScore = 1.0

	type ymKey struct{ y, m int }

	items := make([]domain.DemandForecastItem, 0, len(itemsMap))
	for _, itemID := range itemOrder {
		acc := itemsMap[itemID]

		if len(acc.months) == 0 {
			continue
		}

		// Find first month.
		firstMonth := time.Date(acc.months[0].year, time.Month(acc.months[0].month), 1, 0, 0, 0, 0, time.UTC)
		for _, m := range acc.months[1:] {
			ms := time.Date(m.year, time.Month(m.month), 1, 0, 0, 0, 0, time.UTC)
			if ms.Before(firstMonth) {
				firstMonth = ms
			}
		}

		// Build demand/revenue lookup by year-month key.
		demandByKey := make(map[ymKey]float64)
		revenueByKey := make(map[ymKey]float64)
		for _, m := range acc.months {
			k := ymKey{m.year, m.month}
			demandByKey[k] += m.demand
			revenueByKey[k] += m.revenue
		}

		// Fill months from first observed through fillEndMonthStart.
		var filledMonths []forecastMonth
		cursor := firstMonth
		for !cursor.After(fillEndMonthStart) {
			k := ymKey{cursor.Year(), int(cursor.Month())}
			filledMonths = append(filledMonths, forecastMonth{
				monthStart: cursor,
				demand:     demandByKey[k],
				revenue:    revenueByKey[k],
			})
			cursor = time.Date(cursor.Year(), cursor.Month()+1, 1, 0, 0, 0, 0, time.UTC)
		}

		// Capture current month partial values.
		var actualPartialDemand, actualPartialRevenue float64
		for _, fm := range filledMonths {
			if fm.monthStart.Equal(currentMonthStart) {
				actualPartialDemand = fm.demand
				actualPartialRevenue = fm.revenue
			}
		}

		// Training: only complete months (exclude current partial month).
		var completeMonths []forecastMonth
		for _, fm := range filledMonths {
			if fm.monthStart.Before(currentMonthStart) {
				completeMonths = append(completeMonths, fm)
			}
		}

		// History uses end-of-period convention: Jan data → 02/01.
		history := make([]domain.DemandHistoryPoint, len(completeMonths))
		revenueHistory := make([]domain.RevenueHistoryPoint, len(completeMonths))
		for i, cm := range completeMonths {
			displayDate := time.Date(cm.monthStart.Year(), cm.monthStart.Month()+1, 1, 0, 0, 0, 0, time.UTC)
			history[i] = domain.DemandHistoryPoint{Date: displayDate, Demand: cm.demand}
			revenueHistory[i] = domain.RevenueHistoryPoint{Date: displayDate, Revenue: cm.revenue}
		}

		// --- Demand forecasting: Seasonal EMA ---
		demandForecast := seasonalEMAForecast(completeMonths, safeForecastMonths, lastCompleteMonthStart, zScore, func(fm forecastMonth) float64 { return fm.demand })

		// --- Revenue forecasting: same Seasonal EMA approach ---
		revenueForecast := seasonalEMAForecast(completeMonths, safeForecastMonths, lastCompleteMonthStart, zScore, func(fm forecastMonth) float64 { return fm.revenue })

		// --- Sales (invoice-based) forecasting ---
		salesEntries := salesByItem[acc.itemID]
		salesByKey := make(map[ymKey]float64)
		for _, se := range salesEntries {
			salesByKey[ymKey{se.year, se.month}] += se.revenue
		}

		var filledSalesMonths []forecastMonth
		salesCursor := firstMonth
		for !salesCursor.After(fillEndMonthStart) {
			k := ymKey{salesCursor.Year(), int(salesCursor.Month())}
			filledSalesMonths = append(filledSalesMonths, forecastMonth{
				monthStart: salesCursor,
				revenue:    salesByKey[k],
			})
			salesCursor = time.Date(salesCursor.Year(), salesCursor.Month()+1, 1, 0, 0, 0, 0, time.UTC)
		}

		var actualPartialSales float64
		for _, fm := range filledSalesMonths {
			if fm.monthStart.Equal(currentMonthStart) {
				actualPartialSales = fm.revenue
			}
		}

		var completeSalesMonths []forecastMonth
		for _, fm := range filledSalesMonths {
			if fm.monthStart.Before(currentMonthStart) {
				completeSalesMonths = append(completeSalesMonths, fm)
			}
		}

		salesHistoryPoints := make([]domain.RevenueHistoryPoint, len(completeSalesMonths))
		for i, cm := range completeSalesMonths {
			displayDate := time.Date(cm.monthStart.Year(), cm.monthStart.Month()+1, 1, 0, 0, 0, 0, time.UTC)
			salesHistoryPoints[i] = domain.RevenueHistoryPoint{Date: displayDate, Revenue: cm.revenue}
		}

		var salesForecast []domain.DemandForecastPoint
		if len(completeSalesMonths) > 0 {
			salesForecast = seasonalEMAForecast(completeSalesMonths, safeForecastMonths, lastCompleteMonthStart, zScore, func(fm forecastMonth) float64 { return fm.revenue })
		} else {
			salesForecast = []domain.DemandForecastPoint{}
		}

		revenueForecastPoints := toRevenueForecastPoints(revenueForecast)
		salesForecastPoints := toRevenueForecastPoints(salesForecast)

		items = append(items, domain.DemandForecastItem{
			ItemID:              acc.itemID,
			ProductLineID:       acc.productLineID,
			ProductSku:          acc.productSku,
			ProductDescription:  acc.productDescription,
			Unit:                acc.unit,
			Currency:            acc.currency,
			History:             history,
			Forecast:            demandForecast,
			RevenueHistory:      revenueHistory,
			RevenueForecast:     revenueForecastPoints,
			SalesHistory:        salesHistoryPoints,
			SalesForecast:       salesForecastPoints,
			CurrentMonthDemand:  actualPartialDemand,
			CurrentMonthRevenue: actualPartialRevenue,
			CurrentMonthSales:   actualPartialSales,
		})
	}

	return &domain.DemandForecastResult{
		Items:                items,
		CurrentMonthFraction: currentMonthFraction,
	}, nil
}

// seasonalEMAForecast adapts the shared forecaster (internal/forecast) to the analytics row shape. The algorithm itself lives there because the production scheduler forecasts the same demand and the two must not drift apart.
func seasonalEMAForecast(
	completeMonths []forecastMonth,
	numForecastMonths int,
	baseForecastStart time.Time,
	zScore float64,
	valueExtractor func(forecastMonth) float64,
) []domain.DemandForecastPoint {
	observations := make([]forecast.Observation, len(completeMonths))
	for i, cm := range completeMonths {
		observations[i] = forecast.Observation{
			MonthStart: cm.monthStart,
			Value:      valueExtractor(cm),
		}
	}

	points := forecast.SeasonalEMA(observations, baseForecastStart, numForecastMonths, zScore)

	out := make([]domain.DemandForecastPoint, len(points))
	for i, p := range points {
		out[i] = domain.DemandForecastPoint{
			Date:       p.Date,
			Forecast:   p.Forecast,
			LowerBound: p.LowerBound,
			UpperBound: p.UpperBound,
		}
	}
	return out
}

func toRevenueForecastPoints(points []domain.DemandForecastPoint) []domain.RevenueForecastPoint {
	result := make([]domain.RevenueForecastPoint, len(points))
	for i, p := range points {
		result[i] = domain.RevenueForecastPoint(p)
	}
	return result
}
