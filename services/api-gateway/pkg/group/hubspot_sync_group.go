package httpgroup

import (
	"fmt"

	hubspotsyncep "github.com/open-mrp/api/services/api-gateway/endpoints/hubspot-sync"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
)

type HubspotSyncEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type HubspotSyncEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *HubspotSyncEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("hubspot sync endpoint group: core client is required")
	}
	return nil
}

func (*HubspotSyncEndpointGroup) Materialize(config *HubspotSyncEndpointGroupConfig) *HubspotSyncEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := hubspotsyncep.NewHubspotSyncSvc(&hubspotsyncep.HubspotSyncSvcConfig{
		CoreClient: config.CoreClient.HubspotSync,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "HubSpot Sync",
		Description:  "Start and manage the one-time HubSpot backfill/reconciliation of existing customers and orders.",
		ResourceType: &apiresource.HubspotSyncJob{},
	}

	startEndpoint := apiendpoint.From(&hubspotsyncep.StartHubspotSyncEndpoint{}).WithService(inner, svc)
	getCurrentEndpoint := apiendpoint.From(&hubspotsyncep.GetCurrentHubspotSyncEndpoint{}).WithService(inner, svc)
	getEndpoint := apiendpoint.From(&hubspotsyncep.GetHubspotSyncJobEndpoint{}).WithService(inner, svc)
	listReviewsEndpoint := apiendpoint.From(&hubspotsyncep.ListHubspotCompanyReviewsEndpoint{}).WithService(inner, svc)
	linkReviewEndpoint := apiendpoint.From(&hubspotsyncep.LinkHubspotCompanyReviewEndpoint{}).WithService(inner, svc)
	createNewReviewEndpoint := apiendpoint.From(&hubspotsyncep.CreateNewHubspotCompanyReviewEndpoint{}).WithService(inner, svc)
	skipReviewEndpoint := apiendpoint.From(&hubspotsyncep.SkipHubspotCompanyReviewEndpoint{}).WithService(inner, svc)
	bulkResolveReviewsEndpoint := apiendpoint.From(&hubspotsyncep.BulkResolveHubspotCompanyReviewsEndpoint{}).WithService(inner, svc)
	exportReviewsEndpoint := apiendpoint.From(&hubspotsyncep.ExportHubspotCompanyReviewsEndpoint{}).WithService(inner, svc)
	executeEndpoint := apiendpoint.From(&hubspotsyncep.ExecuteHubspotSyncEndpoint{}).WithService(inner, svc)
	cancelEndpoint := apiendpoint.From(&hubspotsyncep.CancelHubspotSyncEndpoint{}).WithService(inner, svc)
	listRecordsEndpoint := apiendpoint.From(&hubspotsyncep.ListHubspotSyncRecordsEndpoint{}).WithService(inner, svc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		startEndpoint,
		getCurrentEndpoint,
		getEndpoint,
		listReviewsEndpoint,
		linkReviewEndpoint,
		createNewReviewEndpoint,
		skipReviewEndpoint,
		bulkResolveReviewsEndpoint,
		exportReviewsEndpoint,
		executeEndpoint,
		cancelEndpoint,
		listRecordsEndpoint,
	}

	return &HubspotSyncEndpointGroup{inner}
}
