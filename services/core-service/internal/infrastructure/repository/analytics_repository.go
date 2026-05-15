package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
	"go.opentelemetry.io/otel/trace"
)

var analyticsRepoTracer = tracing.GetTracer("core-service.infrastructure.repository.analytics")

type analyticsRepoImpl struct {
	queries *sqlc.Queries
}

func NewAnalyticsRepo(queries *sqlc.Queries) domain.AnalyticsRepo {
	return &analyticsRepoImpl{queries: queries}
}

func toRequiredNullTime(t time.Time) sql.NullTime {
	return sql.NullTime{Time: t, Valid: true}
}

func (r *analyticsRepoImpl) GetSalesEntries(ctx context.Context, params domain.AnalyzeSalesParams) ([]domain.SalesEntry, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_sales_entries")
	defer span.End()

	salesRepIDs := toNullStringSlice(params.SalesRepIDs)
	if salesRepIDs == nil {
		salesRepIDs = []sql.NullString{}
	}
	productLineIDs := toNullStringSlice(params.ProductLineIDs)
	if productLineIDs == nil {
		productLineIDs = []sql.NullString{}
	}
	customerGroupIDs := toNullStringSlice(params.CustomerGroupIDs)
	if customerGroupIDs == nil {
		customerGroupIDs = []sql.NullString{}
	}
	customerIDs := params.CustomerIDs
	if customerIDs == nil {
		customerIDs = []string{}
	}

	rows, err := r.queries.GetSalesEntries(ctx, sqlc.GetSalesEntriesParams{
		OwnerAccountID:             params.AccountID,
		StartDate:                  params.StartDate,
		EndDate:                    params.EndDate,
		IncludeSalesRepFilter:      len(params.SalesRepIDs) > 0,
		SalesRepIds:                salesRepIDs,
		IncludeProductLineFilter:   len(params.ProductLineIDs) > 0,
		ProductLineIds:             productLineIDs,
		IncludeCustomerGroupFilter: len(params.CustomerGroupIDs) > 0,
		CustomerGroupIds:           customerGroupIDs,
		IncludeCustomerFilter:      len(params.CustomerIDs) > 0,
		CustomerIds:                customerIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	entries := make([]domain.SalesEntry, len(rows))
	for i, row := range rows {
		var customerCreatedAt time.Time
		if row.CustomerCreatedAt.Valid {
			customerCreatedAt = row.CustomerCreatedAt.Time
		}

		var productLine *string
		if row.ProductLine != "" {
			pl := row.ProductLine
			productLine = &pl
		}

		unit := ""
		if row.Unit.Valid {
			unit = row.Unit.String
		}

		entries[i] = domain.SalesEntry{
			ID:                  row.ID,
			IssuedAt:            nullTimePtr(row.IssuedAt),
			CompletedAt:         nullTimePtr(row.CompletedAt),
			FirstShipAt:         nullTimePtr(row.FirstShipAt),
			PromisedAt:          nullTimePtr(row.PromisedAt),
			InvoiceDate:         row.InvoiceDate,
			InvoiceID:           row.InvoiceID,
			InvoiceNumber:       row.InvoiceNumber,
			CustomerPO:          nullStringPtr(row.CustomerPo),
			SalesOrderNumber:    row.SalesOrderNumber,
			SalesOrderID:        row.SalesOrderID,
			SalesRepID:          nullStringPtr(row.SalesRepID),
			SalesRepUsername:    nullStringPtr(row.SalesRepUsername),
			CustomerID:          row.CustomerID,
			ParentCustomerID:    nullStringPtr(row.ParentCustomerID),
			CustomerName:        row.CustomerName.String,
			CustomerNumber:      row.CustomerNumber.String,
			CustomerCreatedAt:   customerCreatedAt,
			CustomerTypeGroupID: nullStringPtr(row.CustomerTypeGroupID),
			CustomerGroupName:   nullStringPtr(row.CustomerGroupName),
			ProductLineID:       nullStringPtr(row.ProductLineID),
			ProductTypeCode:     row.ProductTypeCode,
			ItemID:              row.ItemID,
			ProductSku:          row.ProductSku,
			ProductDescription:  nullStringPtr(row.ProductDescription),
			CategoryName:        row.CategoryName,
			ProductLine:         productLine,
			Unit:                unit,
			QuantityInvoiced:    decimalToFloat64(row.QuantityInvoiced),
			TotalInvoiced:       decimalToFloat64(row.TotalInvoiced),
			TotalCost:           decimalToFloat64(row.TotalCost),
			TotalProfit:         decimalToFloat64(row.TotalProfit),
			UnitPrice:           decimalToFloat64(row.UnitPrice),
			UnitCost:            decimalToFloat64(row.UnitCost),
			UnitProfit:          decimalToFloat64(row.UnitProfit),
			ShipToState:         nullStringPtr(row.ShipToState),
			ShipToCity:          nullStringPtr(row.ShipToCity),
			ShipToPostalCode:    nullStringPtr(row.ShipToPostalCode),
			ShipToCountry:       nullStringPtr(row.ShipToCountry),
			OrderDiscountCode:   nullStringPtr(row.OrderDiscountCode),
		}
	}

	return entries, nil
}

func (r *analyticsRepoImpl) GetOpenBatchEntries(ctx context.Context, params domain.AnalyzeOpenBatchesParams) ([]domain.OpenBatchEntry, *apierror.APIError) {
	// Open batches are already handled by the existing batch infrastructure.
	// This is a placeholder that returns empty results.
	return nil, nil
}

func (r *analyticsRepoImpl) GetProductionCostEntries(ctx context.Context, params domain.AnalyzeProductionCostsParams) ([]domain.ProductionCostEntry, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_production_cost_entries")
	defer span.End()

	rows, err := r.queries.GetProductionCostEntries(ctx, params.AccountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	entries := make([]domain.ProductionCostEntry, len(rows))
	for i, row := range rows {
		entries[i] = domain.ProductionCostEntry{
			ItemID:             row.ItemID,
			ProductSku:         row.ProductSku,
			ProductDescription: nullStringPtr(row.ProductDescription),
			ProductLine:        nullStringPtr(row.ProductLine),
			TotalQuantity:      decimalToFloat64(row.TotalQuantity),
			TotalCost:          decimalToFloat64(row.TotalCost),
			CostPerUnit:        decimalToFloat64(row.CostPerUnit),
			Unit:               row.Unit,
		}
	}

	return entries, nil
}

func (r *analyticsRepoImpl) GetDeliveryAnalytics(ctx context.Context, params domain.AnalyzeDeliveriesParams) (*domain.DeliveryAnalyticsResult, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_delivery_analytics")
	defer span.End()

	rows, err := r.queries.GetDeliveryEntries(ctx, sqlc.GetDeliveryEntriesParams{
		OwnerAccountID: params.AccountID,
		StartDate:      params.StartDate,
		EndDate:        params.EndDate,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	entries := make([]domain.DeliveryEntry, len(rows))
	for i, row := range rows {
		invoicedAt := row.InvoicedAt
		entries[i] = domain.DeliveryEntry{
			InvoiceNumber: row.InvoiceNumber,
			InvoicedAt:    &invoicedAt,
			IssuedAt:      nullTimePtr(row.IssuedAt),
			CompletedAt:   nullTimePtr(row.CompletedAt),
			FirstShipAt:   nullTimePtr(row.FirstShipAt),
			PromisedAt:    nullTimePtr(row.PromisedAt),
		}
	}

	// Apply target delivery time if provided (matching dashboard processDeliveryEntryWithTargetTime).
	if params.TargetDeliveryTimeDays != nil && params.OverridePromisedDates != nil && *params.OverridePromisedDates {
		for i := range entries {
			if entries[i].PromisedAt != nil {
				continue
			}
			if entries[i].IssuedAt == nil {
				continue
			}
			targetDate := entries[i].IssuedAt.AddDate(0, 0, int(*params.TargetDeliveryTimeDays))
			entries[i].PromisedAt = &targetDate
		}
	}

	// Compute statistics and chart data in-memory.
	statistics := computeDeliveryStatistics(entries, &params.StartDate, &params.EndDate)
	chartData := computeDeliveryChartData(entries, params.StartDate, params.EndDate, 30)

	return &domain.DeliveryAnalyticsResult{
		Statistics: statistics,
		ChartData:  chartData,
	}, nil
}

func nullTimePtr(nt sql.NullTime) *time.Time {
	if !nt.Valid {
		return nil
	}
	t := nt.Time
	return &t
}

type invoiceDateSummary struct {
	issuedAt    *time.Time
	invoicedAt  *time.Time
	firstShipAt *time.Time
	completedAt *time.Time
	promisedAt  *time.Time
}

func groupByInvoice(entries []domain.DeliveryEntry, startDate, endDate *time.Time) map[string][]domain.DeliveryEntry {
	grouped := make(map[string][]domain.DeliveryEntry)
	for _, e := range entries {
		if startDate != nil && (e.IssuedAt == nil || e.IssuedAt.Before(*startDate)) {
			continue
		}
		if endDate != nil && (e.IssuedAt == nil || !e.IssuedAt.Before(*endDate)) {
			continue
		}
		grouped[e.InvoiceNumber] = append(grouped[e.InvoiceNumber], e)
	}
	return grouped
}

func getInvoiceDateSummary(entries []domain.DeliveryEntry) invoiceDateSummary {
	var s invoiceDateSummary
	for _, e := range entries {
		if e.IssuedAt != nil {
			if s.issuedAt == nil || e.IssuedAt.Before(*s.issuedAt) {
				s.issuedAt = e.IssuedAt
			}
		}
		if e.InvoicedAt != nil {
			if s.invoicedAt == nil || e.InvoicedAt.Before(*s.invoicedAt) {
				s.invoicedAt = e.InvoicedAt
			}
		}
		if e.FirstShipAt != nil {
			if s.firstShipAt == nil || e.FirstShipAt.Before(*s.firstShipAt) {
				s.firstShipAt = e.FirstShipAt
			}
		}
		if e.CompletedAt != nil {
			if s.completedAt == nil || e.CompletedAt.After(*s.completedAt) {
				s.completedAt = e.CompletedAt
			}
		}
		if e.PromisedAt != nil {
			if s.promisedAt == nil || e.PromisedAt.Before(*s.promisedAt) {
				s.promisedAt = e.PromisedAt
			}
		}
	}
	return s
}

func computeDeliveryStatistics(entries []domain.DeliveryEntry, startDate, endDate *time.Time) domain.DeliveryStatistics {
	invoiceGroups := groupByInvoice(entries, startDate, endDate)

	var totalOrders, ordersWithFirstShipment, ordersWithCompletion, ordersWithPromiseDate int32
	var ordersPartiallyFulfilledInPromiseDate, ordersCompletedWithinPromiseDate int32

	var totalDaysToFirstShipment, totalDaysToCompletion float64
	var validFirstShipment, validCompletion int32
	var totalOnTimeDelivery, onTimeDeliveries int32
	var totalOnTimeFirstShipment, onTimeFirstShipments int32

	for _, group := range invoiceGroups {
		if len(group) == 0 {
			continue
		}
		summary := getInvoiceDateSummary(group)
		totalOrders++

		firstEntry := group[0]
		if firstEntry.FirstShipAt != nil {
			ordersWithFirstShipment++
		}
		if firstEntry.CompletedAt != nil {
			ordersWithCompletion++
		}
		if firstEntry.PromisedAt != nil {
			ordersWithPromiseDate++
		}

		if summary.issuedAt != nil && summary.firstShipAt != nil {
			deliveryTime := summary.firstShipAt.Sub(*summary.issuedAt)
			if deliveryTime > 0 {
				totalDaysToFirstShipment += deliveryTime.Hours() / 24.0
				validFirstShipment++
			}
		}

		if summary.issuedAt != nil && summary.invoicedAt != nil {
			deliveryTime := summary.invoicedAt.Sub(*summary.issuedAt)
			if deliveryTime > 0 {
				totalDaysToCompletion += deliveryTime.Hours() / 24.0
				validCompletion++
			}
		}

		if summary.promisedAt != nil && summary.completedAt != nil {
			if summary.issuedAt == nil || summary.completedAt.After(*summary.issuedAt) {
				totalOnTimeDelivery++
				if !summary.completedAt.After(*summary.promisedAt) {
					onTimeDeliveries++
				}
			}
		}

		if summary.promisedAt != nil && summary.firstShipAt != nil {
			if summary.issuedAt == nil || summary.firstShipAt.After(*summary.issuedAt) {
				totalOnTimeFirstShipment++
				if !summary.firstShipAt.After(*summary.promisedAt) {
					onTimeFirstShipments++
				}
			}
		}

		if summary.promisedAt != nil {
			if summary.completedAt != nil && !summary.completedAt.After(*summary.promisedAt) {
				ordersCompletedWithinPromiseDate++
			} else if summary.firstShipAt != nil && !summary.firstShipAt.After(*summary.promisedAt) && summary.completedAt == nil {
				ordersPartiallyFulfilledInPromiseDate++
			}
		}
	}

	stats := domain.DeliveryStatistics{
		TotalOrders:                           totalOrders,
		OrdersWithFirstShipment:               ordersWithFirstShipment,
		OrdersWithCompletion:                  ordersWithCompletion,
		OrdersWithPromiseDate:                 ordersWithPromiseDate,
		OrdersPartiallyFulfilledInPromiseDate: ordersPartiallyFulfilledInPromiseDate,
		OrdersCompletedWithinPromiseDate:      ordersCompletedWithinPromiseDate,
	}

	if validFirstShipment > 0 {
		avg := totalDaysToFirstShipment / float64(validFirstShipment)
		stats.AverageTimeToFirstShipment = &avg
	}
	if validCompletion > 0 {
		avg := totalDaysToCompletion / float64(validCompletion)
		stats.AverageTimeToCompletion = &avg
	}
	if totalOnTimeDelivery > 0 {
		pct := float64(onTimeDeliveries) / float64(totalOnTimeDelivery) * 100
		stats.OnTimeDeliveryPercentage = &pct
	}
	if totalOnTimeFirstShipment > 0 {
		pct := float64(onTimeFirstShipments) / float64(totalOnTimeFirstShipment) * 100
		stats.OnTimeFirstShipmentPercentage = &pct
	}

	return stats
}

func computeDeliveryChartData(entries []domain.DeliveryEntry, startDate, endDate time.Time, numberOfPoints int) domain.DeliveryChartData {
	intervalMs := float64(endDate.Sub(startDate).Milliseconds()) / float64(numberOfPoints)

	var onTimeData, avgDeliveryData, avgFirstShipData []domain.ChartDataPoint

	for i := 0; i < numberOfPoints; i++ {
		intervalStart := startDate.Add(time.Duration(float64(i)*intervalMs) * time.Millisecond)
		intervalEnd := startDate.Add(time.Duration(float64(i+1)*intervalMs) * time.Millisecond)
		xValue := float64(intervalStart.UnixMilli())

		if pct := computeOnTimeDeliveryPct(entries, &intervalStart, &intervalEnd); pct != nil {
			onTimeData = append(onTimeData, domain.ChartDataPoint{X: xValue, Y: *pct})
		}
		if avg := computeAvgDeliveryTimeToCompletion(entries, &intervalStart, &intervalEnd); avg != nil {
			avgDeliveryData = append(avgDeliveryData, domain.ChartDataPoint{X: xValue, Y: *avg})
		}
		if avg := computeAvgDeliveryTimeToFirstShipment(entries, &intervalStart, &intervalEnd); avg != nil {
			avgFirstShipData = append(avgFirstShipData, domain.ChartDataPoint{X: xValue, Y: *avg})
		}
	}

	if onTimeData == nil {
		onTimeData = []domain.ChartDataPoint{}
	}
	if avgDeliveryData == nil {
		avgDeliveryData = []domain.ChartDataPoint{}
	}
	if avgFirstShipData == nil {
		avgFirstShipData = []domain.ChartDataPoint{}
	}

	return domain.DeliveryChartData{
		OnTimeDelivery:           onTimeData,
		AverageDeliveryTime:      avgDeliveryData,
		AverageFirstShipmentTime: avgFirstShipData,
	}
}

func computeOnTimeDeliveryPct(entries []domain.DeliveryEntry, startDate, endDate *time.Time) *float64 {
	invoiceGroups := groupByInvoice(entries, startDate, endDate)
	var total, onTime int32
	for _, group := range invoiceGroups {
		if len(group) == 0 {
			continue
		}
		summary := getInvoiceDateSummary(group)
		if summary.promisedAt == nil || summary.completedAt == nil {
			continue
		}
		if summary.issuedAt != nil && !summary.completedAt.After(*summary.issuedAt) {
			continue
		}
		total++
		if !summary.completedAt.After(*summary.promisedAt) {
			onTime++
		}
	}
	if total == 0 {
		return nil
	}
	pct := float64(onTime) / float64(total) * 100
	return &pct
}

func computeAvgDeliveryTimeToCompletion(entries []domain.DeliveryEntry, startDate, endDate *time.Time) *float64 {
	invoiceGroups := groupByInvoice(entries, startDate, endDate)
	var totalDays float64
	var valid int32
	for _, group := range invoiceGroups {
		if len(group) == 0 {
			continue
		}
		summary := getInvoiceDateSummary(group)
		if summary.issuedAt == nil || summary.invoicedAt == nil {
			continue
		}
		deliveryTime := summary.invoicedAt.Sub(*summary.issuedAt)
		if deliveryTime <= 0 {
			continue
		}
		totalDays += deliveryTime.Hours() / 24.0
		valid++
	}
	if valid == 0 {
		return nil
	}
	avg := totalDays / float64(valid)
	return &avg
}

func computeAvgDeliveryTimeToFirstShipment(entries []domain.DeliveryEntry, startDate, endDate *time.Time) *float64 {
	invoiceGroups := groupByInvoice(entries, startDate, endDate)
	var totalDays float64
	var valid int32
	for _, group := range invoiceGroups {
		if len(group) == 0 {
			continue
		}
		summary := getInvoiceDateSummary(group)
		if summary.issuedAt == nil || summary.firstShipAt == nil {
			continue
		}
		deliveryTime := summary.firstShipAt.Sub(*summary.issuedAt)
		if deliveryTime <= 0 {
			continue
		}
		totalDays += deliveryTime.Hours() / 24.0
		valid++
	}
	if valid == 0 {
		return nil
	}
	avg := totalDays / float64(valid)
	return &avg
}

func (r *analyticsRepoImpl) GetManufacturingMetric(ctx context.Context, params domain.AnalyzeManufacturingParams) (float64, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_manufacturing_metric")
	defer span.End()

	switch params.Type {
	case "production":
		row, err := r.queries.GetManufacturingProduction(ctx, sqlc.GetManufacturingProductionParams{
			OwnerAccountID: params.AccountID,
			StartDate:      toRequiredNullTime(params.StartDate),
			EndDate:        toRequiredNullTime(params.EndDate),
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return 0, tracing.Trace(span, apiErr)
		}
		return decimalToFloat64(row), nil

	case "costsPerUnit":
		row, err := r.queries.GetManufacturingCostsPerUnit(ctx, sqlc.GetManufacturingCostsPerUnitParams{
			OwnerAccountID: params.AccountID,
			StartDate:      params.StartDate,
			EndDate:        params.EndDate,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return 0, tracing.Trace(span, apiErr)
		}
		totalCost := decimalToFloat64(row.TotalCost)
		totalQuantity := decimalToFloat64(row.TotalQuantity)
		if totalQuantity > 0 {
			return totalCost / totalQuantity, nil
		}
		return 0, nil

	case "margin":
		row, err := r.queries.GetManufacturingMargin(ctx, sqlc.GetManufacturingMarginParams{
			OwnerAccountID: params.AccountID,
			StartDate:      params.StartDate,
			EndDate:        params.EndDate,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return 0, tracing.Trace(span, apiErr)
		}
		totalRevenue := decimalToFloat64(row.TotalInvoiced)
		totalProfit := decimalToFloat64(row.TotalProfit)
		if totalRevenue > 0 {
			return totalProfit / totalRevenue, nil
		}
		return 0, nil

	case "quality":
		row, err := r.queries.GetManufacturingQuality(ctx, sqlc.GetManufacturingQualityParams{
			OwnerAccountID: params.AccountID,
			StartDate:      toRequiredNullTime(params.StartDate),
			EndDate:        toRequiredNullTime(params.EndDate),
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return 0, tracing.Trace(span, apiErr)
		}
		return decimalToFloat64(row), nil

	case "laborEfficiency":
		row, err := r.queries.GetManufacturingLaborEfficiency(ctx, sqlc.GetManufacturingLaborEfficiencyParams{
			OwnerAccountID: params.AccountID,
			StartDate:      toRequiredNullTime(params.StartDate),
			EndDate:        toRequiredNullTime(params.EndDate),
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return 0, tracing.Trace(span, apiErr)
		}
		laborQty := decimalToFloat64(row.LaborQuantity)
		laborWaste := decimalToFloat64(row.LaborWaste)
		laborSeconds := decimalToFloat64(row.LaborSeconds)
		overallTotal := laborQty + laborWaste + laborSeconds
		if overallTotal > 0 {
			return laborQty / overallTotal, nil
		}
		return 0, nil

	default:
		return 0, tracing.Trace(span, apierror.NewValidationErrorWithParam(
			fmt.Sprintf("Unsupported manufacturing analytics type: %s", params.Type), "type"))
	}
}

func (r *analyticsRepoImpl) GetManufacturingBatch(ctx context.Context, params domain.AnalyzeManufacturingBatchParams) (*domain.ManufacturingBatchResult, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_manufacturing_batch")
	defer span.End()

	currentMetrics, apiErr := r.getManufacturingMetricsForPeriod(ctx, span, params.AccountID, params.StartDate, params.EndDate)
	if apiErr != nil {
		return nil, apiErr
	}

	compMetrics, apiErr := r.getManufacturingMetricsForPeriod(ctx, span, params.AccountID, params.ComparisonStartDate, params.ComparisonEndDate)
	if apiErr != nil {
		return nil, apiErr
	}

	return &domain.ManufacturingBatchResult{
		Current:    currentMetrics,
		Comparison: compMetrics,
	}, nil
}

func (r *analyticsRepoImpl) getManufacturingMetricsForPeriod(ctx context.Context, span trace.Span, accountID string, startDate, endDate time.Time) (domain.ManufacturingMetrics, *apierror.APIError) {
	// Query A: batch metrics (production, quality, labor efficiency)
	batchRow, err := r.queries.GetManufacturingBatchBatchMetrics(ctx, sqlc.GetManufacturingBatchBatchMetricsParams{
		OwnerAccountID: accountID,
		StartDate:      toRequiredNullTime(startDate),
		EndDate:        toRequiredNullTime(endDate),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return domain.ManufacturingMetrics{}, tracing.Trace(span, apiErr)
	}

	// Query B: invoice metrics (costs per unit, margin)
	invoiceRow, err := r.queries.GetManufacturingBatchInvoiceMetrics(ctx, sqlc.GetManufacturingBatchInvoiceMetricsParams{
		OwnerAccountID: accountID,
		StartDate:      startDate,
		EndDate:        endDate,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return domain.ManufacturingMetrics{}, tracing.Trace(span, apiErr)
	}

	// Production: total quantity from batches
	totalQuantity := decimalToFloat64(batchRow.TotalQuantity)
	totalWaste := decimalToFloat64(batchRow.TotalWaste)
	totalSeconds := decimalToFloat64(batchRow.TotalSeconds)

	// Quality: ratio of good units to total units (good + waste + seconds)
	qualTotal := totalQuantity + totalWaste + totalSeconds
	var quality float64
	if qualTotal > 0 {
		quality = totalQuantity / qualTotal
	}

	// Labor efficiency: ratio of labor-adjusted good units to total labor-adjusted units
	laborQuantity := decimalToFloat64(batchRow.LaborQuantity)
	laborWaste := decimalToFloat64(batchRow.LaborWaste)
	laborSeconds := decimalToFloat64(batchRow.LaborSeconds)
	laborTotal := laborQuantity + laborWaste + laborSeconds
	var laborEfficiency float64
	if laborTotal > 0 {
		laborEfficiency = laborQuantity / laborTotal
	}

	// Costs per unit: total cost / total quantity from invoices
	invCost := decimalToFloat64(invoiceRow.TotalCost)
	invQty := decimalToFloat64(invoiceRow.TotalQuantity)
	var costsPerUnit float64
	if invQty > 0 {
		costsPerUnit = invCost / invQty
	}

	// Margin: total profit / total revenue from invoices
	invRevenue := decimalToFloat64(invoiceRow.TotalRevenue)
	invProfit := decimalToFloat64(invoiceRow.TotalProfit)
	var margin float64
	if invRevenue > 0 {
		margin = invProfit / invRevenue
	}

	return domain.ManufacturingMetrics{
		Production:      totalQuantity,
		CostsPerUnit:    costsPerUnit,
		Margin:          margin,
		Quality:         quality,
		LaborEfficiency: laborEfficiency,
	}, nil
}

func (r *analyticsRepoImpl) GetOrderEntries(ctx context.Context, params domain.AnalyzeOrdersParams) ([]domain.OrderEntry, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_order_entries")
	defer span.End()

	salesRepIDs := toNullStringSlice(params.SalesRepIDs)
	if salesRepIDs == nil {
		salesRepIDs = []sql.NullString{}
	}
	customerGroupIDs := toNullStringSlice(params.CustomerGroupIDs)
	if customerGroupIDs == nil {
		customerGroupIDs = []sql.NullString{}
	}
	productLineIDs := toNullStringSlice(params.ProductLineIDs)
	if productLineIDs == nil {
		productLineIDs = []sql.NullString{}
	}
	customerIDs := params.CustomerIDs
	if customerIDs == nil {
		customerIDs = []string{}
	}

	rows, err := r.queries.GetOrderEntries(ctx, sqlc.GetOrderEntriesParams{
		OwnerAccountID:             params.AccountID,
		IncludeSalesRepFilter:      len(params.SalesRepIDs) > 0,
		SalesRepIds:                salesRepIDs,
		IncludeCustomerFilter:      len(params.CustomerIDs) > 0,
		CustomerIds:                customerIDs,
		IncludeCustomerGroupFilter: len(params.CustomerGroupIDs) > 0,
		CustomerGroupIds:           customerGroupIDs,
		IncludeProductLineFilter:   len(params.ProductLineIDs) > 0,
		ProductLineIds:             productLineIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	entries := make([]domain.OrderEntry, len(rows))
	for i, row := range rows {
		var customerCreatedAt time.Time
		if row.CustomerCreatedAt.Valid {
			customerCreatedAt = row.CustomerCreatedAt.Time
		}

		entries[i] = domain.OrderEntry{
			ID:                  row.ID,
			IssuedAt:            nullTimePtr(row.IssuedAt),
			CompletedAt:         nullTimePtr(row.CompletedAt),
			FirstShipAt:         nullTimePtr(row.FirstShipAt),
			PromisedAt:          nullTimePtr(row.PromisedAt),
			CustomerPO:          nullStringPtr(row.CustomerPo),
			OrderNumber:         row.OrderNumber.String,
			OrderID:             row.OrderID.String,
			SalesRepID:          nullStringPtr(row.SalesRepID),
			SalesRepUsername:    nullStringPtr(row.SalesRepUsername),
			CustomerID:          row.CustomerID.String,
			ParentCustomerID:    nullStringPtr(row.ParentCustomerID),
			CustomerName:        row.CustomerName.String,
			CustomerNumber:      row.CustomerNumber.String,
			CustomerCreatedAt:   customerCreatedAt,
			CustomerTypeGroupID: nullStringPtr(row.CustomerTypeGroupID),
			CustomerGroupName:   nullStringPtr(row.CustomerGroupName),
			ProductLineID:       nullStringPtr(row.ProductLineID),
			ProductTypeCode:     row.ProductTypeCode.String,
			ItemID:              row.ItemID.String,
			ProductSku:          row.ProductSku.String,
			ProductDescription:  nullStringPtr(row.ProductDescription),
			CategoryName:        row.CategoryName.String,
			ProductLine:         nullStringPtr(row.ProductLine),
			QuantityOrdered:     decimalToFloat64(row.QuantityOrdered),
			QuantityInvoiced:    decimalToFloat64(row.QuantityInvoiced),
			QuantityBackOrdered: decimalToFloat64(row.QuantityBackOrdered),
			Unit:                row.Unit.String,
			UnitCost:            decimalToFloat64(row.UnitCost),
			UnitPrice:           decimalToFloat64(row.UnitPrice),
			UnitProfit:          decimalToFloat64(row.UnitProfit),
			TotalInvoiced:       decimalToFloat64(row.TotalInvoiced),
			TotalCost:           decimalToFloat64(row.TotalCost),
			TotalProfit:         decimalToFloat64(row.TotalProfit),
			TotalOrdered:        decimalToFloat64(row.TotalOrdered),
			TotalBackOrdered:    decimalToFloat64(row.TotalBackOrdered),
			ShipToState:         nullStringPtr(row.ShipToState),
			ShipToCity:          nullStringPtr(row.ShipToCity),
			ShipToZipcode:       nullStringPtr(row.ShipToZipcode),
			ShipToCountry:       nullStringPtr(row.ShipToCountry),
			OrderDiscountCode:   nullStringPtr(row.OrderDiscountCode),
		}
	}

	return entries, nil
}

func (r *analyticsRepoImpl) GetQuarterlyOrders(ctx context.Context, params domain.AnalyzeQuarterlyOrdersParams) ([]domain.YearlyQuarterlyData, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_quarterly_orders")
	defer span.End()

	salesRepIDs := toNullStringSlice(params.SalesRepIDs)
	productLineIDs := toNullStringSlice(params.ProductLineIDs)
	customerGroupIDs := toNullStringSlice(params.CustomerGroupIDs)

	rows, err := r.queries.GetQuarterlyOrderTotals(ctx, sqlc.GetQuarterlyOrderTotalsParams{
		OwnerAccountID:             params.AccountID,
		IncludeCustomerFilter:      len(params.CustomerIDs) > 0,
		CustomerIds:                params.CustomerIDs,
		IncludeSalesRepFilter:      len(params.SalesRepIDs) > 0,
		SalesRepIds:                salesRepIDs,
		IncludeProductLineFilter:   len(params.ProductLineIDs) > 0,
		ProductLineIds:             productLineIDs,
		IncludeItemFilter:          len(params.ItemIDs) > 0,
		ItemIds:                    params.ItemIDs,
		IncludeCustomerGroupFilter: len(params.CustomerGroupIDs) > 0,
		CustomerGroupIds:           customerGroupIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	result := make([]domain.YearlyQuarterlyData, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.YearlyQuarterlyData{
			Year: row.OrderYear,
			Data: domain.QuarterlyData{
				Q1:    decimalToFloat64(row.Q1),
				Q2:    decimalToFloat64(row.Q2),
				Q3:    decimalToFloat64(row.Q3),
				Q4:    decimalToFloat64(row.Q4),
				Total: decimalToFloat64(row.Total),
			},
		})
	}

	return result, nil
}

func (r *analyticsRepoImpl) GetMaterialAnalytics(ctx context.Context, params domain.AnalyzeMaterialsParams) ([]domain.MaterialAnalyticsEntry, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_material_analytics")
	defer span.End()

	// 1. Fetch materials with full details.
	materials, err := r.queries.GetMaterialsWithDetails(ctx, params.AccountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if len(materials) == 0 {
		return []domain.MaterialAnalyticsEntry{}, nil
	}

	// 2. Collect item IDs and unit group IDs.
	itemIDs := make([]string, len(materials))
	unitGroupIDSet := make(map[string]bool)
	for i, m := range materials {
		itemIDs[i] = m.ItemID
		unitGroupIDSet[m.UnitGroupID] = true
	}
	unitGroupIDs := make([]string, 0, len(unitGroupIDSet))
	for id := range unitGroupIDSet {
		unitGroupIDs = append(unitGroupIDs, id)
	}

	// 3. Fetch inventory quantities per item.
	onHandRows, err := r.queries.GetMaterialOnHandByItem(ctx, sqlc.GetMaterialOnHandByItemParams{
		AccountID: params.AccountID,
		ItemIds:   itemIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	reservedRows, err := r.queries.GetMaterialReservedByItem(ctx, sqlc.GetMaterialReservedByItemParams{
		AccountID: params.AccountID,
		ItemIds:   itemIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	openRows, err := r.queries.GetMaterialOpenByItem(ctx, sqlc.GetMaterialOpenByItemParams{
		AccountID: params.AccountID,
		ItemIds:   itemIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// 4. Fetch unit group units.
	unitGroupUnitRows, err := r.queries.GetMaterialUnitGroupUnits(ctx, unitGroupIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// 5. Build lookup maps.
	onHandMap := make(map[string]float64)
	for _, row := range onHandRows {
		onHandMap[row.ItemID] = decimalToFloat64(row.RemainingQuantity)
	}
	reservedMap := make(map[string]float64)
	for _, row := range reservedRows {
		reservedMap[row.ItemID] = decimalToFloat64(row.RemainingQuantity)
	}
	openMap := make(map[string]float64)
	for _, row := range openRows {
		openMap[row.ItemID] = decimalToFloat64(row.RemainingQuantity)
	}

	ugUnitsMap := make(map[string][]domain.MaterialUnitGroupUnit)
	for _, u := range unitGroupUnitRows {
		ugUnitsMap[u.UnitGroupID] = append(ugUnitsMap[u.UnitGroupID], domain.MaterialUnitGroupUnit{
			ID:               u.UnitID,
			Name:             u.UnitName,
			Abbreviation:     u.UnitAbbreviation,
			ConversionFactor: decimalToFloat64(u.ConversionFactor),
			IsBaseUnit:       u.IsBaseUnit,
		})
	}

	// 6. Optionally fetch supplier info.
	type supplierEntry struct {
		Name       string
		PartNumber string
	}
	supplierMap := make(map[string][]supplierEntry)
	if len(params.SupplierIDs) > 0 {
		supplierRows, sErr := r.queries.GetMaterialSupplierInfo(ctx, sqlc.GetMaterialSupplierInfoParams{
			OwnerAccountID: params.AccountID,
			SupplierIds:    params.SupplierIDs,
		})
		if apiErr := db.MapSQLError(sErr); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		for _, row := range supplierRows {
			supplierMap[row.ItemID] = append(supplierMap[row.ItemID], supplierEntry{
				Name:       row.SupplierName,
				PartNumber: row.SupplierPartNumber,
			})
		}
	}

	// 7. Build results matching dashboard behavior.
	entries := make([]domain.MaterialAnalyticsEntry, len(materials))
	for i, m := range materials {
		onHand := onHandMap[m.ItemID]
		reserved := reservedMap[m.ItemID]
		open := openMap[m.ItemID]
		availableToPromise := onHand - reserved - open

		opValue := decimalToFloat64(m.OrderPointValue)
		orderPoint := domain.MaterialBaseQuantity{
			Measure:          opValue,
			UnitName:         m.OrderPointUnitName,
			UnitAbbreviation: m.OrderPointUnitAbbreviation,
			UnitType:         m.OrderPointUnitType,
		}

		ltValue := decimalToFloat64(m.LeadTimeValue)
		leadTime := domain.MaterialBaseQuantity{
			Measure:          ltValue,
			UnitName:         m.LeadTimeUnitName,
			UnitAbbreviation: m.LeadTimeUnitAbbreviation,
			UnitType:         m.LeadTimeUnitType,
		}

		// Normalize inventory and demand to order point unit (matches dashboard BaseQuantityUtils.updateUnit).
		inventoryQty := domain.MaterialBaseQuantity{
			Measure:          availableToPromise,
			UnitName:         orderPoint.UnitName,
			UnitAbbreviation: orderPoint.UnitAbbreviation,
			UnitType:         orderPoint.UnitType,
		}
		demandQty := domain.MaterialBaseQuantity{
			Measure:          open,
			UnitName:         orderPoint.UnitName,
			UnitAbbreviation: orderPoint.UnitAbbreviation,
			UnitType:         orderPoint.UnitType,
		}

		// Get supplier info for this item.
		supplierNames := []string{}
		supplierPartNumbers := []string{}
		if suppliers, ok := supplierMap[m.ItemID]; ok {
			for _, s := range suppliers {
				supplierNames = append(supplierNames, s.Name)
				supplierPartNumbers = append(supplierPartNumbers, s.PartNumber)
			}
		}

		units := ugUnitsMap[m.UnitGroupID]
		if units == nil {
			units = []domain.MaterialUnitGroupUnit{}
		}

		entries[i] = domain.MaterialAnalyticsEntry{
			MaterialID:          m.MaterialID,
			ItemID:              m.ItemID,
			Sku:                 m.ItemSku,
			Description:         nullStringPtr(m.ItemDescription),
			QuantityInInventory: inventoryQty,
			OrderPoint:          &orderPoint,
			LeadTime:            &leadTime,
			QuantityInDemand:    demandQty,
			UnitGroup: domain.MaterialUnitGroup{
				ID:    m.UnitGroupID,
				Name:  m.UnitGroupName,
				Units: units,
			},
			SupplierNames:       supplierNames,
			SupplierPartNumbers: supplierPartNumbers,
		}
	}

	return entries, nil
}

func (r *analyticsRepoImpl) GetInventoryReceiptAnalytics(ctx context.Context, params domain.AnalyzeInventoryReceiptsParams) ([]domain.InventoryReceiptEntry, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_inventory_receipt_analytics")
	defer span.End()

	rows, err := r.queries.GetInventoryReceiptEntries(ctx, sqlc.GetInventoryReceiptEntriesParams{
		RequestingAccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Build filter sets for optional filtering (sqlc doesn't support dynamic WHERE with optional slices).
	itemFilter := make(map[string]bool, len(params.ItemIDs))
	for _, id := range params.ItemIDs {
		itemFilter[id] = true
	}
	locationFilter := make(map[string]bool, len(params.LocationIDs))
	for _, id := range params.LocationIDs {
		locationFilter[id] = true
	}
	lotFilter := make(map[string]bool, len(params.LotIDs))
	for _, id := range params.LotIDs {
		lotFilter[id] = true
	}

	var entries []domain.InventoryReceiptEntry
	for _, row := range rows {
		// Apply optional filters.
		if len(itemFilter) > 0 && !itemFilter[row.ItemID] {
			continue
		}
		if len(locationFilter) > 0 {
			if !row.StorageLocationID.Valid || !locationFilter[row.StorageLocationID.String] {
				continue
			}
		}
		if len(lotFilter) > 0 {
			if !row.LotID.Valid || !lotFilter[row.LotID.String] {
				continue
			}
		}

		entry := domain.InventoryReceiptEntry{
			ItemID:                          row.ItemID,
			ProductSku:                      row.ProductSku,
			ProductDescription:              nullStringPtr(row.ProductDescription),
			LocationID:                      nullStringPtr(row.StorageLocationID),
			LocationName:                    nullStringPtr(row.StorageLocationName),
			LotID:                           nullStringPtr(row.LotID),
			LotNumber:                       nullStringPtr(row.LotNumber),
			OwnerAccountID:                  row.OwnerAccountID,
			OwnerAccountName:                row.OwnerAccountName,
			HolderAccountID:                 row.HolderAccountID,
			HolderAccountName:               row.HolderAccountName,
			RemainingQuantity:               decimalToFloat64(row.RemainingQuantity),
			WeightedAverageUnitCost:         decimalToFloat64(row.WeightedAverageUnitCost),
			InventoryValue:                  decimalToFloat64(row.InventoryValue),
			OldestReceiptAt:                 interfaceToTimePtr(row.OldestReceiptAt),
			NewestReceiptAt:                 interfaceToTimePtr(row.NewestReceiptAt),
			Unit:                            row.Unit,
			UnitName:                        row.UnitName,
			CostNumeratorUnitAbbreviation:   row.CostNumeratorUnitAbbreviation,
			CostNumeratorUnitName:           row.CostNumeratorUnitName,
			CostDenominatorUnitAbbreviation: row.CostDenominatorUnitAbbreviation,
			CostDenominatorUnitName:         row.CostDenominatorUnitName,
		}

		entries = append(entries, entry)
	}

	if entries == nil {
		entries = []domain.InventoryReceiptEntry{}
	}

	return entries, nil
}

func (r *analyticsRepoImpl) GetNewCustomerEntries(ctx context.Context, params domain.GetNewCustomersAnalyticsParams) ([]domain.NewCustomerEntry, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_new_customer_entries")
	defer span.End()

	customerGroupIDs := toNullStringSlice(params.CustomerGroupIDs)
	if customerGroupIDs == nil {
		customerGroupIDs = []sql.NullString{}
	}
	priceGroupIDs := params.CustomerGroupIDs
	if priceGroupIDs == nil {
		priceGroupIDs = []string{}
	}
	salesRepIDs := toNullStringSlice(params.SalesRepIDs)
	if salesRepIDs == nil {
		salesRepIDs = []sql.NullString{}
	}

	rows, err := r.queries.GetNewCustomerEntries(ctx, sqlc.GetNewCustomerEntriesParams{
		OwnerAccountID:             params.AccountID,
		StartDate:                  params.StartDate,
		EndDate:                    params.EndDate,
		IncludeCustomerGroupFilter: len(params.CustomerGroupIDs) > 0,
		CustomerGroupIds:           customerGroupIDs,
		PriceGroupIds:              priceGroupIDs,
		IncludeSalesRepFilter:      len(params.SalesRepIDs) > 0,
		SalesRepIds:                salesRepIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	entries := make([]domain.NewCustomerEntry, len(rows))
	for i, row := range rows {
		entries[i] = domain.NewCustomerEntry{
			CreatedAt: row,
		}
	}

	return entries, nil
}

func (r *analyticsRepoImpl) GetDemandForecast(ctx context.Context, params domain.GetDemandForecastParams) (*domain.DemandForecastResult, *apierror.APIError) {
	return r.getDemandForecastImpl(ctx, params)
}

func (r *analyticsRepoImpl) GetOeeByDepartment(ctx context.Context, params domain.AnalyzeOeeParams) ([]domain.OeeDepartment, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_oee_by_department")
	defer span.End()

	dateParams := sqlc.GetOeeDepartmentDataParams{
		OwnerAccountID: params.AccountID,
		StartDate:      toRequiredNullTime(params.StartDate),
		EndDate:        toRequiredNullTime(params.EndDate),
	}

	rows, err := r.queries.GetOeeDepartmentData(ctx, dateParams)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	runtimeRows, err := r.queries.GetOeeEstimatedRuntime(ctx, sqlc.GetOeeEstimatedRuntimeParams{
		OwnerAccountID: params.AccountID,
		StartDate:      toRequiredNullTime(params.StartDate),
		EndDate:        toRequiredNullTime(params.EndDate),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Build runtime map: departmentID -> runtimeSeconds.
	runtimeMap := make(map[string]float64, len(runtimeRows))
	for _, row := range runtimeRows {
		runtimeMap[row.DepartmentID] = decimalToFloat64(row.RuntimeSeconds)
	}

	// Build department filter set for optional filtering.
	deptFilter := make(map[string]bool, len(params.DepartmentIDs))
	for _, id := range params.DepartmentIDs {
		deptFilter[id] = true
	}

	var departments []domain.OeeDepartment
	for _, row := range rows {
		if len(deptFilter) > 0 && !deptFilter[row.DepartmentID] {
			continue
		}

		runtimeSeconds := runtimeMap[row.DepartmentID]

		departments = append(departments, domain.OeeDepartment{
			DepartmentID:          row.DepartmentID,
			DepartmentName:        row.DepartmentName,
			GoodUnits:             decimalToFloat64(row.GoodUnits),
			WasteUnits:            decimalToFloat64(row.WasteUnits),
			SecondsUnits:          decimalToFloat64(row.SecondsUnits),
			EstimatedRuntimeHours: runtimeSeconds / 3600,
		})
	}

	if departments == nil {
		departments = []domain.OeeDepartment{}
	}

	return departments, nil
}

func (r *analyticsRepoImpl) GetWeeksOfSales(ctx context.Context, params domain.AnalyzeWeeksOfSalesParams) (*domain.WeeksOfSalesResult, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_weeks_of_sales")
	defer span.End()

	// 1. Get sale-type product item IDs and their product line IDs.
	productItems, err := r.queries.GetSaleProductItemIDs(ctx, params.AccountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if len(productItems) == 0 {
		return &domain.WeeksOfSalesResult{Items: nil, Count: 0}, nil
	}

	// 2. Build maps: productLine -> []itemID, collect unique product line IDs.
	itemsByProductLine := make(map[string][]string)
	uniquePLIDs := make(map[string]bool)
	var allItemIDs []string
	for _, pi := range productItems {
		if pi.ProductLineID.Valid {
			plID := pi.ProductLineID.String
			itemsByProductLine[plID] = append(itemsByProductLine[plID], pi.ItemID)
			uniquePLIDs[plID] = true
		}
		allItemIDs = append(allItemIDs, pi.ItemID)
	}

	plIDs := make([]string, 0, len(uniquePLIDs))
	for plID := range uniquePLIDs {
		plIDs = append(plIDs, plID)
	}
	if len(plIDs) == 0 {
		return &domain.WeeksOfSalesResult{Items: nil, Count: 0}, nil
	}

	// 3. Get product line info (names).
	plInfoRows, err := r.queries.GetProductLineInfo(ctx, sqlc.GetProductLineInfoParams{
		ProductLineIds: plIDs,
		OwnerAccountID: sql.NullString{String: params.AccountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// 4. Get on-hand inventory for all items.
	inventoryRows, err := r.queries.FetchOnHandInventoryBulk(ctx, sqlc.FetchOnHandInventoryBulkParams{
		ItemIds:        allItemIDs,
		OwnerAccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Build inventory map: itemID -> onHandQuantity.
	invMap := make(map[string]float64)
	for _, row := range inventoryRows {
		invMap[row.ItemID] = float64(row.OnHandQuantity)
	}

	// 5. For each product line, compute metrics.
	weeks := params.PeriodInWeeks
	if weeks < 1 {
		weeks = 4
	}
	endDate := time.Now()
	startDate := endDate.Add(-time.Duration(weeks) * 7 * 24 * time.Hour)

	var items []domain.WeeksOfSalesItem
	for _, plInfo := range plInfoRows {
		// Get order quantity for this product line in the period.
		orderRow, err := r.queries.GetOrderQuantityByProductLine(ctx, sqlc.GetOrderQuantityByProductLineParams{
			TargetProductLineID: plInfo.ID,
			OwnerAccountID:      params.AccountID,
			StartDate:           sql.NullTime{Time: startDate, Valid: true},
			EndDate:             sql.NullTime{Time: endDate, Valid: true},
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		totalDemand := decimalToFloat64(orderRow.TotalQuantity)
		unitAbbrev := fmt.Sprintf("%v", orderRow.UnitAbbreviation)
		unitType := fmt.Sprintf("%v", orderRow.UnitType)

		// Sum on-hand for items in this product line.
		itemIDsForLine := itemsByProductLine[plInfo.ID]
		var onHand float64
		for _, iid := range itemIDsForLine {
			onHand += invMap[iid]
		}

		avgSales := totalDemand / float64(weeks)
		var wos float64
		if avgSales > 0 {
			wos = onHand / avgSales
		}

		items = append(items, domain.WeeksOfSalesItem{
			ProductLineID:                        plInfo.ID,
			ProductLineName:                      plInfo.Name,
			QuantityOnHand:                       onHand,
			QuantityOnHandUnitAbbreviation:       unitAbbrev,
			QuantityOnHandUnitType:               unitType,
			AverageSalesQuantity:                 avgSales,
			AverageSalesQuantityUnitAbbreviation: unitAbbrev,
			AverageSalesQuantityUnitType:         unitType,
			WeeksOfSales:                         wos,
		})
	}

	return &domain.WeeksOfSalesResult{
		Items: items,
		Count: int64(len(items)),
	}, nil
}

// decimalToFloat64 converts a decimal string (from CAST AS DECIMAL) to float64.
func decimalToFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case string:
		f := 0.0
		negative := false
		decimal := false
		divisor := 1.0
		for _, c := range val {
			if c == '-' {
				negative = true
			} else if c == '.' {
				decimal = true
			} else if c >= '0' && c <= '9' {
				if decimal {
					divisor *= 10
					f += float64(c-'0') / divisor
				} else {
					f = f*10 + float64(c-'0')
				}
			}
		}
		if negative {
			f = -f
		}
		return f
	case float64:
		return val
	case int32:
		return float64(val)
	case int64:
		return float64(val)
	case []uint8:
		return decimalToFloat64(string(val))
	default:
		return 0
	}
}

func interfaceToTimePtr(v interface{}) *time.Time {
	switch val := v.(type) {
	case time.Time:
		return &val
	case *time.Time:
		return val
	default:
		return nil
	}
}
