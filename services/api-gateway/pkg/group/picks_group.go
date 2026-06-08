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
		ResourceType: &apiresource.Pick{},
	}

	listEndpoint := apiendpoint.From(&pickep.ListPicksEndpoint{}).WithService(inner, svc)
	retrieveEndpoint := apiendpoint.From(&pickep.RetrievePickEndpoint{}).WithService(inner, svc)
	updateEndpoint := apiendpoint.From(&pickep.UpdatePickEndpoint{}).WithService(inner, svc)
	pickAllLinesEndpoint := apiendpoint.From(&pickep.PickAllLinesEndpoint{}).WithService(inner, svc)
	voidEndpoint := apiendpoint.From(&pickep.VoidPickEndpoint{}).WithService(inner, svc)
	packEndpoint := apiendpoint.From(&pickep.PackPickEndpoint{}).WithService(inner, svc)
	getShipmentsEndpoint := apiendpoint.From(&pickep.GetPickShipmentsEndpoint{}).WithService(inner, svc)
	updateLineEndpoint := apiendpoint.From(&pickep.UpdatePickLineEndpoint{}).WithService(inner, svc)
	pickLineEndpoint := apiendpoint.From(&pickep.PickPickLineEndpoint{}).WithService(inner, svc)
	voidLineEndpoint := apiendpoint.From(&pickep.VoidPickLineEndpoint{}).WithService(inner, svc)

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
