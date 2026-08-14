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

	analyzeSalesEndpoint := apiendpoint.From(&analyticsep.AnalyzeSalesEndpoint{}).WithService(inner, analyticsSvc)
	analyzeOpenBatchesEndpoint := apiendpoint.From(&analyticsep.AnalyzeOpenBatchesEndpoint{}).WithService(inner, analyticsSvc)
	analyzeProductionCostsEndpoint := apiendpoint.From(&analyticsep.AnalyzeProductionCostsEndpoint{}).WithService(inner, analyticsSvc)
	analyzeDeliveriesEndpoint := apiendpoint.From(&analyticsep.AnalyzeDeliveriesEndpoint{}).WithService(inner, analyticsSvc)
	analyzeManufacturingEndpoint := apiendpoint.From(&analyticsep.AnalyzeManufacturingEndpoint{}).WithService(inner, analyticsSvc)
	analyzeManufacturingBatchEndpoint := apiendpoint.From(&analyticsep.AnalyzeManufacturingBatchEndpoint{}).WithService(inner, analyticsSvc)
	analyzeOrdersEndpoint := apiendpoint.From(&analyticsep.AnalyzeOrdersEndpoint{}).WithService(inner, analyticsSvc)
	analyzeQuarterlyOrdersEndpoint := apiendpoint.From(&analyticsep.AnalyzeQuarterlyOrdersEndpoint{}).WithService(inner, analyticsSvc)
	analyzeMaterialsEndpoint := apiendpoint.From(&analyticsep.AnalyzeMaterialsEndpoint{}).WithService(inner, analyticsSvc)
	analyzeInventoryReceiptsEndpoint := apiendpoint.From(&analyticsep.AnalyzeInventoryReceiptsEndpoint{}).WithService(inner, analyticsSvc)
	analyzeNewCustomersEndpoint := apiendpoint.From(&analyticsep.AnalyzeNewCustomersEndpoint{}).WithService(inner, analyticsSvc)
	analyzeDemandForecastEndpoint := apiendpoint.From(&analyticsep.AnalyzeDemandForecastEndpoint{}).WithService(inner, analyticsSvc)
	analyzeOeeEndpoint := apiendpoint.From(&analyticsep.AnalyzeOeeEndpoint{}).WithService(inner, analyticsSvc)
	analyzeOeeTrendEndpoint := apiendpoint.From(&analyticsep.AnalyzeOeeTrendEndpoint{}).WithService(inner, analyticsSvc)
	analyzeScheduleAttainmentEndpoint := apiendpoint.From(&analyticsep.AnalyzeScheduleAttainmentEndpoint{}).WithService(inner, analyticsSvc)
	analyzeDeliveryPerformanceEndpoint := apiendpoint.From(&analyticsep.AnalyzeDeliveryPerformanceEndpoint{}).WithService(inner, analyticsSvc)
	analyzeWeeksOfSalesEndpoint := apiendpoint.From(&analyticsep.AnalyzeWeeksOfSalesEndpoint{}).WithService(inner, analyticsSvc)
	analyzeCustomerPricingEndpoint := apiendpoint.From(&analyticsep.AnalyzeCustomerPricingEndpoint{}).WithService(inner, analyticsSvc)
	analyzeRealizedMarginsEndpoint := apiendpoint.From(&analyticsep.AnalyzeRealizedMarginsEndpoint{}).WithService(inner, analyticsSvc)

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
		analyzeOeeTrendEndpoint,
		analyzeScheduleAttainmentEndpoint,
		analyzeDeliveryPerformanceEndpoint,
		analyzeWeeksOfSalesEndpoint,
		analyzeCustomerPricingEndpoint,
		analyzeRealizedMarginsEndpoint,
	}

	return &AnalyticsEndpointGroup{inner}
}
