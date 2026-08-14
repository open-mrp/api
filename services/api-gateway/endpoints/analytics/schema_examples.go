package analyticsep

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/field"
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
	s := apiresource.SampleAnalyticsPeriodStart
	e := apiresource.SampleAnalyticsPeriodEnd
	return apiexample.ValidateAndMarshalToMap(&AnalyzeProductionCostsRequest{
		StartDate:      &s,
		EndDate:        &e,
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
