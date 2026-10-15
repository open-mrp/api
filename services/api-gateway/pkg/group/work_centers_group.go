package httpgroup

import (
	"fmt"

	workcenterep "github.com/open-mrp/api/services/api-gateway/endpoints/work-centers"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

type WorkCentersEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type WorkCentersEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *WorkCentersEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("work centers endpoint group: core client is required")
	}
	return nil
}

func (*WorkCentersEndpointGroup) Materialize(config *WorkCentersEndpointGroupConfig) *WorkCentersEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := workcenterep.NewWorkCenterSvc(&workcenterep.WorkCenterSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Work Centers",
		Description:  "List and manage work centers: where production work is done, with their machines and scanning stations.",
		ResourceType: &apiresource.WorkCenter{},
	}

	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&workcenterep.ListWorkCentersEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&workcenterep.RetrieveWorkCenterEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&workcenterep.CreateWorkCenterEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&workcenterep.UpdateWorkCenterEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&workcenterep.DeleteWorkCenterEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&workcenterep.BulkUpsertWorkCentersEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&workcenterep.ExportWorkCentersEndpoint{}).WithService(inner, svc),
	}

	return &WorkCentersEndpointGroup{inner}
}
