package httpgroup

import (
	"fmt"

	registrationflowep "github.com/augno/api/services/api-gateway/endpoints/registration-flows"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type RegistrationFlowsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type RegistrationFlowsEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *RegistrationFlowsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("registration flows endpoint group: core client is required")
	}
	return nil
}

func (*RegistrationFlowsEndpointGroup) Materialize(config *RegistrationFlowsEndpointGroupConfig) *RegistrationFlowsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	registrationFlowSvc := registrationflowep.NewRegistrationFlowSvc(&registrationflowep.RegistrationFlowSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Registration Flows Management",
		Description:  "List and manage registration flows.",
		ResourceType: &apiresource.RegistrationFlow{},
	}

	listRegistrationFlowsEndpoint := (&registrationflowep.ListRegistrationFlowsEndpoint{}).Materialize().WithService(inner, registrationFlowSvc)
	getRegistrationFlowEndpoint := (&registrationflowep.RetrieveRegistrationFlowEndpoint{}).Materialize().WithService(inner, registrationFlowSvc)
	createRegistrationFlowEndpoint := (&registrationflowep.CreateRegistrationFlowEndpoint{}).Materialize().WithService(inner, registrationFlowSvc)
	updateRegistrationFlowEndpoint := (&registrationflowep.UpdateRegistrationFlowEndpoint{}).Materialize().WithService(inner, registrationFlowSvc)
	deleteRegistrationFlowEndpoint := (&registrationflowep.DeleteRegistrationFlowEndpoint{}).Materialize().WithService(inner, registrationFlowSvc)
	getRegistrationFlowBySlugEndpoint := (&registrationflowep.RetrieveRegistrationFlowBySlugEndpoint{}).Materialize().WithService(inner, registrationFlowSvc)
	registerCustomerEndpoint := (&registrationflowep.RegisterCustomerEndpoint{}).Materialize().WithService(inner, registrationFlowSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listRegistrationFlowsEndpoint,
		getRegistrationFlowEndpoint,
		createRegistrationFlowEndpoint,
		updateRegistrationFlowEndpoint,
		deleteRegistrationFlowEndpoint,
		getRegistrationFlowBySlugEndpoint,
		registerCustomerEndpoint,
	}

	return &RegistrationFlowsEndpointGroup{inner}
}
