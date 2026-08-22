package httpgroup

import (
	"fmt"

	customerep "github.com/open-mrp/api/services/api-gateway/endpoints/customers"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
)

type CustomersEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type CustomersEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
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

	listEndpoint := apiendpoint.From(&customerep.ListCustomersEndpoint{}).WithService(inner, customerSvc)
	retrieveEndpoint := apiendpoint.From(&customerep.RetrieveCustomerEndpoint{}).WithService(inner, customerSvc)
	createEndpoint := apiendpoint.From(&customerep.CreateCustomerEndpoint{}).WithService(inner, customerSvc)
	deleteEndpoint := apiendpoint.From(&customerep.DeleteCustomerEndpoint{}).WithService(inner, customerSvc)
	bulkDeleteEndpoint := apiendpoint.From(&customerep.BulkDeleteCustomersEndpoint{}).WithService(inner, customerSvc)
	frequentlyOrderedEndpoint := apiendpoint.From(&customerep.GetFrequentlyOrderedProductsEndpoint{}).WithService(inner, customerSvc)
	leadTimeEndpoint := apiendpoint.From(&customerep.RetrieveCustomerLeadTimeEndpoint{}).WithService(inner, customerSvc)
	listNotificationRecipientsEndpoint := apiendpoint.From(&customerep.ListNotificationRecipientsEndpoint{}).WithService(inner, customerSvc)
	updateNotificationRecipientsEndpoint := apiendpoint.From(&customerep.UpdateNotificationRecipientsEndpoint{}).WithService(inner, customerSvc)
	mergeEndpoint := apiendpoint.From(&customerep.MergeCustomersEndpoint{}).WithService(inner, customerSvc)
	updateEndpoint := apiendpoint.From(&customerep.UpdateCustomerEndpoint{}).WithService(inner, customerSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		retrieveEndpoint,
		createEndpoint,
		updateEndpoint,
		deleteEndpoint,
		bulkDeleteEndpoint,
		frequentlyOrderedEndpoint,
		leadTimeEndpoint,
		listNotificationRecipientsEndpoint,
		updateNotificationRecipientsEndpoint,
		mergeEndpoint,
	}

	return &CustomersEndpointGroup{inner}
}
