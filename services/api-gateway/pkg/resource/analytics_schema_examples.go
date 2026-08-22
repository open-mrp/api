package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

func analyticsExampleTime() time.Time {
	return timeutil.TimestampToTime(sampleCreatedAtTimestamp)
}

func sampleSalesEntry() SalesEntry {
	t := analyticsExampleTime()
	inv := analyticsExampleTime()
	srep := SampleAccountUserID
	return SalesEntry{
		ID:                "sale_analytics_doc_01061c4dc0d10772b6f05d1446",
		IssuedAt:          &t,
		OrderNumber:       SampleSalesOrderNumber,
		OrderID:           SampleSalesOrderID,
		SalesRepID:        &srep,
		CustomerID:        SampleCustomerID,
		CustomerName:      SampleCustomerName,
		CustomerNumber:    SampleCustomerNumber,
		CustomerCreatedAt: t,
		ProductTypeID:     SampleProductTypeID,
		ItemID:            SampleItemID,
		ProductSku:        SampleItemSKU,
		CategoryName:      SampleItemCategoryName,
		QuantityInvoiced:  10,
		Unit:              SampleUnitAbbreviation,
		UnitCost:          5,
		UnitPrice:         12,
		UnitProfit:        7,
		TotalInvoiced:     120,
		TotalCost:         50,
		TotalProfit:       70,
		InvoiceID:         SampleInvoiceID,
		InvoiceNumber:     "INV-001",
		InvoicedAt:        inv,
	}
}

func sampleOrderEntry() OrderEntry {
	t := analyticsExampleTime()
	e := sampleSalesEntry()
	return OrderEntry{
		ID:                  e.ID + "_order",
		IssuedAt:            e.IssuedAt,
		OrderNumber:         e.OrderNumber,
		OrderID:             e.OrderID,
		SalesRepID:          e.SalesRepID,
		SalesRepUsername:    e.SalesRepUsername,
		CustomerID:          e.CustomerID,
		CustomerName:        e.CustomerName,
		CustomerNumber:      e.CustomerNumber,
		CustomerTypeGroupID: e.CustomerTypeGroupID,
		CustomerGroupName:   e.CustomerGroupName,
		ParentCustomerID:    e.ParentCustomerID,
		CustomerCreatedAt:   e.CustomerCreatedAt,
		ProductLineID:       e.ProductLineID,
		ProductLine:         e.ProductLine,
		ProductTypeID:       e.ProductTypeID,
		ItemID:              e.ItemID,
		ProductSku:          e.ProductSku,
		ProductDescription:  e.ProductDescription,
		CategoryName:        e.CategoryName,
		QuantityInvoiced:    e.QuantityInvoiced,
		Unit:                e.Unit,
		UnitCost:            e.UnitCost,
		UnitPrice:           e.UnitPrice,
		UnitProfit:          e.UnitProfit,
		TotalInvoiced:       e.TotalInvoiced,
		TotalCost:           e.TotalCost,
		TotalProfit:         e.TotalProfit,
		ShipToCity:          e.ShipToCity,
		ShipToZipcode:       e.ShipToZipcode,
		ShipToState:         e.ShipToState,
		ShipToCountry:       e.ShipToCountry,
		OrderDiscountCode:   e.OrderDiscountCode,
		CompletedAt:         &t,
		FirstShipAt:         &t,
		PromisedAt:          &t,
		QuantityOrdered:     12,
		QuantityBackOrdered: 2,
		TotalOrdered:        144,
		TotalBackOrdered:    24,
	}
}

func (*AnalyzeSalesResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&AnalyzeSalesResponse{
		Object: constants.ObjectTypeList,
		Data:   []SalesEntry{sampleSalesEntry()},
	})
}

func (*AnalyzeOpenBatchesResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&AnalyzeOpenBatchesResponse{
		Object: constants.ObjectTypeList,
		Data:   []OpenBatchSummary{*SampleOpenBatchSummary},
	})
}

func (*AnalyzeProductionCostsResponse) SchemaExample() any {
	categoryName := SampleItemCategoryName
	categoryEnt := NewEntity(SampleItemCategoryID, constants.ObjectTypeItemCategory, &categoryName, nil)
	return apiexample.ValidateAndMarshalToMap(&AnalyzeProductionCostsResponse{
		Object: constants.ObjectTypeList,
		Data: []ProductionCostItem{
			{
				Category:        categoryEnt,
				TotalCosts:      sampleCostBreakdown(),
				ProductiveCosts: sampleCostBreakdown(),
				WasteCosts:      sampleCostBreakdown(),
				SecondsCosts:    sampleCostBreakdown(),
			},
		},
	})
}

func sampleCostBreakdown() CostBreakdown {
	return CostBreakdown{
		Total:     SampleQuantity,
		Labor:     SampleQuantity,
		Materials: SampleQuantity,
		Overhead:  SampleQuantity,
		Time:      SampleQuantity,
		Quantity:  SampleQuantity,
	}
}

func (*AnalyzeDeliveriesResponse) SchemaExample() any {
	coords := []Coordinate{{X: 1, Y: 2}}
	return apiexample.ValidateAndMarshalToMap(&AnalyzeDeliveriesResponse{
		Object: constants.ObjectTypeAnalyzeDeliveriesResponse,
		Statistics: DeliveryStatistics{
			AverageTimeToFirstShipment:            ptrFloat64(3.5),
			AverageTimeToCompletion:               ptrFloat64(7),
			OnTimeDeliveryPercentage:              ptrFloat64(92),
			OnTimeFirstShipmentPercentage:         ptrFloat64(88),
			TotalOrders:                           42,
			OrdersWithFirstShipment:               40,
			OrdersWithCompletion:                  38,
			OrdersWithPromiseDate:                 36,
			OrdersPartiallyFulfilledInPromiseDate: 5,
			OrdersCompletedWithinPromiseDate:      30,
		},
		ChartData: DeliveryChartData{
			OnTimeDelivery: ChartData{
				Name: "On-time delivery",
				Type: "line",
				Data: coords,
			},
			AverageDeliveryTime: ChartData{
				Name: "Average delivery time",
				Type: "line",
				Data: coords,
			},
			AverageFirstShipmentTime: ChartData{
				Name: "Average first shipment time",
				Type: "line",
				Data: coords,
			},
		},
	})
}

func ptrFloat64(v float64) *float64 {
	return &v
}

func (*AnalyzeManufacturingResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&AnalyzeManufacturingResponse{
		Object: constants.ObjectTypeAnalyzeManufacturingResponse,
		Value:  125000.5,
	})
}

func (*AnalyzeManufacturingBatchResponse) SchemaExample() any {
	mm := ManufacturingMetrics{
		Production:      100,
		CostsPerUnit:    4.5,
		Margin:          0.22,
		Quality:         0.98,
		LaborEfficiency: 0.91,
	}
	return apiexample.ValidateAndMarshalToMap(&AnalyzeManufacturingBatchResponse{
		Object:     constants.ObjectTypeAnalyzeManufacturingBatchResponse,
		Current:    mm,
		Comparison: mm,
	})
}

func (*AnalyzeOrdersResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&AnalyzeOrdersResponse{
		Object: constants.ObjectTypeList,
		Data:   []OrderEntry{sampleOrderEntry()},
	})
}

func (*AnalyzeQuarterlyOrdersResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&AnalyzeQuarterlyOrdersResponse{
		Object: constants.ObjectTypeAnalyzeQuarterlyOrdersResponse,
		Data: map[string]QuarterlySalesData{
			"2026": {Q1: 100, Q2: 120, Q3: 110, Q4: 130, Total: 460},
		},
	})
}

func (*AnalyzeMaterialsResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&AnalyzeMaterialsResponse{
		Object: constants.ObjectTypeList,
		Data: []MaterialAnalyticsEntry{
			{
				ID:                  "mat_analytics_018f61eaea85de4fc0ee224c0d",
				ItemID:              SampleMaterialID,
				Sku:                 SampleItemSKU,
				QuantityInInventory: SampleQuantity,
				QuantityInDemand:    SampleQuantity,
				UnitGroup: AnalyticsUnitGroup{
					ID:   SampleUnitGroupID,
					Name: SampleUnitGroupName,
					Units: []AnalyticsUnitGroupUnit{
						{
							ID:               SampleUnitID,
							Name:             SampleUnitName,
							Abbreviation:     SampleUnitAbbreviation,
							ConversionFactor: 1,
							IsBaseUnit:       true,
						},
					},
				},
				SupplierNames:       []string{SampleSupplierName},
				SupplierPartNumbers: []string{"SUP-PART-001"},
			},
		},
	})
}

func (*AnalyzeInventoryReceiptsResponse) SchemaExample() any {
	desc := sampleDescription
	acctName := SampleAccountName
	owner := NewEntity(SampleAccountID, constants.ObjectTypeAccount, &acctName, nil)
	holder := owner
	lotNum := "LOT-442"
	return apiexample.ValidateAndMarshalToMap(&AnalyzeInventoryReceiptsResponse{
		Object: constants.ObjectTypeList,
		Data: []InventoryReceiptSummaryEntry{
			{
				Item: AnalyticsItem{
					ID:          SampleItemID,
					Object:      constants.ObjectTypeItem,
					Sku:         SampleItemSKU,
					Description: &desc,
				},
				Location: NewEntity(SampleLocationID, constants.ObjectTypeLocation, nil, nil),
				Lot: &AnalyticsLot{
					ID:     SampleLotID,
					Object: constants.ObjectTypeLot,
					Number: lotNum,
				},
				OwnerAccount:            owner,
				HolderAccount:           holder,
				RemainingQuantity:       SampleQuantity,
				WeightedAverageUnitCost: sampleAnalyticsRate(),
				InventoryValue:          SampleQuantity,
				OldestReceiptAt:         ptrTime(analyticsExampleTime()),
				NewestReceiptAt:         ptrTime(analyticsExampleTime()),
			},
		},
	})
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func sampleAnalyticsRate() AnalyticsRate {
	return AnalyticsRate{
		Numerator:   SampleQuantity,
		Denominator: SampleQuantity,
	}
}

func (*AnalyzeNewCustomersResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&AnalyzeNewCustomersResponse{
		Object: constants.ObjectTypeAnalyzeNewCustomersResponse,
		NewCustomers: NewCustomersData{
			Label: "New customers",
			Data: []DateTimeCoordinate{
				{X: analyticsExampleTime(), Y: 3},
			},
		},
	})
}

var sampleDemandForecastHistoryPoint = DemandForecastPoint{Date: analyticsExampleTime(), Demand: 48}
var sampleDemandForecastForecastPoint = DemandForecastForecastPoint{Date: analyticsExampleTime(), Forecast: 50, LowerBound: 45, UpperBound: 55}
var sampleRevenueForecastPoint = RevenueForecastPoint{Date: analyticsExampleTime(), Revenue: 960}

var SampleAnalyzeDemandForecastResponse = &AnalyzeDemandForecastResponse{
	Object: constants.ObjectTypeAnalyzeDemandForecastResponse,
	Data: NewList([]DemandForecastRow{
		{
			Item:                NewEntity(SampleItemID, constants.ObjectTypeItem, nil, nil),
			ProductLine:         NewEntity(SampleProductLineID, constants.ObjectTypeProductLine, nil, nil),
			ProductSku:          SampleItemSKU,
			Unit:                SampleUnitAbbreviation,
			Currency:            "USD",
			History:             []DemandForecastPoint{sampleDemandForecastHistoryPoint},
			Forecast:            []DemandForecastForecastPoint{sampleDemandForecastForecastPoint},
			RevenueHistory:      []RevenueForecastPoint{sampleRevenueForecastPoint},
			RevenueForecast:     []DemandForecastForecastPoint{sampleDemandForecastForecastPoint},
			SalesHistory:        []RevenueForecastPoint{sampleRevenueForecastPoint},
			SalesForecast:       []DemandForecastForecastPoint{sampleDemandForecastForecastPoint},
			CurrentMonthDemand:  120,
			CurrentMonthRevenue: 2400,
			CurrentMonthSales:   120,
		},
	}, PageInfo{}),
	CurrentMonthFraction: 0.35,
}

func (*AnalyzeDemandForecastResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAnalyzeDemandForecastResponse)
}

var SampleAnalyzeOeeResponse = &AnalyzeOeeResponse{
	Object: constants.ObjectTypeAnalyzeOeeResponse,
	Departments: NewList([]OeeDepartment{
		{
			Department:              NewEntity(SampleDepartmentID, constants.ObjectTypeDepartment, new(SampleDepartmentName), nil),
			GoodUnits:               980,
			WasteUnits:              20,
			SecondsUnits:            5,
			StandardSecondsEarned:   126000,
			EstimatedRuntimeHours:   40,
			AvailabilityLossSeconds: 14850,
			NotScheduledSeconds:     3600,
			DowntimeEventCount:      6,
			ScheduledSeconds:        165000,
			RunTimeSeconds:          150150,
			AvailabilityPct:         &sampleOeeTrendAvailabilityPct,
			PerformancePct:          &sampleOeeTrendPerformancePct,
			QualityPct:              &sampleOeeTrendQualityPct,
			OeePct:                  &sampleOeeTrendOeePct,
			MeasurementStatus:       constants.OeeMeasurementStatusMeasured,
		},
	}, PageInfo{}),
}

func (*AnalyzeOeeResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAnalyzeOeeResponse)
}

var sampleOeeTrendAvailabilityPct = 0.91
var sampleOeeTrendPerformancePct = 0.84
var sampleOeeTrendQualityPct = 0.98
var sampleOeeTrendOeePct = sampleOeeTrendAvailabilityPct * sampleOeeTrendPerformancePct * sampleOeeTrendQualityPct

var SampleAnalyzeOeeTrendResponse = &AnalyzeOeeTrendResponse{
	Object: constants.ObjectTypeAnalyzeOeeTrendResponse,
	Periods: NewList([]OeeTrendPeriod{
		{
			StartsAt:                SampleAnalyticsPeriodStart,
			EndsAt:                  SampleAnalyticsPeriodEnd,
			GoodUnits:               980,
			WasteUnits:              20,
			SecondsUnits:            5,
			StandardSecondsEarned:   126000,
			ScheduledSeconds:        165000,
			RunTimeSeconds:          150150,
			AvailabilityLossSeconds: 14850,
			NotScheduledSeconds:     3600,
			AvailabilityPct:         &sampleOeeTrendAvailabilityPct,
			PerformancePct:          &sampleOeeTrendPerformancePct,
			QualityPct:              &sampleOeeTrendQualityPct,
			OeePct:                  &sampleOeeTrendOeePct,
			MeasurementStatus:       constants.OeeMeasurementStatusMeasured,
			DowntimeEventCount:      6,
		},
	}, PageInfo{}),
}

func (*AnalyzeOeeTrendResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAnalyzeOeeTrendResponse)
}

func (*AnalyzeWeeksOfSalesResponse) SchemaExample() any {
	plName := SampleProductLineName
	plEnt := NewEntity(SampleProductLineID, constants.ObjectTypeProductLine, &plName, nil)
	return apiexample.ValidateAndMarshalToMap(&AnalyzeWeeksOfSalesResponse{
		Object: constants.ObjectTypeAnalyzeWeeksOfSalesResponse,
		Data: []WeeksOfSalesItem{
			{
				ProductLine:          plEnt,
				QuantityOnHand:       SampleQuantity,
				AverageSalesQuantity: SampleQuantity,
				WeeksOfSales:         4.2,
			},
		},
		Count: 1,
	})
}

var sampleAttainmentPct = 92.5
var sampleOutputRatioPct = 96.0
var sampleLineAdherencePct = 87.5
var sampleUnitsAdherencePct = 94.0

var SampleAnalyzeScheduleAttainmentResponse = &AnalyzeScheduleAttainmentResponse{
	Object:    constants.ObjectTypeAnalyzeScheduleAttainmentResponse,
	StartDate: SampleAnalyticsPeriodStart,
	EndDate:   SampleAnalyticsPeriodEnd,
	GroupBy:   constants.AttainmentGroupByWeek,
	BaselineSchedules: NewList([]Entity{
		*NewEntity(SampleProductionScheduleID, constants.ObjectTypeProductionSchedule, nil, nil),
	}, PageInfo{}),
	Buckets: NewList([]AttainmentBucket{
		{
			Key:               "2026-07-27T00:00:00Z",
			Label:             "2026-07-27T00:00:00Z",
			PlannedQuantity:   4200,
			ActualQuantity:    4032,
			MatchedQuantity:   3885,
			WasteQuantity:     48,
			UnplannedQuantity: 147,
			PlannedRunHours:   63,
			PlannedLines:      7,
			BatchCount:        58,
			AttainmentPct:     &sampleAttainmentPct,
			OutputRatioPct:    &sampleOutputRatioPct,
		},
	}, PageInfo{}),
	Totals: AttainmentBucket{
		Key:               "total",
		Label:             "Total",
		PlannedQuantity:   4200,
		ActualQuantity:    4032,
		MatchedQuantity:   3885,
		WasteQuantity:     48,
		UnplannedQuantity: 147,
		PlannedRunHours:   63,
		PlannedLines:      7,
		BatchCount:        58,
		AttainmentPct:     &sampleAttainmentPct,
		OutputRatioPct:    &sampleOutputRatioPct,
	},
	FrozenAdherence: NewList([]FrozenAdherence{
		{
			Schedule:              NewEntity(SampleProductionScheduleID, constants.ObjectTypeProductionSchedule, nil, nil),
			Version:               1,
			FrozenLineCount:       8,
			FrozenPlannedQuantity: 4200,
			DeviatedLines:         1,
			AddedLines:            0,
			AbsDeltaUnits:         252,
			OffPlanLines:          1,
			OffPlanQuantity:       147,
			LineAdherencePct:      &sampleLineAdherencePct,
			UnitsAdherencePct:     &sampleUnitsAdherencePct,
		},
	}, PageInfo{}),
	BaselineStatus:        constants.AttainmentBaselineStatusMeasured,
	ScheduledMachineCount: 2,
}

func (*AnalyzeScheduleAttainmentResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAnalyzeScheduleAttainmentResponse)
}

func (*AnalyzeDeliveryPerformanceResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAnalyzeDeliveryPerformanceResponse)
}

var (
	samplePricingBelowPeer    = "0.3200"
	samplePricingGrossMargin  = "0.1800"
	sampleRealizedBelowPeer   = "0.2500"
	sampleRealizedGrossMargin = "0.2100"
)

var SampleCustomerPricingFinding = &CustomerPricingFinding{
	ID:             SampleAccountPriceID + ":" + SampleCustomerID,
	Object:         constants.ObjectTypeCustomerPricingFinding,
	AccountPriceID: SampleAccountPriceID,
	Reason:         constants.PricingFindingReasonBelowPeerMedianAndTargetMargin,
	Origin:         constants.AccountPriceOriginDirect,
	UnitPrice: &ComputedRate{
		Object:       constants.ObjectTypeComputedRate,
		Value:        "8.5000",
		DisplayValue: "$8.50 / " + SampleUnitAbbreviation,
	},
	PeerMedianPrice: &ComputedRate{
		Object:       constants.ObjectTypeComputedRate,
		Value:        "12.5000",
		DisplayValue: "$12.50 / " + SampleUnitAbbreviation,
	},
	BelowPeerMedianFraction: &samplePricingBelowPeer,
	GrossMargin:             &samplePricingGrossMargin,
}

var SampleAnalyzeCustomerPricingResponse = &AnalyzeCustomerPricingResponse{
	Object:   constants.ObjectTypeAnalyzeCustomerPricingResponse,
	Findings: NewList([]CustomerPricingFinding{*SampleCustomerPricingFinding}, PageInfo{}),
	Summary: CustomerPricingSummary{
		Object:                 constants.ObjectTypeCustomerPricingSummary,
		PricesAnalyzed:         412,
		BelowPeerMedianCount:   18,
		BelowTargetMarginCount: 7,
		MarginNotAssessedCount: 22,
		Notes:                  []string{},
	},
}

func (*AnalyzeCustomerPricingResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAnalyzeCustomerPricingResponse)
}

var SampleRealizedMarginFinding = &RealizedMarginFinding{
	ID:     SampleCustomerID + ":" + SampleItemID,
	Object: constants.ObjectTypeRealizedMarginFinding,
	Reason: constants.PricingFindingReasonBelowTargetMargin,
	QuantityInvoiced: &ComputedQuantity{
		Object:       constants.ObjectTypeComputedQuantity,
		Value:        "1200",
		DisplayValue: "1,200 " + SampleUnitAbbreviation,
	},
	Revenue: &ComputedQuantity{
		Object:       constants.ObjectTypeComputedQuantity,
		Value:        "16200.00",
		DisplayValue: "16,200.00",
	},
	Cost: &ComputedQuantity{
		Object:       constants.ObjectTypeComputedQuantity,
		Value:        "12800.00",
		DisplayValue: "12,800.00",
	},
	AverageUnitPrice: &ComputedRate{
		Object:       constants.ObjectTypeComputedRate,
		Value:        "13.5000",
		DisplayValue: "$13.50 / " + SampleUnitAbbreviation,
	},
	PeerMedianPrice: &ComputedRate{
		Object:       constants.ObjectTypeComputedRate,
		Value:        "18.0000",
		DisplayValue: "$18.00 / " + SampleUnitAbbreviation,
	},
	LineCount:               14,
	BelowPeerMedianFraction: &sampleRealizedBelowPeer,
	GrossMargin:             &sampleRealizedGrossMargin,
}

var SampleAnalyzeRealizedMarginsResponse = &AnalyzeRealizedMarginsResponse{
	Object:   constants.ObjectTypeAnalyzeRealizedMarginsResponse,
	Findings: NewList([]RealizedMarginFinding{*SampleRealizedMarginFinding}, PageInfo{}),
	Summary: RealizedMarginSummary{
		Object:                 constants.ObjectTypeRealizedMarginSummary,
		LinesAnalyzed:          9840,
		RelationshipsAnalyzed:  1260,
		BelowPeerMedianCount:   31,
		BelowTargetMarginCount: 12,
		MarginNotAssessedCount: 48,
		Notes:                  []string{},
	},
}

func (*AnalyzeRealizedMarginsResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAnalyzeRealizedMarginsResponse)
}
