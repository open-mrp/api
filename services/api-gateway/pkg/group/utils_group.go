package httpgroup

import (
	"fmt"

	utilsep "github.com/open-mrp/api/services/api-gateway/endpoints/utils"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
)

type UtilsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type UtilsEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *UtilsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("utils endpoint group: core client is required")
	}
	return nil
}

func (*UtilsEndpointGroup) Materialize(config *UtilsEndpointGroupConfig) *UtilsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	utilsSvc := utilsep.NewUtilsSvc(&utilsep.UtilsSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Utils",
		Description:  "Utility action endpoints for checking duplicates and emailing records.",
		ResourceType: &apiresource.CheckDuplicateResult{},
	}

	checkDuplicateEndpoint := apiendpoint.From(&utilsep.CheckDuplicateEndpoint{}).WithService(inner, utilsSvc)
	emailRecordEndpoint := apiendpoint.From(&utilsep.EmailRecordEndpoint{}).WithService(inner, utilsSvc)
	requestDemoEndpoint := apiendpoint.From(&utilsep.RequestDemoEndpoint{}).WithService(inner, utilsSvc)
	submitFeedbackEndpoint := apiendpoint.From(&utilsep.SubmitFeedbackEndpoint{}).WithService(inner, utilsSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		checkDuplicateEndpoint,
		emailRecordEndpoint,
		requestDemoEndpoint,
		submitFeedbackEndpoint,
	}

	return &UtilsEndpointGroup{inner}
}
