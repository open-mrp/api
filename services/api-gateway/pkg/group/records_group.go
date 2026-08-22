package httpgroup

import (
	"fmt"

	recordsep "github.com/open-mrp/api/services/api-gateway/endpoints/records"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
)

type RecordsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type RecordsEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *RecordsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("records endpoint group: core client is required")
	}
	return nil
}

func (*RecordsEndpointGroup) Materialize(config *RecordsEndpointGroupConfig) *RecordsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	// The pack list is assembled from shipment, sales-order, and account data, so
	// this group wires all three core sub-clients.
	recordsSvc := recordsep.NewRecordsSvc(&recordsep.RecordsSvcConfig{
		ShippingClient: config.CoreClient.Shipping,
		SalesClient:    config.CoreClient.Sales,
		AccountClient:  config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Records",
		Description:  "Cross-record document generation, such as pack lists.",
		ResourceType: &apiresource.PackList{},
	}

	genPackListEndpoint := apiendpoint.From(&recordsep.GenPackListEndpoint{}).WithService(inner, recordsSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		genPackListEndpoint,
	}

	return &RecordsEndpointGroup{inner}
}
