package httpgroup

import (
	"fmt"

	batchreleaseep "github.com/open-mrp/api/services/api-gateway/endpoints/batch-releases"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
)

type BatchReleasesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type BatchReleasesEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *BatchReleasesEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("batch releases endpoint group: core client is required")
	}
	return nil
}

func (*BatchReleasesEndpointGroup) Materialize(config *BatchReleasesEndpointGroupConfig) *BatchReleasesEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := batchreleaseep.NewBatchReleaseSvc(&batchreleaseep.BatchReleaseSvcConfig{
		CoreClient: config.CoreClient.BatchRelease,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Batch Releases",
		Description:  "Electronic sign-offs for production: the release each step requires, the signatures batches receive, and the operators checked in at scanning stations.",
		ResourceType: &apiresource.BatchSignature{},
	}

	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&batchreleaseep.RetrieveReleasePolicyEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&batchreleaseep.UpdateReleasePolicyEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&batchreleaseep.ListEligibleSignersEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&batchreleaseep.ListBatchSignaturesEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&batchreleaseep.SignBatchEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&batchreleaseep.CheckInOperatorEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&batchreleaseep.CheckOutOperatorEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&batchreleaseep.RetrieveActiveOperatorCheckInEndpoint{}).WithService(inner, svc),
	}

	return &BatchReleasesEndpointGroup{inner}
}
