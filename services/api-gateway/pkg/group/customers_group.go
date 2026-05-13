package httpgroup

import (
	"fmt"

	customerep "github.com/augno/api/services/api-gateway/endpoints/customers"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type CustomersEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type CustomersEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *CustomersEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("customers endpoint group: core client is required")
	}
	return nil
}

func (*CustomersEndpointGroup) Materialize(config *CustomersEndpointGroupConfig) *CustomersEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	customerSvc := customerep.NewCustomerSvc(&customerep.CustomerSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Customers",
		Description:  "Manage customer accounts.",
		ResourceType: &apiresource.Customer{},
	}

	listEndpoint := (&customerep.ListCustomersEndpoint{}).Materialize().WithService(inner, customerSvc)
	retrieveEndpoint := (&customerep.RetrieveCustomerEndpoint{}).Materialize().WithService(inner, customerSvc)
	createEndpoint := (&customerep.CreateCustomerEndpoint{}).Materialize().WithService(inner, customerSvc)
	deleteEndpoint := (&customerep.DeleteCustomerEndpoint{}).Materialize().WithService(inner, customerSvc)
	bulkDeleteEndpoint := (&customerep.BulkDeleteCustomersEndpoint{}).Materialize().WithService(inner, customerSvc)
	frequentlyOrderedEndpoint := (&customerep.GetFrequentlyOrderedProductsEndpoint{}).Materialize().WithService(inner, customerSvc)
	mergeEndpoint := (&customerep.MergeCustomersEndpoint{}).Materialize().WithService(inner, customerSvc)
	updateEndpoint := (&customerep.UpdateCustomerEndpoint{}).Materialize().WithService(inner, customerSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		retrieveEndpoint,
		createEndpoint,
		updateEndpoint,
		deleteEndpoint,
		bulkDeleteEndpoint,
		frequentlyOrderedEndpoint,
		mergeEndpoint,
	}

	return &CustomersEndpointGroup{inner}
}
