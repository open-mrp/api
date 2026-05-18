package httpgroup

import (
	"fmt"

	customerproductlineaccessep "github.com/augno/api/services/api-gateway/endpoints/customer-product-line-access"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type CustomerProductLineAccessEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type CustomerProductLineAccessEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *CustomerProductLineAccessEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("customer product line access endpoint group: core client is required")
	}
	return nil
}

func (*CustomerProductLineAccessEndpointGroup) Materialize(config *CustomerProductLineAccessEndpointGroupConfig) *CustomerProductLineAccessEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := customerproductlineaccessep.NewCustomerProductLineAccessSvc(&customerproductlineaccessep.CustomerProductLineAccessSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Customer Product Line Access",
		Description:  "Manage product line access for customers.",
		ResourceType: &apiresource.CustomerProductLineAccess{},
	}

	listEndpoint := apiendpoint.From(&customerproductlineaccessep.ListCustomerProductLineAccessEndpoint{}).WithService(inner, svc)
	retrieveEndpoint := apiendpoint.From(&customerproductlineaccessep.RetrieveCustomerProductLineAccessEndpoint{}).WithService(inner, svc)
	createEndpoint := apiendpoint.From(&customerproductlineaccessep.CreateCustomerProductLineAccessEndpoint{}).WithService(inner, svc)
	updateEndpoint := apiendpoint.From(&customerproductlineaccessep.UpdateCustomerProductLineAccessEndpoint{}).WithService(inner, svc)
	deleteEndpoint := apiendpoint.From(&customerproductlineaccessep.DeleteCustomerProductLineAccessEndpoint{}).WithService(inner, svc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		retrieveEndpoint,
		createEndpoint,
		updateEndpoint,
		deleteEndpoint,
	}

	return &CustomerProductLineAccessEndpointGroup{inner}
}
