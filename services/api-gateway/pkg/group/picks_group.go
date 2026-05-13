package httpgroup

import (
	"fmt"

	pickep "github.com/augno/api/services/api-gateway/endpoints/picks"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type PicksEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type PicksEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *PicksEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("picks endpoint group: core client is required")
	}
	return nil
}

func (*PicksEndpointGroup) Materialize(config *PicksEndpointGroupConfig) *PicksEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := pickep.NewPickSvc(&pickep.PickSvcConfig{
		CoreClient: config.CoreClient.Picking,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Picks",
		Description:  "List, view, update, pick, void, and pack picks and pick lines.",
		ResourceType: &apiresource.PickDetail{},
	}

	listEndpoint := (&pickep.ListPicksEndpoint{}).Materialize().WithService(inner, svc)
	retrieveEndpoint := (&pickep.RetrievePickEndpoint{}).Materialize().WithService(inner, svc)
	updateEndpoint := (&pickep.UpdatePickEndpoint{}).Materialize().WithService(inner, svc)
	pickAllLinesEndpoint := (&pickep.PickAllLinesEndpoint{}).Materialize().WithService(inner, svc)
	voidEndpoint := (&pickep.VoidPickEndpoint{}).Materialize().WithService(inner, svc)
	packEndpoint := (&pickep.PackPickEndpoint{}).Materialize().WithService(inner, svc)
	getShipmentsEndpoint := (&pickep.GetPickShipmentsEndpoint{}).Materialize().WithService(inner, svc)
	updateLineEndpoint := (&pickep.UpdatePickLineEndpoint{}).Materialize().WithService(inner, svc)
	pickLineEndpoint := (&pickep.PickPickLineEndpoint{}).Materialize().WithService(inner, svc)
	voidLineEndpoint := (&pickep.VoidPickLineEndpoint{}).Materialize().WithService(inner, svc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		retrieveEndpoint,
		updateEndpoint,
		pickAllLinesEndpoint,
		voidEndpoint,
		packEndpoint,
		getShipmentsEndpoint,
		updateLineEndpoint,
		pickLineEndpoint,
		voidLineEndpoint,
	}

	return &PicksEndpointGroup{inner}
}
