package httpgroup

import (
	"fmt"

	manufacturingmethodep "github.com/open-mrp/api/services/api-gateway/endpoints/manufacturing-methods"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

type ManufacturingMethodsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type ManufacturingMethodsEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *ManufacturingMethodsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("manufacturing methods endpoint group: core client is required")
	}
	return nil
}

func (*ManufacturingMethodsEndpointGroup) Materialize(config *ManufacturingMethodsEndpointGroupConfig) *ManufacturingMethodsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := manufacturingmethodep.NewManufacturingMethodSvc(&manufacturingmethodep.ManufacturingMethodSvcConfig{CoreClient: config.CoreClient.ManufacturingMethod})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Manufacturing Methods",
		Description:  "How items are made: versioned operations at work centers and the materials they consume. Edit a draft, then activate it for new production.",
		ResourceType: &apiresource.ManufacturingMethod{},
	}

	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&manufacturingmethodep.ListManufacturingMethodsEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&manufacturingmethodep.RetrieveManufacturingMethodEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&manufacturingmethodep.CreateManufacturingMethodEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&manufacturingmethodep.UpdateManufacturingMethodEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&manufacturingmethodep.DeleteManufacturingMethodEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&manufacturingmethodep.ActivateManufacturingMethodEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&manufacturingmethodep.NewManufacturingMethodVersionEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&manufacturingmethodep.GetProductionFlowEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&manufacturingmethodep.ListMethodOperationsEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&manufacturingmethodep.RetrieveMethodOperationEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&manufacturingmethodep.CreateMethodOperationEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&manufacturingmethodep.UpdateMethodOperationEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&manufacturingmethodep.DeleteMethodOperationEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&manufacturingmethodep.ListMethodMaterialsEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&manufacturingmethodep.RetrieveMethodMaterialEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&manufacturingmethodep.CreateMethodMaterialEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&manufacturingmethodep.UpdateMethodMaterialEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&manufacturingmethodep.DeleteMethodMaterialEndpoint{}).WithService(inner, svc),
	}

	return &ManufacturingMethodsEndpointGroup{inner}
}
