package httpgroup

import (
	"fmt"

	machineep "github.com/augno/api/services/api-gateway/endpoints/machines"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type MachinesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type MachinesEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *MachinesEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("machines endpoint group: core client is required")
	}
	return nil
}

func (*MachinesEndpointGroup) Materialize(config *MachinesEndpointGroupConfig) *MachinesEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	machineSvc := machineep.NewMachineSvc(&machineep.MachineSvcConfig{
		CoreClient: config.CoreClient.Fulfillment,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Machines Management",
		Description:  "List and manage machines.",
		ResourceType: &apiresource.Machine{},
	}

	listMachinesEndpoint := apiendpoint.From(&machineep.ListMachinesEndpoint{}).WithService(inner, machineSvc)
	getMachineEndpoint := apiendpoint.From(&machineep.RetrieveMachineEndpoint{}).WithService(inner, machineSvc)
	createMachineEndpoint := apiendpoint.From(&machineep.CreateMachineEndpoint{}).WithService(inner, machineSvc)
	updateMachineEndpoint := apiendpoint.From(&machineep.UpdateMachineEndpoint{}).WithService(inner, machineSvc)
	deleteMachineEndpoint := apiendpoint.From(&machineep.DeleteMachineEndpoint{}).WithService(inner, machineSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listMachinesEndpoint,
		getMachineEndpoint,
		createMachineEndpoint,
		updateMachineEndpoint,
		deleteMachineEndpoint,
	}

	return &MachinesEndpointGroup{inner}
}
