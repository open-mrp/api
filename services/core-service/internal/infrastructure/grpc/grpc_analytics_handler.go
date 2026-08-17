package grpc

import (
	"context"
	"fmt"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/scheduling"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/safeconv"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// --- Helper functions for analytics proto conversions ---

// derefString safely dereferences a *string, returning empty string for nil.
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// chartDataPointsToProto converts a slice of domain ChartDataPoints into a single ChartDataPointProto with CoordinateProto data points.
func chartDataPointsToProto(name string, chartType string, points []domain.ChartDataPoint) *pb.ChartDataPointProto {
	coords := make([]*pb.CoordinateProto, len(points))
	for i, p := range points {
		coords[i] = &pb.CoordinateProto{
			X: p.X,
			Y: p.Y,
		}
	}
	return &pb.ChartDataPointProto{
		Name: name,
		Type: chartType,
		Data: coords,
	}
}

func manufacturingMetricsToProto(m domain.ManufacturingMetrics) *pb.ManufacturingMetricsProto {
	return &pb.ManufacturingMetricsProto{
		Production:      m.Production,
		CostsPerUnit:    m.CostsPerUnit,
		Margin:          m.Margin,
		Quality:         m.Quality,
		LaborEfficiency: m.LaborEfficiency,
	}
}

func demandHistoryPointsToProto(points []domain.DemandHistoryPoint) []*pb.DemandHistoryPointProto {
	result := make([]*pb.DemandHistoryPointProto, len(points))
	for i, p := range points {
		result[i] = &pb.DemandHistoryPointProto{
			Date:   timestamppb.New(p.Date),
			Demand: p.Demand,
		}
	}
	return result
}

func demandForecastPointsToProto(points []domain.DemandForecastPoint) []*pb.DemandForecastPointProto {
	result := make([]*pb.DemandForecastPointProto, len(points))
	for i, p := range points {
		result[i] = &pb.DemandForecastPointProto{
			Date:       timestamppb.New(p.Date),
			Forecast:   p.Forecast,
			LowerBound: p.LowerBound,
			UpperBound: p.UpperBound,
		}
	}
	return result
}

func revenueHistoryPointsToProto(points []domain.RevenueHistoryPoint) []*pb.RevenueHistoryPointProto {
	result := make([]*pb.RevenueHistoryPointProto, len(points))
	for i, p := range points {
		result[i] = &pb.RevenueHistoryPointProto{
			Date:    timestamppb.New(p.Date),
			Revenue: p.Revenue,
		}
	}
	return result
}

func revenueForecastPointsToProto(points []domain.RevenueForecastPoint) []*pb.DemandForecastPointProto {
	result := make([]*pb.DemandForecastPointProto, len(points))
	for i, p := range points {
		result[i] = &pb.DemandForecastPointProto{
			Date:       timestamppb.New(p.Date),
			Forecast:   p.Forecast,
			LowerBound: p.LowerBound,
			UpperBound: p.UpperBound,
		}
	}
	return result
}

// --- Analytics gRPC Handlers ---

func (h *gRPCHandler) AnalyzeSales(ctx context.Context, req *pb.AnalyzeSalesRequest) (*pb.AnalyzeSalesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.AnalyzeSalesParams{
		ProductLineIDs:   req.ProductLineIds,
		CustomerIDs:      req.CustomerIds,
		SalesRepIDs:      req.SalesRepIds,
		CustomerGroupIDs: req.CustomerGroupIds,
		Query:            req.Query,
		IsSalesRep:       req.IsSalesRep,
	}

	if req.StartDate != nil {
		params.StartDate = req.StartDate.AsTime()
	}
	if req.EndDate != nil {
		params.EndDate = req.EndDate.AsTime()
	}

	entries, apiErr := h.analyticsSvc.AnalyzeSales(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbEntries := make([]*pb.SalesEntryProto, len(entries))
	for i, e := range entries {
		entry := &pb.SalesEntryProto{
			Id:                  e.ID,
			InvoiceId:           e.InvoiceID,
			InvoiceNumber:       e.InvoiceNumber,
			InvoicedAt:          timestamppb.New(e.InvoiceDate),
			CustomerPo:          derefString(e.CustomerPO),
			OrderId:             e.SalesOrderID,
			OrderNumber:         e.SalesOrderNumber,
			CustomerId:          e.CustomerID,
			CustomerName:        e.CustomerName,
			CustomerNumber:      e.CustomerNumber,
			CustomerCreatedAt:   timestamppb.New(e.CustomerCreatedAt),
			CustomerTypeGroupId: derefString(e.CustomerTypeGroupID),
			CustomerGroupName:   derefString(e.CustomerGroupName),
			ParentCustomerId:    derefString(e.ParentCustomerID),
			ProductTypeId:       e.ProductTypeCode,
			ItemId:              e.ItemID,
			ProductSku:          e.ProductSku,
			ProductDescription:  derefString(e.ProductDescription),
			CategoryName:        e.CategoryName,
			ProductLine:         derefString(e.ProductLine),
			ProductLineId:       derefString(e.ProductLineID),
			Unit:                e.Unit,
			QuantityInvoiced:    e.QuantityInvoiced,
			UnitPrice:           e.UnitPrice,
			UnitCost:            e.UnitCost,
			UnitProfit:          e.UnitProfit,
			TotalInvoiced:       e.TotalInvoiced,
			TotalCost:           e.TotalCost,
			TotalProfit:         e.TotalProfit,
			SalesRepUsername:    derefString(e.SalesRepUsername),
			SalesRepId:          derefString(e.SalesRepID),
			ShipToState:         derefString(e.ShipToState),
			ShipToCity:          derefString(e.ShipToCity),
			ShipToZipcode:       derefString(e.ShipToPostalCode),
			ShipToCountry:       derefString(e.ShipToCountry),
			OrderDiscountCode:   derefString(e.OrderDiscountCode),
		}
		if e.IssuedAt != nil {
			entry.IssuedAt = timestamppb.New(*e.IssuedAt)
		}
		if e.CompletedAt != nil {
			entry.CompletedAt = timestamppb.New(*e.CompletedAt)
		}
		if e.FirstShipAt != nil {
			entry.FirstShipAt = timestamppb.New(*e.FirstShipAt)
		}
		if e.PromisedAt != nil {
			entry.PromisedAt = timestamppb.New(*e.PromisedAt)
		}
		pbEntries[i] = entry
	}

	return &pb.AnalyzeSalesResponse{
		Entries: pbEntries,
	}, nil
}

func (h *gRPCHandler) AnalyzeProductionCosts(ctx context.Context, req *pb.AnalyzeProductionCostsRequest) (*pb.AnalyzeProductionCostsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.AnalyzeProductionCostsParams{
		ItemIDs:        req.ItemIds,
		ProductLineIDs: req.ProductLineIds,
		DepartmentIDs:  req.DepartmentIds,
		CategoryIDs:    req.CategoryIds,
	}

	if req.StartDate != nil {
		t := req.StartDate.AsTime()
		params.StartDate = &t
	}
	if req.EndDate != nil {
		t := req.EndDate.AsTime()
		params.EndDate = &t
	}

	entries, apiErr := h.analyticsSvc.AnalyzeProductionCosts(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbEntries := make([]*pb.ProductionCostEntryProto, len(entries))
	for i, e := range entries {
		pbEntries[i] = &pb.ProductionCostEntryProto{
			TotalCosts: &pb.CostBreakdown{
				Total: &pb.BaseQuantity{
					Measure: e.TotalCost,
					Unit: &pb.BaseQuantityUnitProto{
						Abbreviation: e.Unit,
					},
				},
			},
		}
	}

	return &pb.AnalyzeProductionCostsResponse{
		Items: pbEntries,
	}, nil
}

func (h *gRPCHandler) AnalyzeDeliveries(ctx context.Context, req *pb.AnalyzeDeliveriesRequest) (*pb.AnalyzeDeliveriesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.AnalyzeDeliveriesParams{
		ProductLineIDs:         req.ProductLineIds,
		CustomerIDs:            req.CustomerIds,
		CustomerGroupIDs:       req.CustomerGroupIds,
		SalesRepIDs:            req.SalesRepIds,
		TargetDeliveryTimeDays: req.TargetDeliveryTimeDays,
		OverridePromisedDates:  req.OverridePromisedDates,
	}

	if req.StartDate != nil {
		params.StartDate = req.StartDate.AsTime()
	}
	if req.EndDate != nil {
		params.EndDate = req.EndDate.AsTime()
	}

	result, apiErr := h.analyticsSvc.AnalyzeDeliveries(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.AnalyzeDeliveriesResponse{
		Statistics: &pb.DeliveryStatisticsProto{
			AverageTimeToFirstShipment:            result.Statistics.AverageTimeToFirstShipment,
			AverageTimeToCompletion:               result.Statistics.AverageTimeToCompletion,
			OnTimeDeliveryPercentage:              result.Statistics.OnTimeDeliveryPercentage,
			OnTimeFirstShipmentPercentage:         result.Statistics.OnTimeFirstShipmentPercentage,
			TotalOrders:                           result.Statistics.TotalOrders,
			OrdersWithFirstShipment:               result.Statistics.OrdersWithFirstShipment,
			OrdersWithCompletion:                  result.Statistics.OrdersWithCompletion,
			OrdersWithPromiseDate:                 result.Statistics.OrdersWithPromiseDate,
			OrdersPartiallyFulfilledInPromiseDate: result.Statistics.OrdersPartiallyFulfilledInPromiseDate,
			OrdersCompletedWithinPromiseDate:      result.Statistics.OrdersCompletedWithinPromiseDate,
		},
		ChartData: &pb.DeliveryChartDataProto{
			OnTimeDelivery:           chartDataPointsToProto("On-Time Delivery %", "line", result.ChartData.OnTimeDelivery),
			AverageDeliveryTime:      chartDataPointsToProto("Average Delivery Time (Days)", "line", result.ChartData.AverageDeliveryTime),
			AverageFirstShipmentTime: chartDataPointsToProto("Average Time to First Shipment (Days)", "line", result.ChartData.AverageFirstShipmentTime),
		},
	}, nil
}

func (h *gRPCHandler) AnalyzeManufacturing(ctx context.Context, req *pb.AnalyzeManufacturingRequest) (*pb.AnalyzeManufacturingResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.AnalyzeManufacturingParams{
		Type: req.Type,
	}

	if req.StartDate != nil {
		params.StartDate = req.StartDate.AsTime()
	}
	if req.EndDate != nil {
		params.EndDate = req.EndDate.AsTime()
	}

	value, apiErr := h.analyticsSvc.AnalyzeManufacturing(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.AnalyzeManufacturingResponse{
		Value: value,
	}, nil
}

func (h *gRPCHandler) AnalyzeManufacturingBatch(ctx context.Context, req *pb.AnalyzeManufacturingBatchRequest) (*pb.AnalyzeManufacturingBatchResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.AnalyzeManufacturingBatchParams{
		CustomerIDs:      req.CustomerIds,
		ProductLineIDs:   req.ProductLineIds,
		CustomerGroupIDs: req.CustomerGroupIds,
		ItemIDs:          req.ItemIds,
	}

	if req.StartDate != nil {
		params.StartDate = req.StartDate.AsTime()
	}
	if req.EndDate != nil {
		params.EndDate = req.EndDate.AsTime()
	}
	if req.ComparisonStartDate != nil {
		params.ComparisonStartDate = req.ComparisonStartDate.AsTime()
	}
	if req.ComparisonEndDate != nil {
		params.ComparisonEndDate = req.ComparisonEndDate.AsTime()
	}

	result, apiErr := h.analyticsSvc.AnalyzeManufacturingBatch(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.AnalyzeManufacturingBatchResponse{
		Current:    manufacturingMetricsToProto(result.Current),
		Comparison: manufacturingMetricsToProto(result.Comparison),
	}, nil
}

func (h *gRPCHandler) AnalyzeOrders(ctx context.Context, req *pb.AnalyzeOrdersRequest) (*pb.AnalyzeOrdersResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.AnalyzeOrdersParams{
		SalesRepIDs:      req.SalesRepIds,
		ProductLineIDs:   req.ProductLineIds,
		CustomerIDs:      req.CustomerIds,
		CustomerGroupIDs: req.CustomerGroupIds,
		IsSalesRep:       req.IsSalesRep,
	}

	entries, apiErr := h.analyticsSvc.AnalyzeOrders(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbEntries := make([]*pb.OrderEntryProto, len(entries))
	for i, e := range entries {
		entry := &pb.OrderEntryProto{
			Id:                  e.ID,
			CustomerPo:          derefString(e.CustomerPO),
			OrderNumber:         e.OrderNumber,
			OrderId:             e.OrderID,
			SalesRepId:          derefString(e.SalesRepID),
			SalesRepUsername:    derefString(e.SalesRepUsername),
			CustomerId:          e.CustomerID,
			ParentCustomerId:    derefString(e.ParentCustomerID),
			CustomerName:        e.CustomerName,
			CustomerNumber:      e.CustomerNumber,
			CustomerTypeGroupId: derefString(e.CustomerTypeGroupID),
			CustomerGroupName:   derefString(e.CustomerGroupName),
			CustomerCreatedAt:   timestamppb.New(e.CustomerCreatedAt),
			ProductLineId:       derefString(e.ProductLineID),
			ProductLine:         derefString(e.ProductLine),
			ProductTypeId:       e.ProductTypeCode,
			ItemId:              e.ItemID,
			ProductSku:          e.ProductSku,
			ProductDescription:  derefString(e.ProductDescription),
			CategoryName:        e.CategoryName,
			QuantityInvoiced:    e.QuantityInvoiced,
			Unit:                e.Unit,
			UnitCost:            e.UnitCost,
			UnitPrice:           e.UnitPrice,
			UnitProfit:          e.UnitProfit,
			TotalInvoiced:       e.TotalInvoiced,
			TotalCost:           e.TotalCost,
			TotalProfit:         e.TotalProfit,
			ShipToCity:          derefString(e.ShipToCity),
			ShipToZipcode:       derefString(e.ShipToZipcode),
			ShipToState:         derefString(e.ShipToState),
			ShipToCountry:       derefString(e.ShipToCountry),
			OrderDiscountCode:   derefString(e.OrderDiscountCode),
			QuantityOrdered:     e.QuantityOrdered,
			QuantityBackOrdered: e.QuantityBackOrdered,
			TotalOrdered:        e.TotalOrdered,
			TotalBackOrdered:    e.TotalBackOrdered,
		}
		if e.IssuedAt != nil {
			entry.IssuedAt = timestamppb.New(*e.IssuedAt)
		}
		if e.CompletedAt != nil {
			entry.CompletedAt = timestamppb.New(*e.CompletedAt)
		}
		if e.FirstShipAt != nil {
			entry.FirstShipAt = timestamppb.New(*e.FirstShipAt)
		}
		if e.PromisedAt != nil {
			entry.PromisedAt = timestamppb.New(*e.PromisedAt)
		}
		pbEntries[i] = entry
	}

	return &pb.AnalyzeOrdersResponse{
		Entries: pbEntries,
	}, nil
}

func (h *gRPCHandler) AnalyzeQuarterlyOrders(ctx context.Context, req *pb.AnalyzeQuarterlyOrdersRequest) (*pb.AnalyzeQuarterlyOrdersResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.AnalyzeQuarterlyOrdersParams{
		SalesRepIDs:      req.SalesRepIds,
		ItemIDs:          req.ItemIds,
		ProductLineIDs:   req.ProductLineIds,
		CustomerIDs:      req.CustomerIds,
		CustomerGroupIDs: req.CustomerGroupIds,
	}

	years, apiErr := h.analyticsSvc.AnalyzeQuarterlyOrders(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	dataMap := make(map[string]*pb.QuarterlyDataProto, len(years))
	for _, y := range years {
		key := fmt.Sprintf("%d", y.Year)
		dataMap[key] = &pb.QuarterlyDataProto{
			Q1:    y.Data.Q1,
			Q2:    y.Data.Q2,
			Q3:    y.Data.Q3,
			Q4:    y.Data.Q4,
			Total: y.Data.Total,
		}
	}

	return &pb.AnalyzeQuarterlyOrdersResponse{
		Data: dataMap,
	}, nil
}

func (h *gRPCHandler) AnalyzeMaterials(ctx context.Context, req *pb.AnalyzeMaterialsRequest) (*pb.AnalyzeMaterialsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.AnalyzeMaterialsParams{
		SalesOrderIDs: req.SalesOrderIds,
		SupplierIDs:   req.SupplierIds,
	}

	entries, apiErr := h.analyticsSvc.AnalyzeMaterials(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbEntries := make([]*pb.MaterialAnalyticsEntryProto, len(entries))
	for i, e := range entries {
		entry := &pb.MaterialAnalyticsEntryProto{
			Id:     e.MaterialID,
			ItemId: e.ItemID,
			Sku:    e.Sku,
			QuantityInInventory: &pb.BaseQuantity{
				Measure: e.QuantityInInventory.Measure,
				Unit: &pb.BaseQuantityUnitProto{
					Name:         e.QuantityInInventory.UnitName,
					Abbreviation: e.QuantityInInventory.UnitAbbreviation,
					Type:         e.QuantityInInventory.UnitType,
				},
			},
			QuantityInDemand: &pb.BaseQuantity{
				Measure: e.QuantityInDemand.Measure,
				Unit: &pb.BaseQuantityUnitProto{
					Name:         e.QuantityInDemand.UnitName,
					Abbreviation: e.QuantityInDemand.UnitAbbreviation,
					Type:         e.QuantityInDemand.UnitType,
				},
			},
			SupplierNames:       e.SupplierNames,
			SupplierPartNumbers: e.SupplierPartNumbers,
		}

		if e.Description != nil {
			entry.Description = *e.Description
		}

		if e.OrderPoint != nil {
			entry.OrderPoint = &pb.BaseQuantity{
				Measure: e.OrderPoint.Measure,
				Unit: &pb.BaseQuantityUnitProto{
					Name:         e.OrderPoint.UnitName,
					Abbreviation: e.OrderPoint.UnitAbbreviation,
					Type:         e.OrderPoint.UnitType,
				},
			}
		}

		if e.LeadTime != nil {
			entry.LeadTime = &pb.BaseQuantity{
				Measure: e.LeadTime.Measure,
				Unit: &pb.BaseQuantityUnitProto{
					Name:         e.LeadTime.UnitName,
					Abbreviation: e.LeadTime.UnitAbbreviation,
					Type:         e.LeadTime.UnitType,
				},
			}
		}

		ugUnits := make([]*pb.AnalyticsUnitGroupUnitProto, len(e.UnitGroup.Units))
		for j, u := range e.UnitGroup.Units {
			ugUnits[j] = &pb.AnalyticsUnitGroupUnitProto{
				Id:               u.ID,
				Name:             u.Name,
				Abbreviation:     u.Abbreviation,
				ConversionFactor: u.ConversionFactor,
				IsBaseUnit:       u.IsBaseUnit,
			}
		}
		entry.UnitGroup = &pb.AnalyticsUnitGroup{
			Id:    e.UnitGroup.ID,
			Name:  e.UnitGroup.Name,
			Units: ugUnits,
		}

		pbEntries[i] = entry
	}

	return &pb.AnalyzeMaterialsResponse{
		Entries: pbEntries,
	}, nil
}

func (h *gRPCHandler) AnalyzeInventoryReceipts(ctx context.Context, req *pb.AnalyzeInventoryReceiptsRequest) (*pb.AnalyzeInventoryReceiptsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.AnalyzeInventoryReceiptsParams{
		ItemIDs:     req.ItemIds,
		LocationIDs: req.LocationIds,
		LotIDs:      req.LotIds,
	}

	entries, apiErr := h.analyticsSvc.AnalyzeInventoryReceipts(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbEntries := make([]*pb.InventoryReceiptEntryProto, len(entries))
	for i, e := range entries {
		entry := &pb.InventoryReceiptEntryProto{
			Item: &pb.LightItemProto{
				Id:          e.ItemID,
				Sku:         e.ProductSku,
				Description: derefString(e.ProductDescription),
			},
			OwnerAccount: &pb.BasicInfoProto{
				Id:   e.OwnerAccountID,
				Name: e.OwnerAccountName,
			},
			HolderAccount: &pb.BasicInfoProto{
				Id:   e.HolderAccountID,
				Name: e.HolderAccountName,
			},
			RemainingQuantity: &pb.BaseQuantity{
				Measure: e.RemainingQuantity,
				Unit: &pb.BaseQuantityUnitProto{
					Name:         e.UnitName,
					Abbreviation: e.Unit,
				},
			},
			WeightedAverageUnitCost: &pb.AnalyticsRateProto{
				Numerator: &pb.BaseQuantity{
					Measure: e.WeightedAverageUnitCost,
					Unit: &pb.BaseQuantityUnitProto{
						Name:         e.CostNumeratorUnitName,
						Abbreviation: e.CostNumeratorUnitAbbreviation,
					},
				},
				Denominator: &pb.BaseQuantity{
					Measure: 1,
					Unit: &pb.BaseQuantityUnitProto{
						Name:         e.CostDenominatorUnitName,
						Abbreviation: e.CostDenominatorUnitAbbreviation,
					},
				},
			},
			InventoryValue: &pb.BaseQuantity{
				Measure: e.InventoryValue,
				Unit: &pb.BaseQuantityUnitProto{
					Name:         e.CostNumeratorUnitName,
					Abbreviation: e.CostNumeratorUnitAbbreviation,
				},
			},
		}
		if e.OldestReceiptAt != nil {
			entry.OldestReceiptAt = timestamppb.New(*e.OldestReceiptAt)
		}
		if e.NewestReceiptAt != nil {
			entry.NewestReceiptAt = timestamppb.New(*e.NewestReceiptAt)
		}
		if e.LocationID != nil {
			entry.Location = &pb.BasicInfoProto{
				Id:   *e.LocationID,
				Name: derefString(e.LocationName),
			}
		}
		if e.LotID != nil {
			entry.Lot = &pb.AnalyticsLotProto{
				Id:     *e.LotID,
				Number: derefString(e.LotNumber),
			}
		}
		pbEntries[i] = entry
	}

	return &pb.AnalyzeInventoryReceiptsResponse{
		Entries: pbEntries,
	}, nil
}

func (h *gRPCHandler) AnalyzeNewCustomers(ctx context.Context, req *pb.AnalyzeNewCustomersRequest) (*pb.AnalyzeNewCustomersResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.GetNewCustomersAnalyticsParams{
		CustomerGroupIDs: req.CustomerGroupIds,
		SalesRepIDs:      req.SalesRepIds,
	}

	if req.StartDate != nil {
		params.StartDate = req.StartDate.AsTime()
	}
	if req.EndDate != nil {
		params.EndDate = req.EndDate.AsTime()
	}

	entries, apiErr := h.analyticsSvc.GetNewCustomersAnalytics(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	data := make([]*pb.DateTimeCoordinateProto, len(entries))
	for i, e := range entries {
		data[i] = &pb.DateTimeCoordinateProto{
			X: timestamppb.New(e.CreatedAt),
			Y: 1,
		}
	}

	return &pb.AnalyzeNewCustomersResponse{
		NewCustomers: &pb.NewCustomersChartDataProto{
			Label: "New Customers",
			Data:  data,
		},
	}, nil
}

func (h *gRPCHandler) AnalyzeDemandForecast(ctx context.Context, req *pb.AnalyzeDemandForecastRequest) (*pb.AnalyzeDemandForecastResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.GetDemandForecastParams{
		ProductLineIDs: req.ProductLineIds,
		ItemIDs:        req.ItemIds,
		HistoryMonths:  req.HistoryMonths,
		ForecastMonths: req.ForecastMonths,
	}

	result, apiErr := h.analyticsSvc.GetDemandForecast(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbRows := make([]*pb.DemandForecastRowProto, len(result.Items))
	for i, item := range result.Items {
		pbRows[i] = &pb.DemandForecastRowProto{
			ItemId:              item.ItemID,
			ProductLineId:       derefString(item.ProductLineID),
			ProductSku:          item.ProductSku,
			ProductDescription:  derefString(item.ProductDescription),
			Unit:                item.Unit,
			Currency:            item.Currency,
			History:             demandHistoryPointsToProto(item.History),
			Forecast:            demandForecastPointsToProto(item.Forecast),
			RevenueHistory:      revenueHistoryPointsToProto(item.RevenueHistory),
			RevenueForecast:     revenueForecastPointsToProto(item.RevenueForecast),
			SalesHistory:        revenueHistoryPointsToProto(item.SalesHistory),
			SalesForecast:       revenueForecastPointsToProto(item.SalesForecast),
			CurrentMonthDemand:  item.CurrentMonthDemand,
			CurrentMonthRevenue: item.CurrentMonthRevenue,
			CurrentMonthSales:   item.CurrentMonthSales,
		}
	}

	return &pb.AnalyzeDemandForecastResponse{
		Rows:                 pbRows,
		CurrentMonthFraction: result.CurrentMonthFraction,
	}, nil
}

func (h *gRPCHandler) AnalyzeOeeTrend(ctx context.Context, req *pb.AnalyzeOeeTrendRequest) (*pb.AnalyzeOeeTrendResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.AnalyzeOeeTrendParams{DepartmentIDs: req.DepartmentIds}
	if req.StartDate != nil {
		params.StartDate = req.StartDate.AsTime()
	}
	if req.EndDate != nil {
		params.EndDate = req.EndDate.AsTime()
	}

	periods, apiErr := h.analyticsSvc.AnalyzeOeeTrend(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbPeriods := make([]*pb.OeeTrendPeriodProto, len(periods))
	for i, p := range periods {
		pbPeriods[i] = &pb.OeeTrendPeriodProto{
			StartsAt:                timestamppb.New(p.StartsAt),
			EndsAt:                  timestamppb.New(p.EndsAt),
			GoodUnits:               p.GoodUnits,
			WasteUnits:              p.WasteUnits,
			SecondsUnits:            p.SecondsUnits,
			StandardSecondsEarned:   p.StandardSecondsEarned,
			ScheduledSeconds:        p.ScheduledSeconds,
			RunTimeSeconds:          p.RunTimeSeconds,
			AvailabilityLossSeconds: p.AvailabilityLossSeconds,
			NotScheduledSeconds:     p.NotScheduledSeconds,
			AvailabilityPct:         p.AvailabilityPct,
			PerformancePct:          p.PerformancePct,
			QualityPct:              p.QualityPct,
			OeePct:                  p.OeePct,
			HasDowntimeData:         p.HasDowntimeData,
			DowntimeEventCount:      p.DowntimeEventCount,
		}
	}

	return &pb.AnalyzeOeeTrendResponse{Periods: pbPeriods}, nil
}

func (h *gRPCHandler) AnalyzeOee(ctx context.Context, req *pb.AnalyzeOeeRequest) (*pb.AnalyzeOeeResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.AnalyzeOeeParams{
		DepartmentIDs:    req.DepartmentIds,
		PlannedTimeHours: make(map[string]float64, len(req.PlannedTime)),
	}

	if req.StartDate != nil {
		params.StartDate = req.StartDate.AsTime()
	}
	if req.EndDate != nil {
		params.EndDate = req.EndDate.AsTime()
	}
	for _, pt := range req.PlannedTime {
		params.PlannedTimeHours[pt.DepartmentId] = pt.PlannedHours
	}

	departments, apiErr := h.analyticsSvc.AnalyzeOee(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbDepartments := make([]*pb.OeeDepartmentProto, len(departments))
	for i, d := range departments {
		breakdown := make([]*pb.OeeDowntimeReasonProto, len(d.DowntimeBreakdown))
		for j, r := range d.DowntimeBreakdown {
			breakdown[j] = &pb.OeeDowntimeReasonProto{
				ReasonCode:      r.ReasonCode,
				OeeBucket:       r.OeeBucket,
				DowntimeSeconds: r.DowntimeSeconds,
				EventCount:      r.EventCount,
			}
		}

		pbDepartments[i] = &pb.OeeDepartmentProto{
			DepartmentId:            d.DepartmentID,
			DepartmentName:          d.DepartmentName,
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
			DowntimeBreakdown:       breakdown,
			ScheduledSeconds:        d.ScheduledSeconds,
			RunTimeSeconds:          d.RunTimeSeconds,
			AvailabilityPct:         d.AvailabilityPct,
			PerformancePct:          d.PerformancePct,
			QualityPct:              d.QualityPct,
			OeePct:                  d.OeePct,
			HasDowntimeData:         d.HasDowntimeData,
			HasPerformanceAnomaly:   d.HasPerformanceAnomaly,
		}
	}

	return &pb.AnalyzeOeeResponse{
		Departments: pbDepartments,
	}, nil
}

func (h *gRPCHandler) AnalyzeWeeksOfSales(ctx context.Context, req *pb.AnalyzeWeeksOfSalesRequest) (*pb.AnalyzeWeeksOfSalesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	var periodInWeeks int32 = 4
	if req.PeriodInWeeks != nil {
		periodInWeeks = *req.PeriodInWeeks
	}

	result, apiErr := h.analyticsSvc.AnalyzeWeeksOfSales(ctx, domain.AnalyzeWeeksOfSalesParams{
		PeriodInWeeks: periodInWeeks,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbItems := make([]*pb.WeeksOfSalesItemProto, len(result.Items))
	for i, item := range result.Items {
		pbItems[i] = &pb.WeeksOfSalesItemProto{
			ProductLineId:                        item.ProductLineID,
			ProductLineName:                      item.ProductLineName,
			QuantityOnHand:                       item.QuantityOnHand,
			QuantityOnHandUnitAbbreviation:       item.QuantityOnHandUnitAbbreviation,
			QuantityOnHandUnitType:               item.QuantityOnHandUnitType,
			AverageSalesQuantity:                 item.AverageSalesQuantity,
			AverageSalesQuantityUnitAbbreviation: item.AverageSalesQuantityUnitAbbreviation,
			AverageSalesQuantityUnitType:         item.AverageSalesQuantityUnitType,
			WeeksOfSales:                         item.WeeksOfSales,
		}
	}

	return &pb.AnalyzeWeeksOfSalesResponse{
		Items: pbItems,
		Count: result.Count,
	}, nil
}

func attainmentBucketToProto(b domain.AttainmentBucket) *pb.AttainmentBucketInfo {
	info := &pb.AttainmentBucketInfo{
		Key:               b.Key,
		Label:             b.Label,
		PlannedQuantity:   b.PlannedQuantity,
		ActualQuantity:    b.ActualQuantity,
		MatchedQuantity:   b.MatchedQuantity,
		WasteQuantity:     b.WasteQuantity,
		UnplannedQuantity: b.UnplannedQuantity,
		PlannedRunHours:   b.PlannedRunHours,
		PlannedLines:      b.PlannedLines,
		BatchCount:        b.BatchCount,
		// Left nil when there was nothing planned: a bucket with no plan has no attainment, and 0% would read as a total miss.
		AttainmentPct:  b.AttainmentPct,
		OutputRatioPct: b.OutputRatioPct,
	}
	if b.WeekStartDate != nil {
		info.WeekStartDate = timestamppb.New(*b.WeekStartDate)
	}
	return info
}

func (h *gRPCHandler) AnalyzeScheduleAttainment(ctx context.Context, req *pb.AnalyzeScheduleAttainmentRequest) (*pb.AnalyzeScheduleAttainmentResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.AnalyzeScheduleAttainmentParams{
		GroupBy:       req.GroupBy,
		MachineIDs:    req.MachineIds,
		DepartmentIDs: req.DepartmentIds,
	}
	if req.StartDate != nil {
		params.StartDate = req.StartDate.AsTime()
	}
	if req.EndDate != nil {
		params.EndDate = req.EndDate.AsTime()
	}

	result, apiErr := h.analyticsSvc.AnalyzeScheduleAttainment(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	buckets := make([]*pb.AttainmentBucketInfo, len(result.Buckets))
	for i, b := range result.Buckets {
		buckets[i] = attainmentBucketToProto(b)
	}

	adherence := make([]*pb.FrozenAdherenceInfo, len(result.FrozenAdherence))
	for i, a := range result.FrozenAdherence {
		entry := &pb.FrozenAdherenceInfo{
			ScheduleId:            a.ScheduleID,
			Version:               a.Version,
			FrozenLineCount:       a.FrozenLineCount,
			FrozenPlannedQuantity: a.FrozenPlannedQuantity,
			DeviatedLines:         a.DeviatedLines,
			AddedLines:            a.AddedLines,
			AbsDeltaUnits:         a.AbsDeltaUnits,
			LineAdherencePct:      a.LineAdherence,
			UnitsAdherencePct:     a.UnitsAdherence,
			OffPlanLines:          a.OffPlanLines,
			OffPlanQuantity:       a.OffPlanQuantity,
		}
		if a.FrozenThroughAt != nil {
			entry.FrozenThroughDate = timestamppb.New(*a.FrozenThroughAt)
		}
		adherence[i] = entry
	}

	return &pb.AnalyzeScheduleAttainmentResponse{
		StartDate:             timestamppb.New(result.StartDate),
		EndDate:               timestamppb.New(result.EndDate),
		GroupBy:               result.GroupBy,
		BaselineScheduleIds:   result.BaselineScheduleIDs,
		Buckets:               buckets,
		Totals:                attainmentBucketToProto(result.Totals),
		FrozenAdherence:       adherence,
		HasBaseline:           result.HasBaseline,
		ScheduledMachineCount: result.ScheduledMachineCount,
	}, nil
}

func deliveryPerformanceToProto(p scheduling.DeliveryPerformance, includePeriod bool) *pb.DeliveryPerformanceProto {
	out := &pb.DeliveryPerformanceProto{
		CommittedOrderCount:          safeconv.IntToInt32(p.CommittedOrderCount),
		ShippedOrderCount:            safeconv.IntToInt32(p.ShippedOrderCount),
		OnTimeOrderCount:             safeconv.IntToInt32(p.OnTimeOrderCount),
		OnTimeInFullCount:            safeconv.IntToInt32(p.OnTimeInFullCount),
		LateOrderCount:               safeconv.IntToInt32(p.LateOrderCount),
		NotYetShippedCount:           safeconv.IntToInt32(p.NotYetShippedCount),
		OnTimePct:                    p.OnTimePct,
		OnTimeInFullPct:              p.OnTimeInFullPct,
		AverageDaysLate:              p.AverageDaysLate,
		AverageLeadTimeDays:          p.AverageLeadTimeDays,
		AverageCommittedLeadTimeDays: p.AverageCommittedLeadTimeDays,
	}
	if includePeriod && !p.PeriodStart.IsZero() {
		out.PeriodStart = timestamppb.New(p.PeriodStart)
	}
	return out
}

func (h *gRPCHandler) AnalyzeDeliveryPerformance(ctx context.Context, req *pb.AnalyzeDeliveryPerformanceRequest) (*pb.AnalyzeDeliveryPerformanceResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.AnalyzeDeliveryPerformanceParams{
		Granularity: req.Granularity,
		DeliveryFilters: domain.DeliveryFilters{
			CustomerIDs:      req.CustomerIds,
			CustomerGroupIDs: req.CustomerGroupIds,
			ProductLineIDs:   req.ProductLineIds,
			SalesRepIDs:      req.SalesRepIds,
		},
	}
	if req.StartsAt != nil {
		params.StartDate = req.StartsAt.AsTime()
	}
	if req.EndsAt != nil {
		params.EndDate = req.EndsAt.AsTime()
	}

	result, apiErr := h.analyticsSvc.AnalyzeDeliveryPerformance(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	periods := make([]*pb.DeliveryPerformanceProto, 0, len(result.Periods))
	for _, p := range result.Periods {
		periods = append(periods, deliveryPerformanceToProto(p, true))
	}

	backlog := make([]*pb.DeliveryBacklogBucketProto, 0, len(result.Backlog))
	for _, b := range result.Backlog {
		backlog = append(backlog, &pb.DeliveryBacklogBucketProto{
			Label:       b.Label,
			MinDaysLate: safeconv.IntToInt32(b.MinDaysLate),
			MaxDaysLate: safeconv.IntToInt32(b.MaxDaysLate),
			OrderCount:  safeconv.IntToInt32(b.OrderCount),
			Units:       b.Units,
		})
	}

	lateness := make([]*pb.DeliveryLatenessBucketProto, 0, len(result.Lateness))
	for _, b := range result.Lateness {
		lateness = append(lateness, &pb.DeliveryLatenessBucketProto{
			Label:        b.Label,
			MinDaysLate:  safeconv.IntToInt32(b.MinDaysLate),
			MaxDaysLate:  safeconv.IntToInt32(b.MaxDaysLate),
			OrderCount:   safeconv.IntToInt32(b.OrderCount),
			ShippedCount: safeconv.IntToInt32(b.ShippedCount),
			Units:        b.Units,
		})
	}

	return &pb.AnalyzeDeliveryPerformanceResponse{
		Overall:               deliveryPerformanceToProto(result.Overall, false),
		Periods:               periods,
		Backlog:               backlog,
		Lateness:              lateness,
		ByCustomer:            deliveryBreakdownsToProto(result.ByCustomer),
		ByCustomerGroup:       deliveryBreakdownsToProto(result.ByCustomerGroup),
		ByProductLine:         deliveryBreakdownsToProto(result.ByProductLine),
		ByCommitmentSource:    deliveryBreakdownsToProto(result.ByCommitmentSource),
		UncommittedOrderCount: safeconv.IntToInt32(result.UncommittedOrderCount),
	}, nil
}

// deliveryBreakdownsToProto maps one dimension's rows. The period start is never included: a breakdown row is a whole window, not a period inside it.
func deliveryBreakdownsToProto(rows []scheduling.DeliveryBreakdown) []*pb.DeliveryBreakdownProto {
	out := make([]*pb.DeliveryBreakdownProto, 0, len(rows))
	for _, row := range rows {
		out = append(out, &pb.DeliveryBreakdownProto{
			Key:         row.Key,
			Label:       row.Label,
			Performance: deliveryPerformanceToProto(row.DeliveryPerformance, false),
		})
	}
	return out
}

func (h *gRPCHandler) AnalyzeRealizedMargins(ctx context.Context, req *pb.AnalyzeRealizedMarginsRequest) (*pb.AnalyzeRealizedMarginsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.analyticsSvc.AnalyzeRealizedMargins(ctx, domain.AnalyzeRealizedMarginsParams{
		StartDate:         req.StartDate.AsTime(),
		EndDate:           req.EndDate.AsTime(),
		CustomerIDs:       req.CustomerIds,
		CustomerGroupIDs:  req.CustomerGroupIds,
		ProductLineIDs:    req.ProductLineIds,
		TargetGrossMargin: req.TargetGrossMargin,
		OutlierTolerance:  req.OutlierTolerance,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	findings := make([]*pb.RealizedMarginFindingProto, len(result.Findings))
	for i, f := range result.Findings {
		findings[i] = &pb.RealizedMarginFindingProto{
			CustomerId:              f.CustomerID,
			CustomerGroupId:         f.CustomerGroupID,
			ItemId:                  f.ItemID,
			ProductLineId:           f.ProductLineID,
			UnitAbbreviation:        f.UnitAbbreviation,
			QuantityInvoiced:        f.QuantityInvoiced,
			Revenue:                 f.Revenue,
			Cost:                    f.Cost,
			AverageUnitPrice:        f.AverageUnitPrice,
			PeerMedianPrice:         f.PeerMedianPrice,
			BelowPeerMedianFraction: f.BelowPeerMedianFraction,
			GrossMargin:             f.GrossMargin,
			LineCount:               safeconv.IntToInt32(f.LineCount),
			Reason:                  f.Reason,
		}
	}

	return &pb.AnalyzeRealizedMarginsResponse{
		Findings:               findings,
		LinesAnalyzed:          safeconv.IntToInt32(result.LinesAnalyzed),
		RelationshipsAnalyzed:  safeconv.IntToInt32(result.RelationshipsAnalyzed),
		BelowPeerMedianCount:   safeconv.IntToInt32(result.BelowPeerMedianCount),
		BelowTargetMarginCount: safeconv.IntToInt32(result.BelowTargetMarginCount),
		MarginNotAssessedCount: safeconv.IntToInt32(result.MarginNotAssessedCount),
	}, nil
}

func (h *gRPCHandler) AnalyzeCustomerPricing(ctx context.Context, req *pb.AnalyzeCustomerPricingRequest) (*pb.AnalyzeCustomerPricingResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.analyticsSvc.AnalyzeCustomerPricing(ctx, domain.AnalyzeCustomerPricingParams{
		CustomerIDs:       req.CustomerIds,
		CustomerGroupIDs:  req.CustomerGroupIds,
		TargetGrossMargin: req.TargetGrossMargin,
		OutlierTolerance:  req.OutlierTolerance,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	findings := make([]*pb.CustomerPricingFindingProto, len(result.Findings))
	for i, f := range result.Findings {
		findings[i] = &pb.CustomerPricingFindingProto{
			AccountPriceId:          f.AccountPriceID,
			CustomerId:              f.CustomerID,
			ProductLineId:           f.ProductLineID,
			AttributeIds:            f.AttributeIDs,
			UnitPrice:               f.UnitPrice,
			NumeratorUnitId:         f.NumeratorUnitID,
			NumeratorUnitAbbr:       f.NumeratorUnitAbbr,
			DenominatorUnitId:       f.DenominatorUnitID,
			DenominatorAbbr:         f.DenominatorAbbr,
			PeerMedianPrice:         f.PeerMedianPrice,
			BelowPeerMedianFraction: f.BelowPeerMedianFraction,
			GrossMargin:             f.GrossMargin,
			Origin:                  f.Origin,
			Reason:                  f.Reason,
		}
	}

	return &pb.AnalyzeCustomerPricingResponse{
		Findings:               findings,
		PricesAnalyzed:         safeconv.IntToInt32(result.PricesAnalyzed),
		BelowPeerMedianCount:   safeconv.IntToInt32(result.BelowPeerMedianCount),
		BelowTargetMarginCount: safeconv.IntToInt32(result.BelowTargetMarginCount),
		MarginNotAssessedCount: safeconv.IntToInt32(result.MarginNotAssessedCount),
		Notes:                  result.Notes,
	}, nil
}
