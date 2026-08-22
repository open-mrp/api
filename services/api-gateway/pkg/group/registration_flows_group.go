package httpgroup

import (
	"fmt"

	registrationflowep "github.com/open-mrp/api/services/api-gateway/endpoints/registration-flows"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
)

type RegistrationFlowsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type RegistrationFlowsEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
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
		Title:        "Registration Flows",
		Description:  "List and manage registration flows.",
		ResourceType: &apiresource.RegistrationFlow{},
	}

	listRegistrationFlowsEndpoint := apiendpoint.From(&registrationflowep.ListRegistrationFlowsEndpoint{}).WithService(inner, registrationFlowSvc)
	getRegistrationFlowEndpoint := apiendpoint.From(&registrationflowep.RetrieveRegistrationFlowEndpoint{}).WithService(inner, registrationFlowSvc)
	createRegistrationFlowEndpoint := apiendpoint.From(&registrationflowep.CreateRegistrationFlowEndpoint{}).WithService(inner, registrationFlowSvc)
	updateRegistrationFlowEndpoint := apiendpoint.From(&registrationflowep.UpdateRegistrationFlowEndpoint{}).WithService(inner, registrationFlowSvc)
	deleteRegistrationFlowEndpoint := apiendpoint.From(&registrationflowep.DeleteRegistrationFlowEndpoint{}).WithService(inner, registrationFlowSvc)
	getRegistrationFlowBySlugEndpoint := apiendpoint.From(&registrationflowep.RetrieveRegistrationFlowBySlugEndpoint{}).WithService(inner, registrationFlowSvc)
	registerCustomerEndpoint := apiendpoint.From(&registrationflowep.RegisterCustomerEndpoint{}).WithService(inner, registrationFlowSvc)

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
