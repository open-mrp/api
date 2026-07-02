package httpgroup

import (
	"fmt"

	materialep "github.com/augno/api/services/api-gateway/endpoints/materials"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type MaterialsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type MaterialsEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *MaterialsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("materials endpoint group: core client is required")
	}
	return nil
}

func (*MaterialsEndpointGroup) Materialize(config *MaterialsEndpointGroupConfig) *MaterialsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	materialSvc := materialep.NewMaterialSvc(&materialep.MaterialSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Materials",
		Description:  "List and manage materials.",
		ResourceType: &apiresource.Material{},
	}

	listMaterialsEndpoint := apiendpoint.From(&materialep.ListMaterialsEndpoint{}).WithService(inner, materialSvc)
	getMaterialEndpoint := apiendpoint.From(&materialep.RetrieveMaterialEndpoint{}).WithService(inner, materialSvc)
	createMaterialEndpoint := apiendpoint.From(&materialep.CreateMaterialEndpoint{}).WithService(inner, materialSvc)
	updateMaterialEndpoint := apiendpoint.From(&materialep.UpdateMaterialEndpoint{}).WithService(inner, materialSvc)
	deleteMaterialEndpoint := apiendpoint.From(&materialep.DeleteMaterialEndpoint{}).WithService(inner, materialSvc)
	exportMaterialsEndpoint := apiendpoint.From(&materialep.ExportMaterialsEndpoint{}).WithService(inner, materialSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listMaterialsEndpoint,
		getMaterialEndpoint,
		createMaterialEndpoint,
		updateMaterialEndpoint,
		deleteMaterialEndpoint,
		exportMaterialsEndpoint,
	}

	return &MaterialsEndpointGroup{inner}
}
