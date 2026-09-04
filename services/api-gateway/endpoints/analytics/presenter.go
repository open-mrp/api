package analyticsep

import (
	"strconv"

	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/id"
	pb "github.com/open-mrp/api/shared/proto/core"
)

func AnalyzeSalesPresenter(resp *pb.AnalyzeSalesResponse) *apiresource.AnalyzeSalesResponse {
	if resp == nil {
		return &apiresource.AnalyzeSalesResponse{
			Object: constants.ObjectTypeList,
			Data:   []apiresource.SalesEntry{},
		}
	}

	entries := make([]apiresource.SalesEntry, len(resp.Entries))
	for i, e := range resp.Entries {
		entries[i] = salesEntryFromProto(e)
	}

	return &apiresource.AnalyzeSalesResponse{
		Object: constants.ObjectTypeList,
		Data:   entries,
	}
}

func salesEntryFromProto(e *pb.SalesEntryProto) apiresource.SalesEntry {
	if e == nil {
		return apiresource.SalesEntry{}
	}

	entry := apiresource.SalesEntry{
		ID:                  e.Id,
		IssuedAt:            grpcutil.TimestampToTimePtr(e.IssuedAt),
		CustomerPO:          ptrStringOrNil(e.CustomerPo),
		OrderNumber:         e.OrderNumber,
		OrderID:             e.OrderId,
		SalesRepID:          ptrStringOrNil(e.SalesRepId),
		SalesRepUsername:    ptrStringOrNil(e.SalesRepUsername),
		CustomerID:          e.CustomerId,
		CustomerName:        e.CustomerName,
		CustomerNumber:      e.CustomerNumber,
		CustomerTypeGroupID: ptrStringOrNil(e.CustomerTypeGroupId),
		CustomerGroupName:   ptrStringOrNil(e.CustomerGroupName),
		ParentCustomerID:    ptrStringOrNil(e.ParentCustomerId),
		CustomerCreatedAt:   grpcutil.TimestampToTime(e.CustomerCreatedAt),
		ProductLineID:       ptrStringOrNil(e.ProductLineId),
		ProductLine:         ptrStringOrNil(e.ProductLine),
		ProductTypeID:       e.ProductTypeId,
		ItemID:              e.ItemId,
		ProductSku:          e.ProductSku,
		ProductDescription:  ptrStringOrNil(e.ProductDescription),
		CategoryName:        e.CategoryName,
		QuantityInvoiced:    e.QuantityInvoiced,
		Unit:                e.Unit,
		UnitCost:            e.UnitCost,
		UnitPrice:           e.UnitPrice,
		UnitProfit:          e.UnitProfit,
		TotalInvoiced:       e.TotalInvoiced,
		TotalCost:           e.TotalCost,
		TotalProfit:         e.TotalProfit,
		ShipToCity:          ptrStringOrNil(e.ShipToCity),
		ShipToZipcode:       ptrStringOrNil(e.ShipToZipcode),
		ShipToState:         ptrStringOrNil(e.ShipToState),
		ShipToCountry:       ptrStringOrNil(e.ShipToCountry),
		OrderDiscountCode:   ptrStringOrNil(e.OrderDiscountCode),
		CompletedAt:         grpcutil.TimestampToTimePtr(e.CompletedAt),
		FirstShipAt:         grpcutil.TimestampToTimePtr(e.FirstShipAt),
		PromisedAt:          grpcutil.TimestampToTimePtr(e.PromisedAt),
		InvoiceID:           e.InvoiceId,
		InvoiceNumber:       e.InvoiceNumber,
		InvoicedAt:          grpcutil.TimestampToTime(e.InvoicedAt),
	}

	return entry
}

func AnalyzeOpenBatchesPresenter(resp *pb.AnalyzeOpenBatchesResponse) *apiresource.AnalyzeOpenBatchesResponse {
	if resp == nil {
		return &apiresource.AnalyzeOpenBatchesResponse{
			Object: constants.ObjectTypeList,
			Data:   []apiresource.OpenBatchSummary{},
		}
	}

	summaries := make([]apiresource.OpenBatchSummary, len(resp.Summaries))
	for i, s := range resp.Summaries {
		summary := apiresource.OpenBatchSummary{
			Object:         constants.ObjectTypeOpenBatchSummary,
			DepartmentName: s.DepartmentName,
			Count:          s.Count,
			Unit:           s.Unit,
		}
		if s.Item != nil {
			sku := s.Item.Sku
			summary.Item = apiresource.NewEntity(s.Item.Id, constants.ObjectTypeItem, &sku, nil)
		}
		if s.ScanningStation != nil {
			summary.ScanningStation = apiresource.NewEntity(s.ScanningStation.Id, constants.ObjectTypeScanningStation, nil, nil)
		}
		summaries[i] = summary
	}

	return &apiresource.AnalyzeOpenBatchesResponse{
		Object: constants.ObjectTypeList,
		Data:   summaries,
	}
}

func AnalyzeProductionCostsPresenter(resp *pb.AnalyzeProductionCostsResponse) *apiresource.AnalyzeProductionCostsResponse {
	if resp == nil {
		return &apiresource.AnalyzeProductionCostsResponse{
			Object: constants.ObjectTypeList,
			Data:   []apiresource.ProductionCostItem{},
		}
	}

	items := make([]apiresource.ProductionCostItem, len(resp.Items))
	for i, item := range resp.Items {
		var dept *apiresource.Entity
		if item.Department != nil {
			dept = apiresource.NewEntity(item.Department.Id, constants.ObjectTypeDepartment, &item.Department.Name, nil)
		}

		items[i] = apiresource.ProductionCostItem{
			Department:      dept,
			Category:        apiresource.NewEntity(item.Category.Id, constants.ObjectTypeItemCategory, &item.Category.Name, nil),
			TotalCosts:      costBreakdownFromProto(item.TotalCosts),
			ProductiveCosts: costBreakdownFromProto(item.ProductiveCosts),
			WasteCosts:      costBreakdownFromProto(item.WasteCosts),
			SecondsCosts:    costBreakdownFromProto(item.SecondsCosts),
		}
	}

	return &apiresource.AnalyzeProductionCostsResponse{
		Object: constants.ObjectTypeList,
		Data:   items,
	}
}

func costBreakdownFromProto(c *pb.CostBreakdown) apiresource.CostBreakdown {
	if c == nil {
		return apiresource.CostBreakdown{}
	}

	return apiresource.CostBreakdown{
		Total:     quantityFromProto(c.Total),
		Labor:     quantityFromProto(c.Labor),
		Materials: quantityFromProto(c.Materials),
		Overhead:  quantityFromProto(c.Overhead),
		Time:      quantityFromProto(c.Time),
		Quantity:  quantityFromProto(c.Quantity),
	}
}

func quantityFromProto(q *pb.BaseQuantity) *apiresource.Quantity {
	if q == nil {
		return nil
	}

	var unitAbbreviation, unitType string
	var unit *apiresource.Unit
	if q.Unit != nil {
		unitAbbreviation = q.Unit.Abbreviation
		unitType = q.Unit.Type
		unit = &apiresource.Unit{
			Object:       constants.ObjectTypeUnit,
			Name:         q.Unit.Name,
			Abbreviation: unitAbbreviation,
			Type:         constants.UnitType(unitType),
		}
	}

	valueStr := strconv.FormatFloat(q.Measure, 'f', -1, 64)
	qid, _ := id.GenID(id.QuantityIDPrefix, nil)
	return &apiresource.Quantity{
		ID:           qid,
		Object:       constants.ObjectTypeQuantity,
		Value:        apiresource.NormalizeQuantityValue(valueStr, unitType),
		DisplayValue: apiresource.FormatDisplayValue(valueStr, unitAbbreviation, unitType),
		Unit:         unit,
	}
}

func AnalyzeDeliveriesPresenter(resp *pb.AnalyzeDeliveriesResponse) *apiresource.AnalyzeDeliveriesResponse {
	if resp == nil {
		return &apiresource.AnalyzeDeliveriesResponse{
			Object: constants.ObjectTypeAnalyzeDeliveriesResponse,
		}
	}

	result := &apiresource.AnalyzeDeliveriesResponse{
		Object: constants.ObjectTypeAnalyzeDeliveriesResponse,
	}

	if resp.Statistics != nil {
		result.Statistics = apiresource.DeliveryStatistics{
			AverageTimeToFirstShipment:            resp.Statistics.AverageTimeToFirstShipment,
			AverageTimeToCompletion:               resp.Statistics.AverageTimeToCompletion,
			OnTimeDeliveryPercentage:              resp.Statistics.OnTimeDeliveryPercentage,
			OnTimeFirstShipmentPercentage:         resp.Statistics.OnTimeFirstShipmentPercentage,
			TotalOrders:                           int64(resp.Statistics.TotalOrders),
			OrdersWithFirstShipment:               int64(resp.Statistics.OrdersWithFirstShipment),
			OrdersWithCompletion:                  int64(resp.Statistics.OrdersWithCompletion),
			OrdersWithPromiseDate:                 int64(resp.Statistics.OrdersWithPromiseDate),
			OrdersPartiallyFulfilledInPromiseDate: int64(resp.Statistics.OrdersPartiallyFulfilledInPromiseDate),
			OrdersCompletedWithinPromiseDate:      int64(resp.Statistics.OrdersCompletedWithinPromiseDate),
		}
	}

	if resp.ChartData != nil {
		result.ChartData = apiresource.DeliveryChartData{
			OnTimeDelivery:           chartDataFromProto(resp.ChartData.OnTimeDelivery),
			AverageDeliveryTime:      chartDataFromProto(resp.ChartData.AverageDeliveryTime),
			AverageFirstShipmentTime: chartDataFromProto(resp.ChartData.AverageFirstShipmentTime),
		}
	}

	return result
}

func chartDataFromProto(c *pb.ChartDataPointProto) apiresource.ChartData {
	if c == nil {
		return apiresource.ChartData{}
	}

	coords := make([]apiresource.Coordinate, len(c.Data))
	for i, d := range c.Data {
		coords[i] = apiresource.Coordinate{
			X: d.X,
			Y: d.Y,
		}
	}

	return apiresource.ChartData{
		Name: c.Name,
		Type: c.Type,
		Data: coords,
	}
}

func AnalyzeManufacturingPresenter(resp *pb.AnalyzeManufacturingResponse) *apiresource.AnalyzeManufacturingResponse {
	if resp == nil {
		return &apiresource.AnalyzeManufacturingResponse{
			Object: constants.ObjectTypeAnalyzeManufacturingResponse,
		}
	}

	return &apiresource.AnalyzeManufacturingResponse{
		Object: constants.ObjectTypeAnalyzeManufacturingResponse,
		Value:  resp.Value,
	}
}

func AnalyzeManufacturingBatchPresenter(resp *pb.AnalyzeManufacturingBatchResponse) *apiresource.AnalyzeManufacturingBatchResponse {
	if resp == nil {
		return &apiresource.AnalyzeManufacturingBatchResponse{
			Object: constants.ObjectTypeAnalyzeManufacturingBatchResponse,
		}
	}

	return &apiresource.AnalyzeManufacturingBatchResponse{
		Object:     constants.ObjectTypeAnalyzeManufacturingBatchResponse,
		Current:    manufacturingMetricsFromProto(resp.Current),
		Comparison: manufacturingMetricsFromProto(resp.Comparison),
	}
}

func manufacturingMetricsFromProto(m *pb.ManufacturingMetricsProto) apiresource.ManufacturingMetrics {
	if m == nil {
		return apiresource.ManufacturingMetrics{}
	}

	return apiresource.ManufacturingMetrics{
		Production:      m.Production,
		CostsPerUnit:    m.CostsPerUnit,
		Margin:          m.Margin,
		Quality:         m.Quality,
		LaborEfficiency: m.LaborEfficiency,
	}
}

func AnalyzeOrdersPresenter(resp *pb.AnalyzeOrdersResponse) *apiresource.AnalyzeOrdersResponse {
	if resp == nil {
		return &apiresource.AnalyzeOrdersResponse{
			Object: constants.ObjectTypeList,
			Data:   []apiresource.OrderEntry{},
		}
	}

	entries := make([]apiresource.OrderEntry, len(resp.Entries))
	for i, e := range resp.Entries {
		entries[i] = orderEntryFromProto(e)
	}

	return &apiresource.AnalyzeOrdersResponse{
		Object: constants.ObjectTypeList,
		Data:   entries,
	}
}

func orderEntryFromProto(e *pb.OrderEntryProto) apiresource.OrderEntry {
	if e == nil {
		return apiresource.OrderEntry{}
	}

	return apiresource.OrderEntry{
		ID:                  e.Id,
		IssuedAt:            grpcutil.TimestampToTimePtr(e.IssuedAt),
		CustomerPO:          ptrStringOrNil(e.CustomerPo),
		OrderNumber:         e.OrderNumber,
		OrderID:             e.OrderId,
		SalesRepID:          ptrStringOrNil(e.SalesRepId),
		SalesRepUsername:    ptrStringOrNil(e.SalesRepUsername),
		CustomerID:          e.CustomerId,
		CustomerName:        e.CustomerName,
		CustomerNumber:      e.CustomerNumber,
		CustomerTypeGroupID: ptrStringOrNil(e.CustomerTypeGroupId),
		CustomerGroupName:   ptrStringOrNil(e.CustomerGroupName),
		ParentCustomerID:    ptrStringOrNil(e.ParentCustomerId),
		CustomerCreatedAt:   grpcutil.TimestampToTime(e.CustomerCreatedAt),
		ProductLineID:       ptrStringOrNil(e.ProductLineId),
		ProductLine:         ptrStringOrNil(e.ProductLine),
		ProductTypeID:       e.ProductTypeId,
		ItemID:              e.ItemId,
		ProductSku:          e.ProductSku,
		ProductDescription:  ptrStringOrNil(e.ProductDescription),
		CategoryName:        e.CategoryName,
		QuantityInvoiced:    e.QuantityInvoiced,
		Unit:                e.Unit,
		UnitCost:            e.UnitCost,
		UnitPrice:           e.UnitPrice,
		UnitProfit:          e.UnitProfit,
		TotalInvoiced:       e.TotalInvoiced,
		TotalCost:           e.TotalCost,
		TotalProfit:         e.TotalProfit,
		ShipToCity:          ptrStringOrNil(e.ShipToCity),
		ShipToZipcode:       ptrStringOrNil(e.ShipToZipcode),
		ShipToState:         ptrStringOrNil(e.ShipToState),
		ShipToCountry:       ptrStringOrNil(e.ShipToCountry),
		OrderDiscountCode:   ptrStringOrNil(e.OrderDiscountCode),
		CompletedAt:         grpcutil.TimestampToTimePtr(e.CompletedAt),
		FirstShipAt:         grpcutil.TimestampToTimePtr(e.FirstShipAt),
		PromisedAt:          grpcutil.TimestampToTimePtr(e.PromisedAt),
		QuantityOrdered:     e.QuantityOrdered,
		QuantityBackOrdered: e.QuantityBackOrdered,
		TotalOrdered:        e.TotalOrdered,
		TotalBackOrdered:    e.TotalBackOrdered,
	}
}

func AnalyzeQuarterlyOrdersPresenter(resp *pb.AnalyzeQuarterlyOrdersResponse) *apiresource.AnalyzeQuarterlyOrdersResponse {
	if resp == nil {
		return &apiresource.AnalyzeQuarterlyOrdersResponse{
			Object: constants.ObjectTypeAnalyzeQuarterlyOrdersResponse,
			Data:   map[string]apiresource.QuarterlySalesData{},
		}
	}

	data := make(map[string]apiresource.QuarterlySalesData, len(resp.Data))
	for year, qd := range resp.Data {
		data[year] = apiresource.QuarterlySalesData{
			Q1:    qd.Q1,
			Q2:    qd.Q2,
			Q3:    qd.Q3,
			Q4:    qd.Q4,
			Total: qd.Total,
		}
	}

	return &apiresource.AnalyzeQuarterlyOrdersResponse{
		Object: constants.ObjectTypeAnalyzeQuarterlyOrdersResponse,
		Data:   data,
	}
}

func AnalyzeMaterialsPresenter(resp *pb.AnalyzeMaterialsResponse) *apiresource.AnalyzeMaterialsResponse {
	if resp == nil {
		return &apiresource.AnalyzeMaterialsResponse{
			Object: constants.ObjectTypeList,
			Data:   []apiresource.MaterialAnalyticsEntry{},
		}
	}

	entries := make([]apiresource.MaterialAnalyticsEntry, len(resp.Entries))
	for i, e := range resp.Entries {
		entry := apiresource.MaterialAnalyticsEntry{
			ID:                  e.Id,
			ItemID:              e.ItemId,
			Sku:                 e.Sku,
			Description:         ptrStringOrNil(e.Description),
			QuantityInInventory: quantityFromProto(e.QuantityInInventory),
			QuantityInDemand:    quantityFromProto(e.QuantityInDemand),
			SupplierNames:       e.SupplierNames,
			SupplierPartNumbers: e.SupplierPartNumbers,
		}

		if e.OrderPoint != nil {
			entry.OrderPoint = quantityFromProto(e.OrderPoint)
		}
		if e.LeadTime != nil {
			entry.LeadTime = quantityFromProto(e.LeadTime)
		}
		if e.UnitGroup != nil {
			entry.UnitGroup = unitGroupFromProto(e.UnitGroup)
		}

		entries[i] = entry
	}

	return &apiresource.AnalyzeMaterialsResponse{
		Object: constants.ObjectTypeList,
		Data:   entries,
	}
}

func unitGroupFromProto(ug *pb.AnalyticsUnitGroup) apiresource.AnalyticsUnitGroup {
	if ug == nil {
		return apiresource.AnalyticsUnitGroup{}
	}

	units := make([]apiresource.AnalyticsUnitGroupUnit, len(ug.Units))
	for i, u := range ug.Units {
		units[i] = apiresource.AnalyticsUnitGroupUnit{
			ID:               u.Id,
			Name:             u.Name,
			Abbreviation:     u.Abbreviation,
			ConversionFactor: u.ConversionFactor,
			IsBaseUnit:       u.IsBaseUnit,
		}
	}

	return apiresource.AnalyticsUnitGroup{
		ID:    ug.Id,
		Name:  ug.Name,
		Units: units,
	}
}

func AnalyzeInventoryReceiptsPresenter(resp *pb.AnalyzeInventoryReceiptsResponse) *apiresource.AnalyzeInventoryReceiptsResponse {
	if resp == nil {
		return &apiresource.AnalyzeInventoryReceiptsResponse{
			Object: constants.ObjectTypeList,
			Data:   []apiresource.InventoryReceiptSummaryEntry{},
		}
	}

	entries := make([]apiresource.InventoryReceiptSummaryEntry, len(resp.Entries))
	for i, e := range resp.Entries {
		entry := apiresource.InventoryReceiptSummaryEntry{
			Item: apiresource.AnalyticsItem{
				ID:          e.Item.Id,
				Object:      constants.ObjectTypeItem,
				Sku:         e.Item.Sku,
				Description: ptrStringOrNil(e.Item.Description),
			},
			OwnerAccount:      apiresource.NewEntity(e.OwnerAccount.Id, constants.ObjectTypeAccount, &e.OwnerAccount.Name, nil),
			HolderAccount:     apiresource.NewEntity(e.HolderAccount.Id, constants.ObjectTypeAccount, &e.HolderAccount.Name, nil),
			RemainingQuantity: quantityFromProto(e.RemainingQuantity),
			WeightedAverageUnitCost: apiresource.AnalyticsRate{
				Numerator:   quantityFromProto(e.WeightedAverageUnitCost.Numerator),
				Denominator: quantityFromProto(e.WeightedAverageUnitCost.Denominator),
			},
			OldestReceiptAt: grpcutil.TimestampToTimePtr(e.OldestReceiptAt),
			NewestReceiptAt: grpcutil.TimestampToTimePtr(e.NewestReceiptAt),
		}

		if e.Location != nil {
			entry.Location = apiresource.NewEntity(e.Location.Id, constants.ObjectTypeLocation, &e.Location.Name, nil)
		}
		if e.Lot != nil {
			entry.Lot = &apiresource.AnalyticsLot{
				ID:     e.Lot.Id,
				Object: constants.ObjectTypeLot,
				Number: e.Lot.Number,
			}
		}
		if e.InventoryValue != nil {
			entry.InventoryValue = quantityFromProto(e.InventoryValue)
		}

		entries[i] = entry
	}

	return &apiresource.AnalyzeInventoryReceiptsResponse{
		Object: constants.ObjectTypeList,
		Data:   entries,
	}
}

func AnalyzeNewCustomersPresenter(resp *pb.AnalyzeNewCustomersResponse) *apiresource.AnalyzeNewCustomersResponse {
	if resp == nil {
		return &apiresource.AnalyzeNewCustomersResponse{
			Object: constants.ObjectTypeAnalyzeNewCustomersResponse,
		}
	}

	var newCustomers apiresource.NewCustomersData
	if resp.NewCustomers != nil {
		coords := make([]apiresource.DateTimeCoordinate, len(resp.NewCustomers.Data))
		for i, d := range resp.NewCustomers.Data {
			coords[i] = apiresource.DateTimeCoordinate{
				X: grpcutil.TimestampToTime(d.X),
				Y: d.Y,
			}
		}
		newCustomers = apiresource.NewCustomersData{
			Label: resp.NewCustomers.Label,
			Data:  coords,
		}
	}

	return &apiresource.AnalyzeNewCustomersResponse{
		Object:       constants.ObjectTypeAnalyzeNewCustomersResponse,
		NewCustomers: newCustomers,
	}
}

func AnalyzeDemandForecastPresenter(resp *pb.AnalyzeDemandForecastResponse) *apiresource.AnalyzeDemandForecastResponse {
	if resp == nil {
		return &apiresource.AnalyzeDemandForecastResponse{
			Object: constants.ObjectTypeAnalyzeDemandForecastResponse,
			Data:   apiresource.NewList([]apiresource.DemandForecastRow{}, apiresource.PageInfo{}),
		}
	}

	rows := make([]apiresource.DemandForecastRow, len(resp.Rows))
	for i, r := range resp.Rows {
		var productLine *apiresource.Entity
		if r.ProductLineId != "" {
			productLine = apiresource.NewEntity(r.ProductLineId, constants.ObjectTypeProductLine, nil, nil)
		}

		rows[i] = apiresource.DemandForecastRow{
			Item:                apiresource.NewEntity(r.ItemId, constants.ObjectTypeItem, nil, nil),
			ProductLine:         productLine,
			ProductSku:          r.ProductSku,
			ProductDescription:  ptrStringOrNil(r.ProductDescription),
			Unit:                r.Unit,
			Currency:            r.Currency,
			History:             demandForecastPointsFromProto(r.History),
			Forecast:            demandForecastForecastPointsFromProto(r.Forecast),
			RevenueHistory:      revenueForecastPointsFromProto(r.RevenueHistory),
			RevenueForecast:     demandForecastForecastPointsFromProto(r.RevenueForecast),
			SalesHistory:        revenueForecastPointsFromProto(r.SalesHistory),
			SalesForecast:       demandForecastForecastPointsFromProto(r.SalesForecast),
			CurrentMonthDemand:  r.CurrentMonthDemand,
			CurrentMonthRevenue: r.CurrentMonthRevenue,
			CurrentMonthSales:   r.CurrentMonthSales,
		}
	}

	return &apiresource.AnalyzeDemandForecastResponse{
		Object:               constants.ObjectTypeAnalyzeDemandForecastResponse,
		Data:                 apiresource.NewList(rows, apiresource.PageInfo{}),
		CurrentMonthFraction: resp.CurrentMonthFraction,
	}
}

func demandForecastPointsFromProto(points []*pb.DemandHistoryPointProto) []apiresource.DemandForecastPoint {
	result := make([]apiresource.DemandForecastPoint, len(points))
	for i, p := range points {
		result[i] = apiresource.DemandForecastPoint{
			Date:   grpcutil.TimestampToTime(p.Date),
			Demand: p.Demand,
		}
	}
	return result
}

func demandForecastForecastPointsFromProto(points []*pb.DemandForecastPointProto) []apiresource.DemandForecastForecastPoint {
	result := make([]apiresource.DemandForecastForecastPoint, len(points))
	for i, p := range points {
		result[i] = apiresource.DemandForecastForecastPoint{
			Date:       grpcutil.TimestampToTime(p.Date),
			Forecast:   p.Forecast,
			LowerBound: p.LowerBound,
			UpperBound: p.UpperBound,
		}
	}
	return result
}

func revenueForecastPointsFromProto(points []*pb.RevenueHistoryPointProto) []apiresource.RevenueForecastPoint {
	result := make([]apiresource.RevenueForecastPoint, len(points))
	for i, p := range points {
		result[i] = apiresource.RevenueForecastPoint{
			Date:    grpcutil.TimestampToTime(p.Date),
			Revenue: p.Revenue,
		}
	}
	return result
}

func AnalyzeOeeTrendPresenter(resp *pb.AnalyzeOeeTrendResponse) *apiresource.AnalyzeOeeTrendResponse {
	if resp == nil {
		return &apiresource.AnalyzeOeeTrendResponse{
			Object:  constants.ObjectTypeAnalyzeOeeTrendResponse,
			Periods: apiresource.NewList([]apiresource.OeeTrendPeriod{}, apiresource.PageInfo{}),
		}
	}

	periods := make([]apiresource.OeeTrendPeriod, len(resp.Periods))
	for i, p := range resp.Periods {
		periods[i] = apiresource.OeeTrendPeriod{
			StartsAt:                grpcutil.TimestampToTime(p.StartsAt),
			EndsAt:                  grpcutil.TimestampToTime(p.EndsAt),
			GoodUnits:               p.GoodUnits,
			WasteUnits:              p.WasteUnits,
			SecondsUnits:            p.SecondsUnits,
			StandardSecondsEarned:   p.StandardSecondsEarned,
			ScheduledSeconds:        p.ScheduledSeconds,
			OperatingTimeSeconds:    p.OperatingTimeSeconds,
			RunTimeSeconds:          p.RunTimeSeconds,
			OverrunSeconds:          p.OverrunSeconds,
			AvailabilityLossSeconds: p.AvailabilityLossSeconds,
			NotScheduledSeconds:     p.NotScheduledSeconds,
			AvailabilityPct:         p.AvailabilityPct,
			PerformancePct:          p.PerformancePct,
			QualityPct:              p.QualityPct,
			OeePct:                  p.OeePct,
			MeasurementStatus:       oeeMeasurementStatus(p.HasDowntimeData),
			DowntimeEventCount:      p.DowntimeEventCount,
		}
	}

	return &apiresource.AnalyzeOeeTrendResponse{
		Object:  constants.ObjectTypeAnalyzeOeeTrendResponse,
		Periods: apiresource.NewList(periods, apiresource.PageInfo{}),
	}
}

func AnalyzeOeePresenter(resp *pb.AnalyzeOeeResponse) *apiresource.AnalyzeOeeResponse {
	if resp == nil {
		return &apiresource.AnalyzeOeeResponse{
			Object:      constants.ObjectTypeAnalyzeOeeResponse,
			Departments: apiresource.NewList([]apiresource.OeeDepartment{}, apiresource.PageInfo{}),
		}
	}

	depts := make([]apiresource.OeeDepartment, len(resp.Departments))
	for i, d := range resp.Departments {
		breakdown := make([]apiresource.OeeDowntimeReason, len(d.DowntimeBreakdown))
		for j, r := range d.DowntimeBreakdown {
			breakdown[j] = apiresource.OeeDowntimeReason{
				Reason:          constants.MachineDowntimeReasonCode(r.ReasonCode),
				OeeBucket:       constants.OeeBucket(r.OeeBucket),
				DowntimeSeconds: r.DowntimeSeconds,
				EventCount:      r.EventCount,
			}
		}

		depts[i] = apiresource.OeeDepartment{
			Department:              apiresource.NewEntity(d.DepartmentId, constants.ObjectTypeDepartment, &d.DepartmentName, nil),
			GoodUnits:               d.GoodUnits,
			WasteUnits:              d.WasteUnits,
			SecondsUnits:            d.SecondsUnits,
			StandardSecondsEarned:   d.StandardSecondsEarned,
			EstimatedRuntimeHours:   d.EstimatedRuntimeHours,
			AvailabilityLossSeconds: d.AvailabilityLossSeconds,
			PerformanceLossSeconds:  d.PerformanceLossSeconds,
			QualityLossSeconds:      d.QualityLossSeconds,
			NotScheduledSeconds:     d.NotScheduledSeconds,
			ChangeoverSeconds:       d.ChangeoverSeconds,
			DowntimeEventCount:      d.DowntimeEventCount,
			DowntimeBreakdown:       apiresource.NewList(breakdown, apiresource.PageInfo{}),
			ScheduledSeconds:        d.ScheduledSeconds,
			OperatingTimeSeconds:    d.OperatingTimeSeconds,
			RunTimeSeconds:          d.RunTimeSeconds,
			OverrunSeconds:          d.OverrunSeconds,
			AvailabilityPct:         d.AvailabilityPct,
			PerformancePct:          d.PerformancePct,
			QualityPct:              d.QualityPct,
			OeePct:                  d.OeePct,
			MeasurementStatus:       oeeMeasurementStatus(d.HasDowntimeData),
			Anomalies:               oeeAnomalies(d.HasPerformanceAnomaly),
		}
	}

	return &apiresource.AnalyzeOeeResponse{
		Object:      constants.ObjectTypeAnalyzeOeeResponse,
		Departments: apiresource.NewList(depts, apiresource.PageInfo{}),
	}
}

func AnalyzeWeeksOfSalesPresenter(resp *pb.AnalyzeWeeksOfSalesResponse) *apiresource.AnalyzeWeeksOfSalesResponse {
	if resp == nil {
		return &apiresource.AnalyzeWeeksOfSalesResponse{
			Object: constants.ObjectTypeAnalyzeWeeksOfSalesResponse,
			Data:   []apiresource.WeeksOfSalesItem{},
			Count:  0,
		}
	}

	items := make([]apiresource.WeeksOfSalesItem, len(resp.Items))
	for i, item := range resp.Items {
		onHandValueStr := strconv.FormatFloat(item.QuantityOnHand, 'f', -1, 64)
		avgSalesValueStr := strconv.FormatFloat(item.AverageSalesQuantity, 'f', -1, 64)

		onHandID, _ := id.GenID(id.QuantityIDPrefix, nil)
		avgSalesID, _ := id.GenID(id.QuantityIDPrefix, nil)

		items[i] = apiresource.WeeksOfSalesItem{
			ProductLine: apiresource.NewEntity(item.ProductLineId, constants.ObjectTypeProductLine, &item.ProductLineName, nil),
			QuantityOnHand: &apiresource.Quantity{
				ID:     onHandID,
				Object: constants.ObjectTypeQuantity,
				Value:  apiresource.NormalizeQuantityValue(onHandValueStr, item.QuantityOnHandUnitType),
				DisplayValue: apiresource.FormatDisplayValue(
					onHandValueStr,
					item.QuantityOnHandUnitAbbreviation,
					item.QuantityOnHandUnitType,
				),
				Unit: &apiresource.Unit{
					Object:       constants.ObjectTypeUnit,
					Name:         item.QuantityOnHandUnitAbbreviation,
					Abbreviation: item.QuantityOnHandUnitAbbreviation,
					Type:         constants.UnitType(item.QuantityOnHandUnitType),
				},
			},
			AverageSalesQuantity: &apiresource.Quantity{
				ID:     avgSalesID,
				Object: constants.ObjectTypeQuantity,
				Value:  apiresource.NormalizeQuantityValue(avgSalesValueStr, item.AverageSalesQuantityUnitType),
				DisplayValue: apiresource.FormatDisplayValue(
					avgSalesValueStr,
					item.AverageSalesQuantityUnitAbbreviation,
					item.AverageSalesQuantityUnitType,
				),
				Unit: &apiresource.Unit{
					Object:       constants.ObjectTypeUnit,
					Name:         item.AverageSalesQuantityUnitAbbreviation,
					Abbreviation: item.AverageSalesQuantityUnitAbbreviation,
					Type:         constants.UnitType(item.AverageSalesQuantityUnitType),
				},
			},
			WeeksOfSales: item.WeeksOfSales,
		}
	}

	return &apiresource.AnalyzeWeeksOfSalesResponse{
		Object: constants.ObjectTypeAnalyzeWeeksOfSalesResponse,
		Data:   items,
		Count:  resp.Count,
	}
}

func ptrStringOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// oeeMeasurementStatus labels an estimate as an estimate. A department with no logged downtime computes as perfectly available, so presenting that as a measurement would make OEE jump for the worst possible reason.
func oeeMeasurementStatus(hasDowntimeData bool) constants.OeeMeasurementStatus {
	if hasDowntimeData {
		return constants.OeeMeasurementStatusMeasured
	}
	return constants.OeeMeasurementStatusEstimated
}

func oeeAnomalies(performanceAboveCapacity bool) []constants.OeeAnomaly {
	out := []constants.OeeAnomaly{}
	if performanceAboveCapacity {
		out = append(out, constants.OeeAnomalyPerformanceAboveCapacity)
	}
	return out
}
