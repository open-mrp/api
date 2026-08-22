package httpgroup

import (
	"fmt"

	productionschedulesettingsep "github.com/open-mrp/api/services/api-gateway/endpoints/production-schedule-settings"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
)

type ProductionScheduleSettingsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type ProductionScheduleSettingsEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *ProductionScheduleSettingsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("production schedule settings endpoint group: core client is required")
	}
	return nil
}

func (*ProductionScheduleSettingsEndpointGroup) Materialize(config *ProductionScheduleSettingsEndpointGroupConfig) *ProductionScheduleSettingsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := productionschedulesettingsep.NewProductionScheduleSettingsSvc(&productionschedulesettingsep.ProductionScheduleSettingsSvcConfig{
		CoreClient: config.CoreClient.ProductionSchedule,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Production Schedule Settings",
		Description:  "The planning assumptions production schedules are solved against, and the per-resource overrides that mark which machines constrain the plan.",
		ResourceType: &apiresource.ProductionScheduleSettings{},
	}

	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&productionschedulesettingsep.RetrieveProductionScheduleSettingsEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&productionschedulesettingsep.UpdateProductionScheduleSettingsEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&productionschedulesettingsep.ListResourceSettingsEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&productionschedulesettingsep.UpsertResourceSettingEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&productionschedulesettingsep.DeleteResourceSettingEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&productionschedulesettingsep.ListItemSettingsEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&productionschedulesettingsep.RetrieveItemSettingEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&productionschedulesettingsep.UpsertItemSettingEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&productionschedulesettingsep.DeleteItemSettingEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&productionschedulesettingsep.ListFulfillmentRecommendationsEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&productionschedulesettingsep.ApplyFulfillmentRecommendationsEndpoint{}).WithService(inner, svc),
	}

	return &ProductionScheduleSettingsEndpointGroup{inner}
}
