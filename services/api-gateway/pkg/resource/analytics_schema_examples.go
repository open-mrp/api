package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
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
		OrderID:           SampleSalesOrderDetailID,
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

func (*AnalyzeDemandForecastResponse) SchemaExample() any {
	d := analyticsExampleTime()
	pt := DemandForecastPoint{Date: d, Demand: 48}
	fp := DemandForecastForecastPoint{Date: d, Forecast: 50, LowerBound: 45, UpperBound: 55}
	rv := RevenueForecastPoint{Date: d, Revenue: 960}
	pl := SampleProductLineID
	return apiexample.ValidateAndMarshalToMap(&AnalyzeDemandForecastResponse{
		Object: constants.ObjectTypeList,
		Data: []DemandForecastRow{
			{
				ItemID:              SampleItemID,
				ProductLineID:       &pl,
				ProductSku:          SampleItemSKU,
				Unit:                SampleUnitAbbreviation,
				Currency:            "USD",
				History:             []DemandForecastPoint{pt},
				Forecast:            []DemandForecastForecastPoint{fp},
				RevenueHistory:      []RevenueForecastPoint{rv},
				RevenueForecast:     []DemandForecastForecastPoint{fp},
				SalesHistory:        []RevenueForecastPoint{rv},
				SalesForecast:       []DemandForecastForecastPoint{fp},
				CurrentMonthDemand:  120,
				CurrentMonthRevenue: 2400,
				CurrentMonthSales:   120,
			},
		},
		CurrentMonthFraction: 0.35,
	})
}

func (*AnalyzeOeeResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&AnalyzeOeeResponse{
		Object: constants.ObjectTypeAnalyzeOeeResponse,
		Departments: []OeeDepartment{
			{
				DepartmentID:          SampleDepartmentID,
				DepartmentName:        SampleDepartmentName,
				GoodUnits:             980,
				WasteUnits:            20,
				SecondsUnits:          5,
				EstimatedRuntimeHours: 40,
			},
		},
	})
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
