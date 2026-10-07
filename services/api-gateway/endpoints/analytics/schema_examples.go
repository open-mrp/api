package analyticsep

import (
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/field"
	"github.com/open-mrp/api/shared/pagination"
)

func (*AnalyzeSalesRequest) SchemaExample() any {
	q := "6061"
	return apiexample.ValidateAndMarshalToMap(&AnalyzeSalesRequest{
		StartDate:        apiresource.SampleAnalyticsPeriodStart,
		EndDate:          apiresource.SampleAnalyticsPeriodEnd,
		ProductLineIDs:   []string{apiresource.SampleProductLineID},
		CustomerIDs:      []string{apiresource.SampleCustomerID},
		SalesRepIDs:      []string{apiresource.SampleAccountUserID},
		CustomerGroupIDs: []string{apiresource.SampleAccountGroupID},
		Query:            &q,
	})
}

func (*AnalyzeDeliveriesRequest) SchemaExample() any {
	td := int64(7)
	ov := true
	return apiexample.ValidateAndMarshalToMap(&AnalyzeDeliveriesRequest{
		StartDate:              apiresource.SampleAnalyticsPeriodStart,
		EndDate:                apiresource.SampleAnalyticsPeriodEnd,
		ProductLineIDs:         []string{apiresource.SampleProductLineID},
		CustomerIDs:            []string{apiresource.SampleCustomerID},
		CustomerGroupIDs:       []string{apiresource.SampleAccountGroupID},
		SalesRepIDs:            []string{apiresource.SampleAccountUserID},
		TargetDeliveryTimeDays: &td,
		OverridePromisedDates:  &ov,
	})
}

func (*AnalyzeDemandForecastRequest) SchemaExample() any {
	hm := int64(6)
	fm := int64(3)
	return apiexample.ValidateAndMarshalToMap(&AnalyzeDemandForecastRequest{
		ProductLineIDs: []string{apiresource.SampleProductLineID},
		ItemIDs:        []string{apiresource.SampleItemID},
		HistoryMonths:  &hm,
		ForecastMonths: &fm,
	})
}

func (*AnalyzeInventoryReceiptsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&AnalyzeInventoryReceiptsRequest{
		ItemIDs:     []string{apiresource.SampleItemID},
		LocationIDs: []string{apiresource.SampleLocationID},
		LotIDs:      []string{apiresource.SampleLotID},
	})
}

func (*AnalyzeManufacturingRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&AnalyzeManufacturingRequest{
		StartDate: apiresource.SampleAnalyticsPeriodStart,
		EndDate:   apiresource.SampleAnalyticsPeriodEnd,
		Type:      "production",
	})
}

func (*AnalyzeManufacturingBatchRequest) SchemaExample() any {
	cs := apiresource.SampleAnalyticsPeriodStart.AddDate(0, -1, 0)
	ce := apiresource.SampleAnalyticsPeriodEnd.AddDate(0, -1, 0)
	return apiexample.ValidateAndMarshalToMap(&AnalyzeManufacturingBatchRequest{
		StartDate:           apiresource.SampleAnalyticsPeriodStart,
		EndDate:             apiresource.SampleAnalyticsPeriodEnd,
		ComparisonStartDate: cs,
		ComparisonEndDate:   ce,
		CustomerIDs:         []string{apiresource.SampleCustomerID},
		ProductLineIDs:      []string{apiresource.SampleProductLineID},
		CustomerGroupIDs:    []string{apiresource.SampleAccountGroupID},
		ItemIDs:             []string{apiresource.SampleItemID},
	})
}

func (*AnalyzeMaterialsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&AnalyzeMaterialsRequest{
		SalesOrderIDs: []string{apiresource.SampleSalesOrderID},
		SupplierIDs:   []string{apiresource.SampleSupplierID},
	})
}

func (*AnalyzeNewCustomersRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&AnalyzeNewCustomersRequest{
		StartDate:        apiresource.SampleAnalyticsPeriodStart,
		EndDate:          apiresource.SampleAnalyticsPeriodEnd,
		CustomerGroupIDs: []string{apiresource.SampleAccountGroupID},
		SalesRepIDs:      []string{apiresource.SampleAccountUserID},
	})
}

func (*AnalyzeOeeRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&AnalyzeOeeRequest{
		StartDate:     apiresource.SampleAnalyticsPeriodStart,
		EndDate:       apiresource.SampleAnalyticsPeriodEnd,
		DepartmentIDs: []string{apiresource.SampleDepartmentID},
	})
}

func (*AnalyzeOeeTrendRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&AnalyzeOeeTrendRequest{
		StartDate:     apiresource.SampleAnalyticsPeriodStart,
		EndDate:       apiresource.SampleAnalyticsPeriodEnd,
		DepartmentIDs: []string{apiresource.SampleDepartmentID},
	})
}

func (*AnalyzeOpenBatchesRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&AnalyzeOpenBatchesRequest{
		ItemIDs:        []string{apiresource.SampleItemID},
		ProductLineIDs: []string{apiresource.SampleProductLineID},
	})
}

func (*AnalyzeOrdersRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&AnalyzeOrdersRequest{
		ProductLineIDs:   []string{apiresource.SampleProductLineID},
		CustomerIDs:      []string{apiresource.SampleCustomerID},
		SalesRepIDs:      []string{apiresource.SampleAccountUserID},
		CustomerGroupIDs: []string{apiresource.SampleAccountGroupID},
	})
}

func (*AnalyzeProductionCostsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&AnalyzeProductionCostsRequest{
		StartDate:      apiresource.SampleAnalyticsPeriodStart,
		EndDate:        apiresource.SampleAnalyticsPeriodEnd,
		ItemIDs:        []string{apiresource.SampleItemID},
		ProductLineIDs: []string{apiresource.SampleProductLineID},
		DepartmentIDs:  []string{apiresource.SampleDepartmentID},
		CategoryIDs:    []string{apiresource.SampleItemCategoryID},
	})
}

func (*AnalyzeQuarterlyOrdersRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&AnalyzeQuarterlyOrdersRequest{
		SalesRepIDs:      []string{apiresource.SampleAccountUserID},
		ItemIDs:          []string{apiresource.SampleItemID},
		ProductLineIDs:   []string{apiresource.SampleProductLineID},
		CustomerIDs:      []string{apiresource.SampleCustomerID},
		CustomerGroupIDs: []string{apiresource.SampleAccountGroupID},
		YearsBack:        field.Some(int32(5)),
	})
}

func (*AnalyzeWeeksOfSalesRequest) SchemaExample() any {
	return map[string]any{"period_in_weeks": int64(4)}
}

func (*AnalyzeScheduleAttainmentRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&AnalyzeScheduleAttainmentRequest{
		StartDate: apiresource.SampleAnalyticsPeriodStart,
		EndDate:   apiresource.SampleAnalyticsPeriodEnd,
		GroupBy:   field.Some(constants.AttainmentGroupByWeek),
	})
}

func (*AnalyzeDeliveryPerformanceRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&AnalyzeDeliveryPerformanceRequest{
		StartDate:   apiresource.SampleAnalyticsPeriodStart,
		EndDate:     apiresource.SampleAnalyticsPeriodEnd,
		Granularity: field.Some(constants.DeliveryGranularityWeek),
	})
}

func (*AnalyzeCustomerPricingRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&AnalyzeCustomerPricingRequest{
		CustomerIDs:       []string{apiresource.SampleCustomerID},
		CustomerGroupIDs:  []string{apiresource.SampleAccountGroupID},
		TargetGrossMargin: field.Some("0.30"),
		OutlierTolerance:  field.Some("0.15"),
	})
}

func (*AnalyzeRealizedMarginsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&AnalyzeRealizedMarginsRequest{
		StartDate:         apiresource.SampleAnalyticsPeriodStart,
		EndDate:           apiresource.SampleAnalyticsPeriodEnd,
		CustomerIDs:       []string{apiresource.SampleCustomerID},
		CustomerGroupIDs:  []string{apiresource.SampleAccountGroupID},
		ProductLineIDs:    []string{apiresource.SampleProductLineID},
		TargetGrossMargin: field.Some("0.30"),
		OutlierTolerance:  field.Some("0.15"),
	})
}

func sampleSalesReportFilters() SalesReportFilters {
	return SalesReportFilters{
		CustomerIDs:      []string{apiresource.SampleCustomerID},
		CustomerGroupIDs: []string{apiresource.SampleAccountGroupID},
		ProductLineIDs:   []string{apiresource.SampleProductLineID},
		SalesRepIDs:      []string{apiresource.SampleAccountUserID},
		ItemIDs:          []string{apiresource.SampleItemID},
	}
}

func sampleSalesComparisonPeriod() SalesComparisonPeriod {
	return SalesComparisonPeriod{
		ComparisonStartDate: field.Some(apiresource.SampleAnalyticsPeriodStart.AddDate(-1, 0, 0)),
		ComparisonEndDate:   field.Some(apiresource.SampleAnalyticsPeriodEnd.AddDate(-1, 0, 0)),
	}
}

func (*AnalyzeSalesSummaryRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&AnalyzeSalesSummaryRequest{
		StartDate:             apiresource.SampleAnalyticsPeriodStart,
		EndDate:               apiresource.SampleAnalyticsPeriodEnd,
		SalesComparisonPeriod: sampleSalesComparisonPeriod(),
		SalesReportFilters:    sampleSalesReportFilters(),
		TZOffsetMinutes:       field.Some(int32(-300)),
	})
}

func (*AnalyzeSalesBreakdownRequest) SchemaExample() any {
	ex := apiexample.ValidateAndMarshalToMap(&AnalyzeSalesBreakdownRequest{
		GroupBy:               constants.SalesBreakdownGroupByCustomer,
		StartDate:             apiresource.SampleAnalyticsPeriodStart,
		EndDate:               apiresource.SampleAnalyticsPeriodEnd,
		SalesComparisonPeriod: sampleSalesComparisonPeriod(),
		SalesReportFilters:    sampleSalesReportFilters(),
		Limit:                 10,
	})
	// The cursor is a query parameter, documented from this map by its query key.
	ex["cursor"] = pagination.EncodeDocumentationValueCursor("16200.5", apiresource.SampleCustomerID)
	return ex
}

func (*AnalyzeSalesInvoicesRequest) SchemaExample() any {
	ex := apiexample.ValidateAndMarshalToMap(&AnalyzeSalesInvoicesRequest{
		StartDate:          apiresource.SampleAnalyticsPeriodStart,
		EndDate:            apiresource.SampleAnalyticsPeriodEnd,
		SalesReportFilters: sampleSalesReportFilters(),
		Limit:              10,
	})
	// The cursor is a query parameter, documented from this map by its query key.
	ex["cursor"] = pagination.EncodeDocumentationStringCursor(apiresource.SampleAnalyticsPeriodStart, apiresource.SampleInvoiceID)
	return ex
}

func (*ListNewCustomersRequest) SchemaExample() any {
	ex := apiexample.ValidateAndMarshalToMap(&ListNewCustomersRequest{
		StartDate:        apiresource.SampleAnalyticsPeriodStart,
		EndDate:          apiresource.SampleAnalyticsPeriodEnd,
		CustomerGroupIDs: []string{apiresource.SampleAccountGroupID},
		Limit:            100,
	})
	// The cursor is a query parameter, documented from this map by its query key.
	ex["cursor"] = pagination.EncodeDocumentationStringCursor(apiresource.SampleAnalyticsPeriodStart, apiresource.SampleCustomerID)
	return ex
}

func (*ListSalesLinesRequest) SchemaExample() any {
	ex := apiexample.ValidateAndMarshalToMap(&ListSalesLinesRequest{
		StartDate:          field.Some(apiresource.SampleAnalyticsPeriodStart),
		EndDate:            field.Some(apiresource.SampleAnalyticsPeriodEnd),
		SalesReportFilters: sampleSalesReportFilters(),
		Limit:              50,
	})
	// The cursor is a query parameter, documented from this map by its query key.
	ex["cursor"] = pagination.EncodeDocumentationStringCursor(apiresource.SampleAnalyticsPeriodStart, apiresource.SampleInvoiceID)
	return ex
}

func (*ExportSalesLinesRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&ExportSalesLinesRequest{
		StartDate:          field.Some(apiresource.SampleAnalyticsPeriodStart),
		EndDate:            field.Some(apiresource.SampleAnalyticsPeriodEnd),
		SalesReportFilters: sampleSalesReportFilters(),
	})
}

func sampleOpenOrderFilters() OpenOrderFilters {
	return OpenOrderFilters{
		CustomerIDs:      []string{apiresource.SampleCustomerID},
		CustomerGroupIDs: []string{apiresource.SampleAccountGroupID},
		SalesRepIDs:      []string{apiresource.SampleAccountUserID},
		ProductLineIDs:   []string{apiresource.SampleProductLineID},
		ItemIDs:          []string{apiresource.SampleItemID},
	}
}

func (*AnalyzeOpenOrdersSummaryRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&AnalyzeOpenOrdersSummaryRequest{OpenOrderFilters: sampleOpenOrderFilters()})
}

func (*AnalyzeOpenOrdersBreakdownRequest) SchemaExample() any {
	ex := apiexample.ValidateAndMarshalToMap(&AnalyzeOpenOrdersBreakdownRequest{OpenOrderFilters: sampleOpenOrderFilters(), Limit: 10})
	// The cursor is a query parameter, documented from this map by its query key.
	ex["cursor"] = pagination.EncodeDocumentationValueCursor("840", apiresource.SampleItemID)
	return ex
}

func (*ListOpenOrdersRequest) SchemaExample() any {
	ex := apiexample.ValidateAndMarshalToMap(&ListOpenOrdersRequest{OpenOrderFilters: sampleOpenOrderFilters(), Limit: 10})
	// The cursor is a query parameter, documented from this map by its query key.
	ex["cursor"] = pagination.EncodeDocumentationStringCursor(apiresource.SampleAnalyticsPeriodStart, apiresource.SampleSalesOrderID)
	return ex
}

func (*ListOpenOrderLinesRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&ListOpenOrderLinesRequest{SalesOrderID: apiresource.SampleSalesOrderID})
}

func (*ExportOpenOrderLinesRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(&ExportOpenOrderLinesRequest{OpenOrderFilters: sampleOpenOrderFilters()})
}
