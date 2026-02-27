package httpgroup

import (
	"fmt"

	requestlogep "github.com/augno/api/services/api-gateway/endpoints/request_logs"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type RequestLogsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type RequestLogsEndpointGroupConfig struct {
	PlatformClient *grpcclient.PlatformServiceClient
}

func (c *RequestLogsEndpointGroupConfig) validate() error {
	if c.PlatformClient == nil {
		return fmt.Errorf("request logs endpoint group: platform client is required")
	}
	return nil
}

func (*RequestLogsEndpointGroup) Materialize(config *RequestLogsEndpointGroupConfig) *RequestLogsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	requestLogSvc := requestlogep.NewRequestLogSvc(&requestlogep.RequestLogSvcConfig{
		LoggingClient: config.PlatformClient.LoggingClient,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Request Log Management",
		Description:  "Handles listing and retrieving request logs.",
		ResourceType: &apiresource.RequestLog{},
	}

	listEndpoint := (&requestlogep.ListRequestLogsEndpoint{}).Materialize().WithService(inner, requestLogSvc)
	getEndpoint := (&requestlogep.GetRequestLogEndpoint{}).Materialize().WithService(inner, requestLogSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		getEndpoint,
	}

	return &RequestLogsEndpointGroup{inner}
}
