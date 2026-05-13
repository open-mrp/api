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
		Title:        "Materials Management",
		Description:  "List and manage materials.",
		ResourceType: &apiresource.Material{},
	}

	listMaterialsEndpoint := (&materialep.ListMaterialsEndpoint{}).Materialize().WithService(inner, materialSvc)
	getMaterialEndpoint := (&materialep.RetrieveMaterialEndpoint{}).Materialize().WithService(inner, materialSvc)
	createMaterialEndpoint := (&materialep.CreateMaterialEndpoint{}).Materialize().WithService(inner, materialSvc)
	updateMaterialEndpoint := (&materialep.UpdateMaterialEndpoint{}).Materialize().WithService(inner, materialSvc)
	deleteMaterialEndpoint := (&materialep.DeleteMaterialEndpoint{}).Materialize().WithService(inner, materialSvc)
	exportMaterialsEndpoint := (&materialep.ExportMaterialsEndpoint{}).Materialize().WithService(inner, materialSvc)

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
