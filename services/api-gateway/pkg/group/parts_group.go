package httpgroup

import (
	"fmt"

	partep "github.com/augno/api/services/api-gateway/endpoints/parts"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type PartsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type PartsEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *PartsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("parts endpoint group: core client is required")
	}
	return nil
}

func (*PartsEndpointGroup) Materialize(config *PartsEndpointGroupConfig) *PartsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	partSvc := partep.NewPartSvc(&partep.PartSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Parts Management",
		Description:  "List and manage parts.",
		ResourceType: &apiresource.Part{},
	}

	createPartEndpoint := (&partep.CreatePartEndpoint{}).Materialize().WithService(inner, partSvc)
	getPartEndpoint := (&partep.RetrievePartEndpoint{}).Materialize().WithService(inner, partSvc)
	listPartsEndpoint := (&partep.ListPartsEndpoint{}).Materialize().WithService(inner, partSvc)
	updatePartEndpoint := (&partep.UpdatePartEndpoint{}).Materialize().WithService(inner, partSvc)
	deletePartEndpoint := (&partep.DeletePartEndpoint{}).Materialize().WithService(inner, partSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listPartsEndpoint,
		getPartEndpoint,
		createPartEndpoint,
		updatePartEndpoint,
		deletePartEndpoint,
	}

	return &PartsEndpointGroup{inner}
}
