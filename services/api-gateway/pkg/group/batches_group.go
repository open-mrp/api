package httpgroup

import (
	"fmt"

	batchep "github.com/open-mrp/api/services/api-gateway/endpoints/batches"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
)

type BatchesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type BatchesEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *BatchesEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("batches endpoint group: core client is required")
	}
	return nil
}

func (*BatchesEndpointGroup) Materialize(config *BatchesEndpointGroupConfig) *BatchesEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	batchSvc := batchep.NewBatchSvc(&batchep.BatchSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Batches",
		Description:  "Manage production batches, batch flows, and scanning station operations.",
		ResourceType: &apiresource.Batch{},
	}

	getBatchFlowEndpoint := apiendpoint.From(&batchep.GetBatchFlowEndpoint{}).WithService(inner, batchSvc)
	listByScanningStationEndpoint := apiendpoint.From(&batchep.ListBatchesByScanningStationEndpoint{}).WithService(inner, batchSvc)
	getPossibleNextStepsEndpoint := apiendpoint.From(&batchep.GetPossibleNextStepsEndpoint{}).WithService(inner, batchSvc)
	getPossibleInitStepsEndpoint := apiendpoint.From(&batchep.GetPossibleInitStepsEndpoint{}).WithService(inner, batchSvc)
	analyzeOpenBatchesEndpoint := apiendpoint.From(&batchep.AnalyzeOpenBatchesEndpoint{}).WithService(inner, batchSvc)
	initializeBatchEndpoint := apiendpoint.From(&batchep.InitializeBatchEndpoint{}).WithService(inner, batchSvc)
	moveBatchesEndpoint := apiendpoint.From(&batchep.MoveBatchesEndpoint{}).WithService(inner, batchSvc)
	mergeBatchesEndpoint := apiendpoint.From(&batchep.MergeBatchesEndpoint{}).WithService(inner, batchSvc)
	splitBatchEndpoint := apiendpoint.From(&batchep.SplitBatchEndpoint{}).WithService(inner, batchSvc)
	getRemainingToSplitEndpoint := apiendpoint.From(&batchep.GetRemainingQuantityToSplitEndpoint{}).WithService(inner, batchSvc)
	getConsumptionEndpoint := apiendpoint.From(&batchep.GetScanningStationConsumptionEndpoint{}).WithService(inner, batchSvc)
	closeBatchEndpoint := apiendpoint.From(&batchep.CloseBatchEndpoint{}).WithService(inner, batchSvc)
	deleteBatchEndpoint := apiendpoint.From(&batchep.DeleteBatchEndpoint{}).WithService(inner, batchSvc)
	bulkDeleteBatchesEndpoint := apiendpoint.From(&batchep.BulkDeleteBatchesEndpoint{}).WithService(inner, batchSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		getBatchFlowEndpoint,
		listByScanningStationEndpoint,
		getPossibleNextStepsEndpoint,
		getPossibleInitStepsEndpoint,
		analyzeOpenBatchesEndpoint,
		initializeBatchEndpoint,
		moveBatchesEndpoint,
		mergeBatchesEndpoint,
		splitBatchEndpoint,
		getRemainingToSplitEndpoint,
		getConsumptionEndpoint,
		closeBatchEndpoint,
		deleteBatchEndpoint,
		bulkDeleteBatchesEndpoint,
	}

	return &BatchesEndpointGroup{inner}
}
