package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
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
	buyers, filtered, apiErr := resolveCustomerBuyers(ctx, r.queries.DB(), params.AccountID, params.CustomerIDs, params.CustomerGroupIDs)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if filtered && len(buyers) == 0 {
		return []domain.SalesEntry{}, nil
	}

	rows, err := r.queries.GetSalesEntries(ctx, sqlc.GetSalesEntriesParams{
		OwnerAccountID:           params.AccountID,
		StartDate:                params.StartDate,
		EndDate:                  params.EndDate,
		IncludeSalesRepFilter:    len(params.SalesRepIDs) > 0,
		SalesRepIds:              salesRepIDs,
		IncludeProductLineFilter: len(params.ProductLineIDs) > 0,
		ProductLineIds:           productLineIDs,
		IncludeBuyerFilter:       filtered,
		BuyerIds:                 buyers,
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
	// Open batches are already handled by the existing batch infrastructure. This is a placeholder that returns empty results.
	return nil, nil
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

	for i := range numberOfPoints {
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

// quarterlyOrdersQuery totals the ordered value of sale lines on sales orders issued since issuedFrom, per year and quarter of issue. Every order counts whatever its status now; an estimate has not been issued.
func quarterlyOrdersQuery(orderIndex, accountID string, issuedFrom time.Time, buyers []string, buyersFiltered bool, salesRepIDs, productLineIDs, itemIDs []string) (string, []any) {
	preds := []string{
		"so.owner_account_id = ?",
		"so.sales_order_type_code = 'sales_order'",
		"so.issued_at >= ?",
		"fg.product_type_code = 'sale'",
	}
	args := []any{accountID, issuedFrom}
	in := func(column string, ids []string) {
		preds = append(preds, column+" IN ("+placeholders(len(ids))+")")
		args = append(args, stringsToAny(ids)...)
	}
	if buyersFiltered {
		in("so.buyer_account_id", buyers)
	}
	if len(salesRepIDs) > 0 {
		in("so.sales_rep_id", salesRepIDs)
	}
	if len(productLineIDs) > 0 {
		in("fg.product_line_id", productLineIDs)
	}
	if len(itemIDs) > 0 {
		in("fg.item_id", itemIDs)
	}
	// The orders drive, through orderIndex: left to choose, the planner starts from a product line's every line ever ordered.
	query := `SELECT STRAIGHT_JOIN
    YEAR(so.issued_at) AS order_year,
    QUARTER(so.issued_at) AS order_quarter,
    ` + obDecimal("SUM("+obTotalOrdered+")") + ` AS total
FROM sales_order so FORCE INDEX (` + orderIndex + `)
JOIN sales_order_line sol ON sol.sales_order_id = so.id
JOIN product fg ON fg.id = sol.product_id
JOIN quantity q_ord ON q_ord.id = sol.quantity_id
JOIN unit u_ord ON u_ord.id = q_ord.unit_id
LEFT JOIN rate r_price ON r_price.id = sol.unit_price_id
LEFT JOIN unit u_price_num ON u_price_num.id = r_price.numerator_unit_id
LEFT JOIN unit u_price_den ON u_price_den.id = r_price.denominator_unit_id
WHERE ` + strings.Join(preds, " AND ") + `
GROUP BY order_year, order_quarter
ORDER BY order_year ASC, order_quarter ASC`
	return query, args
}

func (r *analyticsRepoImpl) GetQuarterlyOrders(ctx context.Context, params domain.AnalyzeQuarterlyOrdersParams) ([]domain.YearlyQuarterlyData, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_quarterly_orders")
	defer span.End()

	buyers, filtered, apiErr := resolveCustomerBuyers(ctx, r.queries.DB(), params.AccountID, params.CustomerIDs, params.CustomerGroupIDs)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if filtered && len(buyers) == 0 {
		return []domain.YearlyQuarterlyData{}, nil
	}

	// The orders are read by the window or by the buyer or rep, whichever reaches fewer.
	orderIndex, apiErr := r.cheapestOrderKey(ctx, append([]keyRange{{
		index: "sales_order_owner_type_issued_idx",
		where: "kso.owner_account_id = ? AND kso.sales_order_type_code = 'sales_order' AND kso.issued_at >= ?",
		args:  []any{params.AccountID, params.IssuedFrom},
	}}, filterOrderKeys(params.AccountID, buyers, filtered, params.SalesRepIDs)...))
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	query, args := quarterlyOrdersQuery(orderIndex, params.AccountID, params.IssuedFrom, buyers, filtered, params.SalesRepIDs, params.ProductLineIDs, params.ItemIDs)
	rows, err := r.queries.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	defer rows.Close()

	result := []domain.YearlyQuarterlyData{}
	for rows.Next() {
		var (
			year, quarter int32
			total         sql.NullString
		)
		if err := rows.Scan(&year, &quarter, &total); err != nil {
			return nil, tracing.Trace(span, db.MapSQLError(err))
		}
		if len(result) == 0 || result[len(result)-1].Year != year {
			result = append(result, domain.YearlyQuarterlyData{Year: year})
		}
		addQuarter(&result[len(result)-1].Data, quarter, decimalToFloat(total))
	}
	if err := rows.Err(); err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}
	return result, nil
}

// addQuarter books one quarter's total; the year's total is the quarters added in order, as the dashboard added them.
func addQuarter(d *domain.QuarterlyData, quarter int32, value float64) {
	switch quarter {
	case 1:
		d.Q1 = value
	case 2:
		d.Q2 = value
	case 3:
		d.Q3 = value
	case 4:
		d.Q4 = value
	}
	d.Total += value
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

	// 5. Build lookup maps, each net in the base unit of the item's dimension.
	onHandMap := make(map[string]decimal.Decimal)
	for _, row := range onHandRows {
		onHandMap[row.ItemID] = parseDecimalOrZero(row.RemainingQuantity)
	}
	reservedMap := make(map[string]decimal.Decimal)
	for _, row := range reservedRows {
		reservedMap[row.ItemID] = parseDecimalOrZero(row.RemainingQuantity)
	}
	openMap := make(map[string]decimal.Decimal)
	for _, row := range openRows {
		openMap[row.ItemID] = parseDecimalOrZero(row.RemainingQuantity)
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

	// 7. Build results matching dashboard behavior: stock and demand are stated in the order point's unit, or in the item's base unit when the order point's row is gone.
	entries := make([]domain.MaterialAnalyticsEntry, len(materials))
	for i, m := range materials {
		onHand := onHandMap[m.ItemID]
		open := openMap[m.ItemID]
		availableToPromise := onHand.Sub(reservedMap[m.ItemID]).Sub(open)

		display := materialUnit{
			name: m.BaseUnitName, abbreviation: m.BaseUnitAbbreviation, dimension: m.BaseUnitType,
			ratioNumerator: m.BaseUnitRatioNumerator, ratioDenominator: m.BaseUnitRatioDenominator,
			offsetNumerator: m.BaseUnitOffsetNumerator, offsetDenominator: m.BaseUnitOffsetDenominator,
		}
		var orderPoint *domain.MaterialBaseQuantity
		if m.OrderPointValue.Valid && m.OrderPointUnitRatioNumerator.Valid {
			display = materialUnit{
				name: m.OrderPointUnitName.String, abbreviation: m.OrderPointUnitAbbreviation.String, dimension: m.OrderPointUnitType.String,
				ratioNumerator: m.OrderPointUnitRatioNumerator.String, ratioDenominator: m.OrderPointUnitRatioDenominator.String,
				offsetNumerator: m.OrderPointUnitOffsetNumerator.String, offsetDenominator: m.OrderPointUnitOffsetDenominator.String,
			}
			orderPoint = &domain.MaterialBaseQuantity{
				Measure:          decimalToFloat64(m.OrderPointValue),
				UnitName:         display.name,
				UnitAbbreviation: display.abbreviation,
				UnitType:         display.dimension,
			}
		}
		var leadTime *domain.MaterialBaseQuantity
		if m.LeadTimeValue.Valid && m.LeadTimeUnitName.Valid {
			leadTime = &domain.MaterialBaseQuantity{
				Measure:          decimalToFloat64(m.LeadTimeValue),
				UnitName:         m.LeadTimeUnitName.String,
				UnitAbbreviation: m.LeadTimeUnitAbbreviation.String,
				UnitType:         m.LeadTimeUnitType.String,
			}
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
			QuantityInInventory: display.quantity(availableToPromise),
			OrderPoint:          orderPoint,
			LeadTime:            leadTime,
			QuantityInDemand:    display.quantity(open),
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

// materialUnit is a unit a material's stock is stated in, with the ratio and offset that convert from its dimension's base unit.
type materialUnit struct {
	name, abbreviation, dimension                                        string
	ratioNumerator, ratioDenominator, offsetNumerator, offsetDenominator string
}

// quantity states an amount held in the dimension's base unit in this unit, as the dashboard's updateUnit converted it.
func (u materialUnit) quantity(base decimal.Decimal) domain.MaterialBaseQuantity {
	measure, _ := fromDimensionBase(base, u.ratioNumerator, u.ratioDenominator, u.offsetNumerator, u.offsetDenominator).Float64()
	return domain.MaterialBaseQuantity{Measure: measure, UnitName: u.name, UnitAbbreviation: u.abbreviation, UnitType: u.dimension}
}

// fromDimensionBase converts an amount in its dimension's base unit into a unit with the given ratio and offset. A malformed or zero ratio leaves the amount as it is.
func fromDimensionBase(base decimal.Decimal, ratioNumerator, ratioDenominator, offsetNumerator, offsetDenominator string) decimal.Decimal {
	rn, rd := parseDecimalOrZero(ratioNumerator), parseDecimalOrZero(ratioDenominator)
	if rn.IsZero() || rd.IsZero() {
		return base
	}
	offset := decimal.Zero
	if od := parseDecimalOrZero(offsetDenominator); !od.IsZero() {
		offset = parseDecimalOrZero(offsetNumerator).DivRound(od, 30)
	}
	return base.Sub(offset).Mul(rd).DivRound(rn, 30)
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

// GetSaleProductItemIDs returns sale-type product item IDs and their product line IDs.
func (r *analyticsRepoImpl) GetSaleProductItemIDs(ctx context.Context, accountID string) ([]domain.SaleProductItemRow, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_sale_product_item_ids")
	defer span.End()

	rows, err := r.queries.GetSaleProductItemIDs(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	result := make([]domain.SaleProductItemRow, len(rows))
	for i, row := range rows {
		result[i] = domain.SaleProductItemRow{
			ItemID:        row.ItemID,
			ProductLineID: nullStringPtr(row.ProductLineID),
		}
	}

	return result, nil
}

// GetProductLineInfo returns id and name for the given product lines.
func (r *analyticsRepoImpl) GetProductLineInfo(ctx context.Context, accountID string, productLineIDs []string) ([]domain.ProductLineInfoRow, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_product_line_info")
	defer span.End()

	rows, err := r.queries.GetProductLineInfo(ctx, sqlc.GetProductLineInfoParams{
		ProductLineIds: productLineIDs,
		OwnerAccountID: sql.NullString{String: accountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	result := make([]domain.ProductLineInfoRow, len(rows))
	for i, row := range rows {
		result[i] = domain.ProductLineInfoRow{
			ID:   row.ID,
			Name: row.Name,
		}
	}

	return result, nil
}

// GetOrderQuantitiesByProductLines returns each product line's ordered quantity in a window, in the line's base unit.
func (r *analyticsRepoImpl) GetOrderQuantitiesByProductLines(ctx context.Context, params domain.GetOrderQuantitiesByProductLinesParams) ([]domain.OrderQuantityByProductLineRow, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_order_quantities_by_product_lines")
	defer span.End()

	if len(params.ProductLineIDs) == 0 {
		return []domain.OrderQuantityByProductLineRow{}, nil
	}

	demandIDs := make([]sql.NullString, len(params.ProductLineIDs))
	for i, id := range params.ProductLineIDs {
		demandIDs[i] = sql.NullString{String: id, Valid: true}
	}

	rows, err := r.queries.GetOrderQuantitiesByProductLines(ctx, sqlc.GetOrderQuantitiesByProductLinesParams{
		OwnerAccountID:       params.AccountID,
		DemandProductLineIds: demandIDs,
		StartDate:            sql.NullTime{Time: params.StartDate, Valid: true},
		EndDate:              sql.NullTime{Time: params.EndDate, Valid: true},
		ProductLineIds:       params.ProductLineIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]domain.OrderQuantityByProductLineRow, len(rows))
	for i, row := range rows {
		out[i] = domain.OrderQuantityByProductLineRow{
			ProductLineID:    row.ProductLineID,
			TotalQuantity:    decimalToFloat64(row.TotalQuantity),
			UnitAbbreviation: row.UnitAbbreviation,
			UnitType:         row.UnitType,
			BaseRatio:        decimalToFloat64(row.BaseRatio),
		}
	}
	return out, nil
}

func (r *analyticsRepoImpl) GetWeeksOfSalesOnHand(ctx context.Context, accountID string, itemIDs []string) ([]domain.ItemOnHandRow, *apierror.APIError) {
	ctx, span := analyticsRepoTracer.Start(ctx, "repository.analytics.get_weeks_of_sales_on_hand")
	defer span.End()

	if len(itemIDs) == 0 {
		return []domain.ItemOnHandRow{}, nil
	}
	rows, err := r.queries.GetWeeksOfSalesOnHand(ctx, sqlc.GetWeeksOfSalesOnHandParams{AccountID: accountID, ItemIds: itemIDs})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make([]domain.ItemOnHandRow, len(rows))
	for i, row := range rows {
		out[i] = domain.ItemOnHandRow{ItemID: row.ItemID, OnHand: decimalToFloat64(row.OnHand)}
	}
	return out, nil
}

// decimalToFloat64 converts a decimal string (from CAST AS DECIMAL) to the nearest float64, the rounding Decimal.toNumber() applied in the dashboard.
//
// It takes `any` because sqlc types the same column differently per query, so the sql.NullString case is not optional: whether a ratio column arrives bare or wrapped depends on whether its query happened to LEFT JOIN the table, and the unknown-type default silently returns 0. That cost a real bug — a nullable rate-denominator ratio read as zero, which skipped the unit conversion and left every pair-rated step at twice its true seconds per unit. Handling the wrapper here means a caller cannot lose a number by passing the type sqlc actually gave it.
func decimalToFloat64(v any) float64 {
	switch val := v.(type) {
	case sql.NullString:
		if !val.Valid {
			return 0
		}
		return decimalToFloat64(val.String)
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
		if err != nil {
			return 0
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

func interfaceToTimePtr(v any) *time.Time {
	switch val := v.(type) {
	case time.Time:
		return &val
	case *time.Time:
		return val
	default:
		return nil
	}
}
