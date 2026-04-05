package analyticsep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
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
			summary.Item = &apiresource.Item{
				ID:  s.Item.Id,
				SKU: s.Item.Sku,
			}
		}
		if s.ScanningStation != nil {
			summary.ScanningStation = &apiresource.ScanningStation{
				ID: s.ScanningStation.Id,
			}
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
		var dept *apiresource.BasicInfo
		if item.Department != nil {
			dept = &apiresource.BasicInfo{
				ID:   item.Department.Id,
				Name: item.Department.Name,
			}
		}

		items[i] = apiresource.ProductionCostItem{
			Department: dept,
			Category: apiresource.BasicInfo{
				ID:   item.Category.Id,
				Name: item.Category.Name,
			},
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
		Total:     baseQuantityFromProto(c.Total),
		Labor:     baseQuantityFromProto(c.Labor),
		Materials: baseQuantityFromProto(c.Materials),
		Overhead:  baseQuantityFromProto(c.Overhead),
		Time:      baseQuantityFromProto(c.Time),
		Quantity:  baseQuantityFromProto(c.Quantity),
	}
}

func baseQuantityFromProto(q *pb.BaseQuantity) apiresource.BaseQuantity {
	if q == nil {
		return apiresource.BaseQuantity{}
	}

	var unit apiresource.BaseQuantityUnit
	if q.Unit != nil {
		unit = apiresource.BaseQuantityUnit{
			Name:         q.Unit.Name,
			Abbreviation: q.Unit.Abbreviation,
			Type:         q.Unit.Type,
		}
	}

	return apiresource.BaseQuantity{
		Measure: q.Measure,
		Unit:    unit,
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
			QuantityInInventory: baseQuantityFromProto(e.QuantityInInventory),
			QuantityInDemand:    baseQuantityFromProto(e.QuantityInDemand),
			SupplierNames:       e.SupplierNames,
			SupplierPartNumbers: e.SupplierPartNumbers,
		}

		if e.OrderPoint != nil {
			op := baseQuantityFromProto(e.OrderPoint)
			entry.OrderPoint = &op
		}
		if e.LeadTime != nil {
			lt := baseQuantityFromProto(e.LeadTime)
			entry.LeadTime = &lt
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
				Sku:         e.Item.Sku,
				Description: ptrStringOrNil(e.Item.Description),
			},
			OwnerAccount: apiresource.BasicInfo{
				ID:   e.OwnerAccount.Id,
				Name: e.OwnerAccount.Name,
			},
			HolderAccount: apiresource.BasicInfo{
				ID:   e.HolderAccount.Id,
				Name: e.HolderAccount.Name,
			},
			RemainingQuantity: baseQuantityFromProto(e.RemainingQuantity),
			WeightedAverageUnitCost: apiresource.AnalyticsRate{
				Numerator:   baseQuantityFromProto(e.WeightedAverageUnitCost.Numerator),
				Denominator: baseQuantityFromProto(e.WeightedAverageUnitCost.Denominator),
			},
			OldestReceiptAt: grpcutil.TimestampToTimePtr(e.OldestReceiptAt),
			NewestReceiptAt: grpcutil.TimestampToTimePtr(e.NewestReceiptAt),
		}

		if e.Location != nil {
			entry.Location = &apiresource.BasicInfo{
				ID:   e.Location.Id,
				Name: e.Location.Name,
			}
		}
		if e.Lot != nil {
			entry.Lot = &apiresource.AnalyticsLot{
				ID:     e.Lot.Id,
				Number: e.Lot.Number,
			}
		}
		if e.InventoryValue != nil {
			iv := baseQuantityFromProto(e.InventoryValue)
			entry.InventoryValue = &iv
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
			Object: constants.ObjectTypeList,
			Data:   []apiresource.DemandForecastRow{},
		}
	}

	rows := make([]apiresource.DemandForecastRow, len(resp.Rows))
	for i, r := range resp.Rows {
		rows[i] = apiresource.DemandForecastRow{
			ItemID:              r.ItemId,
			ProductLineID:       ptrStringOrNil(r.ProductLineId),
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
		Object:               constants.ObjectTypeList,
		Data:                 rows,
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

func AnalyzeOeePresenter(resp *pb.AnalyzeOeeResponse) *apiresource.AnalyzeOeeResponse {
	if resp == nil {
		return &apiresource.AnalyzeOeeResponse{
			Object:      constants.ObjectTypeAnalyzeOeeResponse,
			Departments: []apiresource.OeeDepartment{},
		}
	}

	depts := make([]apiresource.OeeDepartment, len(resp.Departments))
	for i, d := range resp.Departments {
		depts[i] = apiresource.OeeDepartment{
			DepartmentID:          d.DepartmentId,
			DepartmentName:        d.DepartmentName,
			GoodUnits:             d.GoodUnits,
			WasteUnits:            d.WasteUnits,
			SecondsUnits:          d.SecondsUnits,
			EstimatedRuntimeHours: d.EstimatedRuntimeHours,
		}
	}

	return &apiresource.AnalyzeOeeResponse{
		Object:      constants.ObjectTypeAnalyzeOeeResponse,
		Departments: depts,
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
		items[i] = apiresource.WeeksOfSalesItem{
			ProductLine: apiresource.BasicInfo{
				ID:   item.ProductLineId,
				Name: item.ProductLineName,
			},
			QuantityOnHand: apiresource.BaseQuantity{
				Measure: item.QuantityOnHand,
				Unit: apiresource.BaseQuantityUnit{
					Name:         item.QuantityOnHandUnitAbbreviation,
					Abbreviation: item.QuantityOnHandUnitAbbreviation,
					Type:         item.QuantityOnHandUnitType,
				},
			},
			AverageSalesQuantity: apiresource.BaseQuantity{
				Measure: item.AverageSalesQuantity,
				Unit: apiresource.BaseQuantityUnit{
					Name:         item.AverageSalesQuantityUnitAbbreviation,
					Abbreviation: item.AverageSalesQuantityUnitAbbreviation,
					Type:         item.AverageSalesQuantityUnitType,
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
