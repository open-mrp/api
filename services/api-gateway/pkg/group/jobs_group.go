package httpgroup

import (
	"fmt"

	jobep "github.com/augno/api/services/api-gateway/endpoints/jobs"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type JobsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type JobsEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *JobsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("jobs endpoint group: core client is required")
	}
	return nil
}

func (*JobsEndpointGroup) Materialize(config *JobsEndpointGroupConfig) *JobsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := jobep.NewJobSvc(&jobep.JobSvcConfig{
		CoreClient: config.CoreClient.Job,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Jobs",
		Description:  "View the jobs that track asynchronous work. Endpoints that answer 202 Accepted raise one and point at it with a Location header.",
		ResourceType: &apiresource.Job{},
	}

	retrieveEndpoint := apiendpoint.From(&jobep.RetrieveJobEndpoint{}).WithService(inner, svc)
	cancelEndpoint := apiendpoint.From(&jobep.CancelJobEndpoint{}).WithService(inner, svc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		retrieveEndpoint,
		cancelEndpoint,
	}

	return &JobsEndpointGroup{inner}
}
