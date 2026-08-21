package httpgroup

import (
	"fmt"

	productionscheduleep "github.com/augno/api/services/api-gateway/endpoints/production-schedules"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type ProductionSchedulesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type ProductionSchedulesEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *ProductionSchedulesEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("production schedules endpoint group: core client is required")
	}
	return nil
}

func (*ProductionSchedulesEndpointGroup) Materialize(config *ProductionSchedulesEndpointGroupConfig) *ProductionSchedulesEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := productionscheduleep.NewProductionScheduleSvc(&productionscheduleep.ProductionScheduleSvcConfig{
		CoreClient: config.CoreClient.ProductionSchedule,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Production Schedules",
		Description:  "Generate and review machine-level production schedules.",
		ResourceType: &apiresource.ProductionSchedule{},
	}

	previewEndpoint := apiendpoint.From(&productionscheduleep.PreviewProductionScheduleEndpoint{}).WithService(inner, svc)
	generateEndpoint := apiendpoint.From(&productionscheduleep.GenerateProductionScheduleEndpoint{}).WithService(inner, svc)
	previewRegenerateEndpoint := apiendpoint.From(&productionscheduleep.PreviewRegenerateProductionScheduleEndpoint{}).WithService(inner, svc)
	regenerateEndpoint := apiendpoint.From(&productionscheduleep.RegenerateProductionScheduleEndpoint{}).WithService(inner, svc)
	listEndpoint := apiendpoint.From(&productionscheduleep.ListProductionSchedulesEndpoint{}).WithService(inner, svc)
	// Registered before the {id} route so "current" is not swallowed as an ID.
	currentEndpoint := apiendpoint.From(&productionscheduleep.RetrieveCurrentProductionScheduleEndpoint{}).WithService(inner, svc)
	retrieveEndpoint := apiendpoint.From(&productionscheduleep.RetrieveProductionScheduleEndpoint{}).WithService(inner, svc)
	linesEndpoint := apiendpoint.From(&productionscheduleep.ListProductionScheduleLinesEndpoint{}).WithService(inner, svc)
	itemPoliciesEndpoint := apiendpoint.From(&productionscheduleep.ListProductionScheduleItemPoliciesEndpoint{}).WithService(inner, svc)
	finishedPoliciesEndpoint := apiendpoint.From(&productionscheduleep.ListProductionScheduleFinishedPoliciesEndpoint{}).WithService(inner, svc)
	finishingLinesEndpoint := apiendpoint.From(&productionscheduleep.ListProductionScheduleFinishingLinesEndpoint{}).WithService(inner, svc)
	derivedLinesEndpoint := apiendpoint.From(&productionscheduleep.ListProductionScheduleDerivedLinesEndpoint{}).WithService(inner, svc)
	atRiskOrdersEndpoint := apiendpoint.From(&productionscheduleep.ListAtRiskOrdersEndpoint{}).WithService(inner, svc)
	deviationTypesEndpoint := apiendpoint.From(&productionscheduleep.ListScheduleDeviationTypesEndpoint{}).WithService(inner, svc)
	deviationsEndpoint := apiendpoint.From(&productionscheduleep.ListProductionScheduleDeviationsEndpoint{}).WithService(inner, svc)
	createLineEndpoint := apiendpoint.From(&productionscheduleep.CreateProductionScheduleLineEndpoint{}).WithService(inner, svc)
	updateLineEndpoint := apiendpoint.From(&productionscheduleep.UpdateProductionScheduleLineEndpoint{}).WithService(inner, svc)
	deleteLineEndpoint := apiendpoint.From(&productionscheduleep.DeleteProductionScheduleLineEndpoint{}).WithService(inner, svc)
	publishEndpoint := apiendpoint.From(&productionscheduleep.PublishProductionScheduleEndpoint{}).WithService(inner, svc)
	previewReleaseWeekEndpoint := apiendpoint.From(&productionscheduleep.PreviewReleaseProductionScheduleWeekEndpoint{}).WithService(inner, svc)
	releaseWeekEndpoint := apiendpoint.From(&productionscheduleep.ReleaseProductionScheduleWeekEndpoint{}).WithService(inner, svc)
	archiveEndpoint := apiendpoint.From(&productionscheduleep.ArchiveProductionScheduleEndpoint{}).WithService(inner, svc)
	deleteEndpoint := apiendpoint.From(&productionscheduleep.DeleteProductionScheduleEndpoint{}).WithService(inner, svc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		generateEndpoint,
		currentEndpoint,
		retrieveEndpoint,
		linesEndpoint,
		itemPoliciesEndpoint,
		finishedPoliciesEndpoint,
		finishingLinesEndpoint,
		previewEndpoint,
		previewRegenerateEndpoint,
		regenerateEndpoint,
		derivedLinesEndpoint,
		atRiskOrdersEndpoint,
		deviationTypesEndpoint,
		deviationsEndpoint,
		createLineEndpoint,
		updateLineEndpoint,
		deleteLineEndpoint,
		publishEndpoint,
		previewReleaseWeekEndpoint,
		releaseWeekEndpoint,
		archiveEndpoint,
		deleteEndpoint,
	}

	return &ProductionSchedulesEndpointGroup{inner}
}
