package httpgroup

import (
	"fmt"

	machinestatusep "github.com/augno/api/services/api-gateway/endpoints/machine-status"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type MachineStatusEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type MachineStatusEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *MachineStatusEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("machine status endpoint group: core client is required")
	}
	return nil
}

func (*MachineStatusEndpointGroup) Materialize(config *MachineStatusEndpointGroupConfig) *MachineStatusEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := machinestatusep.NewMachineStatusSvc(&machinestatusep.MachineStatusSvcConfig{
		CoreClient: config.CoreClient.Fulfillment,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Machine Status",
		Description:  "What every machine is running right now, how much is left on it, what is queued behind that, and whether it is down.",
		ResourceType: &apiresource.MachineStatus{},
	}

	listEndpoint := apiendpoint.From(&machinestatusep.ListMachineStatusEndpoint{}).WithService(inner, svc)

	inner.Endpoints = []apiendpoint.APIEndpointer{listEndpoint}

	return &MachineStatusEndpointGroup{inner}
}
