package httpgroup

import (
	"fmt"

	salestargetep "github.com/augno/api/services/api-gateway/endpoints/sales-targets"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type SalesTargetsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type SalesTargetsEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *SalesTargetsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("sales targets endpoint group: core client is required")
	}
	return nil
}

func (*SalesTargetsEndpointGroup) Materialize(config *SalesTargetsEndpointGroupConfig) *SalesTargetsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := salestargetep.NewSalesTargetSvc(&salestargetep.SalesTargetSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Sales Targets",
		Description:  "List and manage sales targets for account users.",
		ResourceType: &apiresource.SalesTarget{},
	}

	listEndpoint := apiendpoint.From(&salestargetep.ListSalesTargetsEndpoint{}).WithService(inner, svc)
	createEndpoint := apiendpoint.From(&salestargetep.CreateSalesTargetEndpoint{}).WithService(inner, svc)
	upsertEndpoint := apiendpoint.From(&salestargetep.UpsertSalesTargetEndpoint{}).WithService(inner, svc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		createEndpoint,
		upsertEndpoint,
	}

	return &SalesTargetsEndpointGroup{inner}
}
