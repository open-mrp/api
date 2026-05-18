package httpgroup

import (
	"fmt"

	salesorderstatusep "github.com/augno/api/services/api-gateway/endpoints/sales-order-statuses"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type SalesOrderStatusesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type SalesOrderStatusesEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *SalesOrderStatusesEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("sales order statuses endpoint group: core client is required")
	}
	return nil
}

func (*SalesOrderStatusesEndpointGroup) Materialize(config *SalesOrderStatusesEndpointGroupConfig) *SalesOrderStatusesEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := salesorderstatusep.NewSalesOrderStatusSvc(&salesorderstatusep.SalesOrderStatusSvcConfig{
		CoreClient: config.CoreClient.Sales,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Sales Order Statuses",
		Description:  "List sales order statuses.",
		ResourceType: &apiresource.SalesOrderStatus{},
	}

	listEndpoint := apiendpoint.From(&salesorderstatusep.ListSalesOrderStatusesEndpoint{}).WithService(inner, svc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
	}

	return &SalesOrderStatusesEndpointGroup{inner}
}
