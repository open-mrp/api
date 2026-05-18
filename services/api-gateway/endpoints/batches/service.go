package batchep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type BatchSvc interface {
	GetBatchFlow(ctx context.Context, req *GetBatchFlowRequest) (*apiresource.List[apiresource.BatchFlowNode], *apierror.APIError)
	ListBatchesByScanningStation(ctx context.Context, req *ListBatchesByScanningStationRequest) (*apiresource.List[apiresource.Batch], *apierror.APIError)
	GetPossibleNextSteps(ctx context.Context, req *GetPossibleNextStepsRequest) (*apiresource.List[apiresource.ScanningProductionStepInfo], *apierror.APIError)
	AnalyzeOpenBatches(ctx context.Context, req *AnalyzeOpenBatchesRequest) (*apiresource.List[apiresource.OpenBatchSummary], *apierror.APIError)
	InitializeBatch(ctx context.Context, req *InitializeBatchRequest) (*apiresource.Batch, *apierror.APIError)
	MoveBatches(ctx context.Context, req *MoveBatchesRequest) (*apiresource.Batch, *apierror.APIError)
	MergeBatches(ctx context.Context, req *MergeBatchesRequest) (*apiresource.Batch, *apierror.APIError)
	SplitBatch(ctx context.Context, req *SplitBatchRequest) (*apiresource.Batch, *apierror.APIError)
	GetRemainingQuantityToSplit(ctx context.Context, req *GetRemainingQuantityToSplitRequest) (*apiresource.Quantity, *apierror.APIError)
	GetScanningStationConsumption(ctx context.Context, req *GetScanningStationConsumptionRequest) (*apiresource.List[apiresource.ScanningConsumption], *apierror.APIError)
	CloseBatch(ctx context.Context, req *CloseBatchRequest) (*apiresource.Batch, *apierror.APIError)
	DeleteBatch(ctx context.Context, req *DeleteBatchRequest) (*apiresource.Batch, *apierror.APIError)
	DeleteManyBatches(ctx context.Context, req *DeleteManyBatchesRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type BatchSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type batchSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var batchSvcTracer = tracing.GetTracer("api-gateway.endpoints.batches.service")

func (c *BatchSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("batch endpoint service: core client is required")
	}
	return nil
}

func NewBatchSvc(config *BatchSvcConfig) BatchSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &batchSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *batchSvcImpl) GetBatchFlow(ctx context.Context, req *GetBatchFlowRequest) (*apiresource.List[apiresource.BatchFlowNode], *apierror.APIError) {
	pbReq := &pb.GetBatchFlowRequest{
		BatchId: req.BatchID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, batchSvcTracer, "service.batches.get_batch_flow", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetBatchFlowResponse, error) {
			return m.coreClient.GetBatchFlow(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	nodes := make([]apiresource.BatchFlowNode, len(resp.Nodes))
	for i, n := range resp.Nodes {
		nodes[i] = BatchFlowNodePresenter(n)
	}

	return apiresource.NewList(nodes, apiresource.PageInfo{}), nil
}

func (m *batchSvcImpl) ListBatchesByScanningStation(ctx context.Context, req *ListBatchesByScanningStationRequest) (*apiresource.List[apiresource.Batch], *apierror.APIError) {
	pbReq := &pb.ListBatchesByScanningStationRequest{
		ScanningStationId: req.ScanningStationID,
		Cursor:            req.Cursor,
		Limit:             req.Limit,
		Query:             req.Query,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, batchSvcTracer, "service.batches.list_by_scanning_station", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListBatchesByScanningStationResponse, error) {
			return m.coreClient.ListBatchesByScanningStation(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return BatchListPresenter(ctx, resp), nil
}

func (m *batchSvcImpl) GetPossibleNextSteps(ctx context.Context, req *GetPossibleNextStepsRequest) (*apiresource.List[apiresource.ScanningProductionStepInfo], *apierror.APIError) {
	pbReq := &pb.GetBatchPossibleNextStepsRequest{
		ScanningStationId: req.ScanningStationID,
		BatchId:           req.BatchID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, batchSvcTracer, "service.batches.get_possible_next_steps", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetBatchPossibleNextStepsResponse, error) {
			return m.coreClient.GetBatchPossibleNextSteps(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	steps := make([]apiresource.ScanningProductionStepInfo, len(resp.Steps))
	for i, s := range resp.Steps {
		steps[i] = ScanningProductionStepInfoPresenter(s)
	}

	return apiresource.NewList(steps, apiresource.PageInfo{}), nil
}

func (m *batchSvcImpl) AnalyzeOpenBatches(ctx context.Context, req *AnalyzeOpenBatchesRequest) (*apiresource.List[apiresource.OpenBatchSummary], *apierror.APIError) {
	pbReq := &pb.AnalyzeOpenBatchesRequest{
		ItemIds:        req.ItemIDs,
		ProductLineIds: req.ProductLineIDs,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, batchSvcTracer, "service.batches.analyze_open_batches", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AnalyzeOpenBatchesResponse, error) {
			return m.coreClient.AnalyzeOpenBatches(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	summaries := make([]apiresource.OpenBatchSummary, len(resp.Summaries))
	for i, s := range resp.Summaries {
		summaries[i] = OpenBatchSummaryPresenter(s)
	}

	return apiresource.NewList(summaries, apiresource.PageInfo{}), nil
}

func (m *batchSvcImpl) InitializeBatch(ctx context.Context, req *InitializeBatchRequest) (*apiresource.Batch, *apierror.APIError) {
	pbReq := &pb.InitializeBatchRequest{
		BatchId:           req.BatchID,
		ScanningStationId: req.ScanningStationID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, batchSvcTracer, "service.batches.initialize", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.InitializeBatchResponse, error) {
			return m.coreClient.InitializeBatch(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := BaseBatchPresenter(resp.Batch)
	return &result, nil
}

func (m *batchSvcImpl) MoveBatches(ctx context.Context, req *MoveBatchesRequest) (*apiresource.Batch, *apierror.APIError) {
	pbReq := &pb.MoveBatchesRequest{
		BatchIds:          req.BatchIDs,
		ProductionStepId:  req.ProductionStepID,
		ScanningStationId: req.ScanningStationID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, batchSvcTracer, "service.batches.move", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.MoveBatchesResponse, error) {
			return m.coreClient.MoveBatches(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := BaseBatchPresenter(resp.Batch)
	return &result, nil
}

func (m *batchSvcImpl) MergeBatches(ctx context.Context, req *MergeBatchesRequest) (*apiresource.Batch, *apierror.APIError) {
	pbReq := &pb.MergeBatchesRequest{
		BatchIds:          req.BatchIDs,
		ScanningStationId: req.ScanningStationID,
		ProductionStepId:  req.ProductionStepID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, batchSvcTracer, "service.batches.merge", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.MergeBatchesResponse, error) {
			return m.coreClient.MergeBatches(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := BaseBatchPresenter(resp.Batch)
	return &result, nil
}

func (m *batchSvcImpl) SplitBatch(ctx context.Context, req *SplitBatchRequest) (*apiresource.Batch, *apierror.APIError) {
	pbReq := &pb.SplitBatchRequest{
		BatchIds:          req.BatchIDs,
		ScanningStationId: req.ScanningStationID,
		ProductionStepId:  req.ProductionStepID,
		Firsts:            &pb.BatchQuantityInfo{Id: req.Firsts.ID, Measure: req.Firsts.Measure, UnitId: req.Firsts.UnitID},
		CloseBatch:        req.CloseBatch,
	}

	if req.Seconds != nil {
		pbReq.Seconds = &pb.BatchQuantityInfo{Id: req.Seconds.ID, Measure: req.Seconds.Measure, UnitId: req.Seconds.UnitID}
	}
	if req.Waste != nil {
		pbReq.Waste = &pb.BatchQuantityInfo{Id: req.Waste.ID, Measure: req.Waste.Measure, UnitId: req.Waste.UnitID}
	}

	resp, apiErr := grpcutil.CallRPC(ctx, batchSvcTracer, "service.batches.split", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.SplitBatchResponse, error) {
			return m.coreClient.SplitBatch(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := BaseBatchPresenter(resp.Batch)
	return &result, nil
}

func (m *batchSvcImpl) GetRemainingQuantityToSplit(ctx context.Context, req *GetRemainingQuantityToSplitRequest) (*apiresource.Quantity, *apierror.APIError) {
	pbReq := &pb.GetRemainingQuantityToSplitRequest{
		BatchIds:         req.BatchIDs,
		ProductionStepId: req.ProductionStepID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, batchSvcTracer, "service.batches.get_remaining_quantity_to_split", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetRemainingQuantityToSplitResponse, error) {
			return m.coreClient.GetRemainingQuantityToSplit(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return BatchQuantityPresenter(resp.Quantity), nil
}

func (m *batchSvcImpl) GetScanningStationConsumption(ctx context.Context, req *GetScanningStationConsumptionRequest) (*apiresource.List[apiresource.ScanningConsumption], *apierror.APIError) {
	pbReq := &pb.GetScanningStationConsumptionRequest{
		ScanningStationId: req.ScanningStationID,
		BatchIds:          req.BatchIDs,
		ProductionStepId:  req.ProductionStepID,
	}

	if req.SplitQuantity != nil {
		pbReq.SplitQuantity = &pb.BatchQuantityInfo{
			Id:      req.SplitQuantity.ID,
			Measure: req.SplitQuantity.Measure,
			UnitId:  req.SplitQuantity.UnitID,
		}
	}

	resp, apiErr := grpcutil.CallRPC(ctx, batchSvcTracer, "service.batches.get_scanning_station_consumption", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetScanningStationConsumptionResponse, error) {
			return m.coreClient.GetScanningStationConsumption(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	consumptions := make([]apiresource.ScanningConsumption, len(resp.Consumptions))
	for i, c := range resp.Consumptions {
		consumptions[i] = ScanningConsumptionPresenter(c)
	}

	return apiresource.NewList(consumptions, apiresource.PageInfo{}), nil
}

func (m *batchSvcImpl) CloseBatch(ctx context.Context, req *CloseBatchRequest) (*apiresource.Batch, *apierror.APIError) {
	pbReq := &pb.CloseBatchRequest{
		BatchId: req.BatchID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, batchSvcTracer, "service.batches.close", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CloseBatchResponse, error) {
			return m.coreClient.CloseBatch(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := BaseBatchPresenter(resp.Batch)
	return &result, nil
}

func (m *batchSvcImpl) DeleteBatch(ctx context.Context, req *DeleteBatchRequest) (*apiresource.Batch, *apierror.APIError) {
	pbReq := &pb.DeleteBatchRequest{
		BatchId: req.BatchID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, batchSvcTracer, "service.batches.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.DeleteBatchResponse, error) {
			return m.coreClient.DeleteBatch(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := BaseBatchPresenter(resp.Batch)
	return &result, nil
}

func (m *batchSvcImpl) DeleteManyBatches(ctx context.Context, req *DeleteManyBatchesRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteManyBatchesRequest{
		BatchIds: req.BatchIDs,
	}

	_, apiErr := grpcutil.CallRPC(ctx, batchSvcTracer, "service.batches.delete_many", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteManyBatches(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}
