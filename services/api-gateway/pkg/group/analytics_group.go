package httpgroup

import (
	"fmt"

	analyticsep "github.com/augno/api/services/api-gateway/endpoints/analytics"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type AnalyticsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type AnalyticsEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *AnalyticsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("analytics endpoint group: core client is required")
	}
	return nil
}

func (*AnalyticsEndpointGroup) Materialize(config *AnalyticsEndpointGroupConfig) *AnalyticsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	analyticsSvc := analyticsep.NewAnalyticsSvc(&analyticsep.AnalyticsSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Analytics",
		Description:  "Analyze sales, orders, manufacturing, materials, and other business metrics.",
		ResourceType: &apiresource.AnalyzeSalesResponse{},
	}

	analyzeSalesEndpoint := (&analyticsep.AnalyzeSalesEndpoint{}).Materialize().WithService(inner, analyticsSvc)
	analyzeOpenBatchesEndpoint := (&analyticsep.AnalyzeOpenBatchesEndpoint{}).Materialize().WithService(inner, analyticsSvc)
	analyzeProductionCostsEndpoint := (&analyticsep.AnalyzeProductionCostsEndpoint{}).Materialize().WithService(inner, analyticsSvc)
	analyzeDeliveriesEndpoint := (&analyticsep.AnalyzeDeliveriesEndpoint{}).Materialize().WithService(inner, analyticsSvc)
	analyzeManufacturingEndpoint := (&analyticsep.AnalyzeManufacturingEndpoint{}).Materialize().WithService(inner, analyticsSvc)
	analyzeManufacturingBatchEndpoint := (&analyticsep.AnalyzeManufacturingBatchEndpoint{}).Materialize().WithService(inner, analyticsSvc)
	analyzeOrdersEndpoint := (&analyticsep.AnalyzeOrdersEndpoint{}).Materialize().WithService(inner, analyticsSvc)
	analyzeQuarterlyOrdersEndpoint := (&analyticsep.AnalyzeQuarterlyOrdersEndpoint{}).Materialize().WithService(inner, analyticsSvc)
	analyzeMaterialsEndpoint := (&analyticsep.AnalyzeMaterialsEndpoint{}).Materialize().WithService(inner, analyticsSvc)
	analyzeInventoryReceiptsEndpoint := (&analyticsep.AnalyzeInventoryReceiptsEndpoint{}).Materialize().WithService(inner, analyticsSvc)
	analyzeNewCustomersEndpoint := (&analyticsep.AnalyzeNewCustomersEndpoint{}).Materialize().WithService(inner, analyticsSvc)
	analyzeDemandForecastEndpoint := (&analyticsep.AnalyzeDemandForecastEndpoint{}).Materialize().WithService(inner, analyticsSvc)
	analyzeOeeEndpoint := (&analyticsep.AnalyzeOeeEndpoint{}).Materialize().WithService(inner, analyticsSvc)
	analyzeWeeksOfSalesEndpoint := (&analyticsep.AnalyzeWeeksOfSalesEndpoint{}).Materialize().WithService(inner, analyticsSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		analyzeSalesEndpoint,
		analyzeOpenBatchesEndpoint,
		analyzeProductionCostsEndpoint,
		analyzeDeliveriesEndpoint,
		analyzeManufacturingEndpoint,
		analyzeManufacturingBatchEndpoint,
		analyzeOrdersEndpoint,
		analyzeQuarterlyOrdersEndpoint,
		analyzeMaterialsEndpoint,
		analyzeInventoryReceiptsEndpoint,
		analyzeNewCustomersEndpoint,
		analyzeDemandForecastEndpoint,
		analyzeOeeEndpoint,
		analyzeWeeksOfSalesEndpoint,
	}

	return &AnalyticsEndpointGroup{inner}
}
