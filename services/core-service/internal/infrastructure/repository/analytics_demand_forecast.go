package repository

import (
	"context"
	"math"
	"slices"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

// forecastMonth holds filled month data used by the demand forecast algorithm.
type forecastMonth struct {
	monthStart time.Time
	demand     float64
	revenue    float64
}

func (r *analyticsRepoImpl) getDemandForecastImpl(ctx context.Context, params domain.GetDemandForecastParams) (*domain.DemandForecastResult, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_demand_forecast")
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

	// Fetch monthly demand data (order-based).
	demandRows, err := r.queries.GetDemandForecastMonthlyDemand(ctx, sqlc.GetDemandForecastMonthlyDemandParams{
		OwnerAccountID: params.AccountID,
		StartDate:      historyStart,
		EndDate:        nextMonthStart,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Fetch monthly sales data (invoice-based).
	salesRows, err := r.queries.GetDemandForecastMonthlyRevenue(ctx, sqlc.GetDemandForecastMonthlyRevenueParams{
		OwnerAccountID: params.AccountID,
		StartDate:      historyStart,
		EndDate:        nextMonthStart,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
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
			revenue: decimalToFloat64(sr.MonthlyRevenue),
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
			if !row.ProductLineID.Valid {
				continue
			}
			found := slices.Contains(params.ProductLineIDs, row.ProductLineID.String)
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
				productLineID:      nullStringPtr(row.ProductLineID),
				productSku:         row.ProductSku,
				productDescription: nullStringPtr(row.ProductDescription),
				unit:               row.Unit,
				currency:           row.Currency,
			}
			itemsMap[row.ItemID] = acc
			itemOrder = append(itemOrder, row.ItemID)
		}
		acc.months = append(acc.months, monthData{
			year:    int(row.DemandYear),
			month:   int(row.DemandMonth),
			demand:  decimalToFloat64(row.MonthlyDemand),
			revenue: decimalToFloat64(row.MonthlyRevenue),
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

// seasonalEMAForecast implements the Seasonal EMA forecasting algorithm.
// It computes seasonal factors, deseasonalizes the data, applies EMA, and generates forecast points with confidence bounds.
func seasonalEMAForecast(
	completeMonths []forecastMonth,
	numForecastMonths int,
	baseForecastStart time.Time,
	zScore float64,
	valueExtractor func(forecastMonth) float64,
) []domain.DemandForecastPoint {
	observations := len(completeMonths)
	if observations == 0 {
		return []domain.DemandForecastPoint{}
	}

	// 1. Compute seasonal factors.
	type seasonalAgg struct {
		total float64
		count int
	}
	seasonalTotals := make(map[time.Month]*seasonalAgg)
	var overallTotal float64
	for _, cm := range completeMonths {
		season := cm.monthStart.Month()
		agg, ok := seasonalTotals[season]
		if !ok {
			agg = &seasonalAgg{}
			seasonalTotals[season] = agg
		}
		val := valueExtractor(cm)
		agg.total += val
		agg.count++
		overallTotal += val
	}

	overallAverage := 0.0
	if overallTotal > 0 {
		overallAverage = overallTotal / float64(observations)
	}

	seasonalFactors := make(map[time.Month]float64)
	for season, agg := range seasonalTotals {
		if overallAverage > 0 {
			seasonalFactors[season] = (agg.total / float64(agg.count)) / overallAverage
		} else {
			seasonalFactors[season] = 1
		}
	}

	// 2. Deseasonalize.
	deseasonalized := make([]float64, observations)
	for i, cm := range completeMonths {
		factor := seasonalFactors[cm.monthStart.Month()]
		if factor <= 0 {
			factor = 1
		}
		deseasonalized[i] = valueExtractor(cm) / factor
	}

	// 3. EMA.
	smoothingPeriod := min(observations, 12)
	emaAlpha := 2.0 / (float64(smoothingPeriod) + 1.0)
	emaLevel := deseasonalized[0]
	residuals := make([]float64, observations)
	for i := range observations {
		prediction := emaLevel
		if i == 0 {
			prediction = deseasonalized[0]
		}
		residuals[i] = deseasonalized[i] - prediction
		emaLevel = emaAlpha*deseasonalized[i] + (1-emaAlpha)*emaLevel
	}

	// 4. Residual std dev (last 12 only to avoid warm-up inflation).
	recentResiduals := residuals
	if len(recentResiduals) > 12 {
		recentResiduals = recentResiduals[len(recentResiduals)-12:]
	}
	var residualStdDev float64
	if len(recentResiduals) > 1 {
		var sumSq float64
		for _, r := range recentResiduals {
			sumSq += r * r
		}
		variance := sumSq / float64(len(recentResiduals)-1)
		if variance > 0 {
			residualStdDev = math.Sqrt(variance)
		}
	}

	// 5. Generate forecast points.
	forecast := make([]domain.DemandForecastPoint, numForecastMonths)
	for idx := range numForecastMonths {
		fMonth := time.Date(baseForecastStart.Year(), baseForecastStart.Month()+time.Month(idx+1), 1, 0, 0, 0, 0, time.UTC)
		displayDate := time.Date(fMonth.Year(), fMonth.Month()+1, 1, 0, 0, 0, 0, time.UTC)
		season := fMonth.Month()
		factor := seasonalFactors[season]
		if factor <= 0 {
			factor = 1
		}
		baseLevel := emaLevel
		if baseLevel < 0 {
			baseLevel = 0
		}
		forecastMean := baseLevel * factor

		band := residualStdDev * factor * zScore
		lowerBound := forecastMean - band
		if lowerBound < 0 {
			lowerBound = 0
		}

		forecast[idx] = domain.DemandForecastPoint{
			Date:       displayDate,
			Forecast:   forecastMean,
			LowerBound: lowerBound,
			UpperBound: forecastMean + band,
		}
	}

	return forecast
}

func toRevenueForecastPoints(points []domain.DemandForecastPoint) []domain.RevenueForecastPoint {
	result := make([]domain.RevenueForecastPoint, len(points))
	for i, p := range points {
		result[i] = domain.RevenueForecastPoint(p)
	}
	return result
}
