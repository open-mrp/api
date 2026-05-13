package httpgroup

import (
	"fmt"

	invoiceep "github.com/augno/api/services/api-gateway/endpoints/invoices"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type InvoicesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type InvoicesEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *InvoicesEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("invoices endpoint group: core client is required")
	}
	return nil
}

func (*InvoicesEndpointGroup) Materialize(config *InvoicesEndpointGroupConfig) *InvoicesEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	invoiceSvc := invoiceep.NewInvoiceSvc(&invoiceep.InvoiceSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Invoices",
		Description:  "List, view, and update invoices.",
		ResourceType: &apiresource.Invoice{},
	}

	listInvoicesEndpoint := (&invoiceep.ListInvoicesEndpoint{}).Materialize().WithService(inner, invoiceSvc)
	getInvoiceEndpoint := (&invoiceep.RetrieveInvoiceEndpoint{}).Materialize().WithService(inner, invoiceSvc)
	updateInvoiceEndpoint := (&invoiceep.UpdateInvoiceEndpoint{}).Materialize().WithService(inner, invoiceSvc)
	listCustomerInvoicesEndpoint := (&invoiceep.ListCustomerInvoicesEndpoint{}).Materialize().WithService(inner, invoiceSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listInvoicesEndpoint,
		getInvoiceEndpoint,
		updateInvoiceEndpoint,
		listCustomerInvoicesEndpoint,
	}

	return &InvoicesEndpointGroup{inner}
}
