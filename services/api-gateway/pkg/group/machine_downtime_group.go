package httpgroup

import (
	"fmt"

	machinedowntimeep "github.com/augno/api/services/api-gateway/endpoints/machine-downtime"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type MachineDowntimeEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type MachineDowntimeEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *MachineDowntimeEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("machine downtime endpoint group: core client is required")
	}
	return nil
}

func (*MachineDowntimeEndpointGroup) Materialize(config *MachineDowntimeEndpointGroupConfig) *MachineDowntimeEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := machinedowntimeep.NewMachineDowntimeSvc(&machinedowntimeep.MachineDowntimeSvcConfig{
		CoreClient: config.CoreClient.MachineDowntime,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Machine Downtime",
		Description:  "Log and review machine stoppages. Downtime is the source of OEE availability and changeover time.",
		ResourceType: &apiresource.MachineDowntimeEvent{},
	}

	listReasonsEndpoint := apiendpoint.From(&machinedowntimeep.ListMachineDowntimeReasonsEndpoint{}).WithService(inner, svc)
	listEndpoint := apiendpoint.From(&machinedowntimeep.ListMachineDowntimeEventsEndpoint{}).WithService(inner, svc)
	retrieveEndpoint := apiendpoint.From(&machinedowntimeep.RetrieveMachineDowntimeEventEndpoint{}).WithService(inner, svc)
	createEndpoint := apiendpoint.From(&machinedowntimeep.CreateMachineDowntimeEventEndpoint{}).WithService(inner, svc)
	updateEndpoint := apiendpoint.From(&machinedowntimeep.UpdateMachineDowntimeEventEndpoint{}).WithService(inner, svc)
	deleteEndpoint := apiendpoint.From(&machinedowntimeep.DeleteMachineDowntimeEventEndpoint{}).WithService(inner, svc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listReasonsEndpoint,
		listEndpoint,
		retrieveEndpoint,
		createEndpoint,
		updateEndpoint,
		deleteEndpoint,
	}

	return &MachineDowntimeEndpointGroup{inner}
}
