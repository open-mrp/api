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
	// CoreClient (required) is the core-service gRPC client.
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
		Title:        "Parts",
		Description:  "List and manage parts.",
		ResourceType: &apiresource.Part{},
	}

	createPartEndpoint := apiendpoint.From(&partep.CreatePartEndpoint{}).WithService(inner, partSvc)
	getPartEndpoint := apiendpoint.From(&partep.RetrievePartEndpoint{}).WithService(inner, partSvc)
	listPartsEndpoint := apiendpoint.From(&partep.ListPartsEndpoint{}).WithService(inner, partSvc)
	updatePartEndpoint := apiendpoint.From(&partep.UpdatePartEndpoint{}).WithService(inner, partSvc)
	deletePartEndpoint := apiendpoint.From(&partep.DeletePartEndpoint{}).WithService(inner, partSvc)
	exportPartsEndpoint := apiendpoint.From(&partep.ExportPartsEndpoint{}).WithService(inner, partSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listPartsEndpoint,
		getPartEndpoint,
		createPartEndpoint,
		updatePartEndpoint,
		deletePartEndpoint,
		exportPartsEndpoint,
	}

	return &PartsEndpointGroup{inner}
}
