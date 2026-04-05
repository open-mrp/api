package httpgroup

import (
	"fmt"

	ediep "github.com/augno/api/services/api-gateway/endpoints/edi"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type EDIEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type EDIEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *EDIEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("edi endpoint group: core client is required")
	}
	return nil
}

func (*EDIEndpointGroup) Materialize(config *EDIEndpointGroupConfig) *EDIEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	ediSvc := ediep.NewEDISvc(&ediep.EDISvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "EDI",
		Description:  "EDI action endpoints for pulling orders and resubmitting invoices.",
		ResourceType: &apiresource.MessageResource{},
	}

	pullOrdersEndpoint := (&ediep.PullEDIOrdersEndpoint{}).Materialize().WithService(inner, ediSvc)
	resubmitInvoiceEndpoint := (&ediep.ResubmitEDIInvoiceEndpoint{}).Materialize().WithService(inner, ediSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		pullOrdersEndpoint,
		resubmitInvoiceEndpoint,
	}

	return &EDIEndpointGroup{inner}
}
