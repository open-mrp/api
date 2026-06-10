package httpgroup

import (
	"fmt"

	receivableep "github.com/augno/api/services/api-gateway/endpoints/receivables"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type ReceivablesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type ReceivablesEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *ReceivablesEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("receivables endpoint group: core client is required")
	}
	return nil
}

func (*ReceivablesEndpointGroup) Materialize(config *ReceivablesEndpointGroupConfig) *ReceivablesEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	receivableSvc := receivableep.NewReceivableSvc(&receivableep.ReceivableSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Receivables",
		Description:  "List, export, and email receivable entries.",
		ResourceType: &apiresource.ReceivableEntry{},
	}

	listReceivablesEndpoint := apiendpoint.From(&receivableep.ListReceivablesEndpoint{}).WithService(inner, receivableSvc)
	listReceivablesByCustomerEndpoint := apiendpoint.From(&receivableep.ListReceivablesByCustomerEndpoint{}).WithService(inner, receivableSvc)
	exportReceivablesByCustomerEndpoint := apiendpoint.From(&receivableep.ExportReceivablesByCustomerEndpoint{}).WithService(inner, receivableSvc)
	emailReceivablesForCustomerEndpoint := apiendpoint.From(&receivableep.EmailReceivablesForCustomerEndpoint{}).WithService(inner, receivableSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listReceivablesEndpoint,
		listReceivablesByCustomerEndpoint,
		exportReceivablesByCustomerEndpoint,
		emailReceivablesForCustomerEndpoint,
	}

	return &ReceivablesEndpointGroup{inner}
}
