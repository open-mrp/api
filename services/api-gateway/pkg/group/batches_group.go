package httpgroup

import (
	"fmt"

	batchep "github.com/augno/api/services/api-gateway/endpoints/batches"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type BatchesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type BatchesEndpointGroupConfig struct {
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

	getBatchFlowEndpoint := (&batchep.GetBatchFlowEndpoint{}).Materialize().WithService(inner, batchSvc)
	listByScanningStationEndpoint := (&batchep.ListBatchesByScanningStationEndpoint{}).Materialize().WithService(inner, batchSvc)
	getPossibleNextStepsEndpoint := (&batchep.GetPossibleNextStepsEndpoint{}).Materialize().WithService(inner, batchSvc)
	analyzeOpenBatchesEndpoint := (&batchep.AnalyzeOpenBatchesEndpoint{}).Materialize().WithService(inner, batchSvc)
	initializeBatchEndpoint := (&batchep.InitializeBatchEndpoint{}).Materialize().WithService(inner, batchSvc)
	moveBatchesEndpoint := (&batchep.MoveBatchesEndpoint{}).Materialize().WithService(inner, batchSvc)
	mergeBatchesEndpoint := (&batchep.MergeBatchesEndpoint{}).Materialize().WithService(inner, batchSvc)
	splitBatchEndpoint := (&batchep.SplitBatchEndpoint{}).Materialize().WithService(inner, batchSvc)
	getRemainingToSplitEndpoint := (&batchep.GetRemainingQuantityToSplitEndpoint{}).Materialize().WithService(inner, batchSvc)
	getConsumptionEndpoint := (&batchep.GetScanningStationConsumptionEndpoint{}).Materialize().WithService(inner, batchSvc)
	closeBatchEndpoint := (&batchep.CloseBatchEndpoint{}).Materialize().WithService(inner, batchSvc)
	deleteBatchEndpoint := (&batchep.DeleteBatchEndpoint{}).Materialize().WithService(inner, batchSvc)
	bulkDeleteBatchesEndpoint := (&batchep.BulkDeleteBatchesEndpoint{}).Materialize().WithService(inner, batchSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		getBatchFlowEndpoint,
		listByScanningStationEndpoint,
		getPossibleNextStepsEndpoint,
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
