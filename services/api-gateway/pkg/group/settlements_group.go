package httpgroup

import (
	"fmt"

	settlementep "github.com/augno/api/services/api-gateway/endpoints/settlements"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type SettlementsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type SettlementsEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *SettlementsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("settlements endpoint group: core client is required")
	}
	return nil
}

func (*SettlementsEndpointGroup) Materialize(config *SettlementsEndpointGroupConfig) *SettlementsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	settlementSvc := settlementep.NewSettlementSvc(&settlementep.SettlementSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Settlements",
		Description:  "Create, view, update, and delete settlements.",
		ResourceType: &apiresource.Settlement{},
	}

	listSettlementsEndpoint := (&settlementep.ListSettlementsEndpoint{}).Materialize().WithService(inner, settlementSvc)
	getSettlementEndpoint := (&settlementep.RetrieveSettlementEndpoint{}).Materialize().WithService(inner, settlementSvc)
	createSettlementEndpoint := (&settlementep.CreateSettlementEndpoint{}).Materialize().WithService(inner, settlementSvc)
	updateSettlementEndpoint := (&settlementep.UpdateSettlementEndpoint{}).Materialize().WithService(inner, settlementSvc)
	deleteSettlementEndpoint := (&settlementep.DeleteSettlementEndpoint{}).Materialize().WithService(inner, settlementSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listSettlementsEndpoint,
		getSettlementEndpoint,
		createSettlementEndpoint,
		updateSettlementEndpoint,
		deleteSettlementEndpoint,
	}

	return &SettlementsEndpointGroup{inner}
}
