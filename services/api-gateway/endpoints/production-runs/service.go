package productionrunep

import (
	"context"
	"fmt"

	batchep "github.com/augno/api/services/api-gateway/endpoints/batches"
	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ProductionRunSvc interface {
	ListProductionRuns(ctx context.Context, req *ListProductionRunsRequest) (*apiresource.List[apiresource.ProductionRunSummary], *apierror.APIError)
	GetProductionRun(ctx context.Context, req *RetrieveProductionRunRequest) (*apiresource.ProductionRunDetail, *apierror.APIError)
	CreateProductionRun(ctx context.Context, req *CreateProductionRunRequest) (*apiresource.ProductionRunDetail, *apierror.APIError)
	UpdateProductionRun(ctx context.Context, req *UpdateProductionRunRequest) (*apiresource.ProductionRunDetail, *apierror.APIError)
	DeleteProductionRun(ctx context.Context, req *DeleteProductionRunRequest) (*apiresource.EmptyResource, *apierror.APIError)
	AddBatchesToProductionRun(ctx context.Context, req *AddBatchesToProductionRunRequest) (*apiresource.List[apiresource.Batch], *apierror.APIError)
	ListBatchesByProductionRun(ctx context.Context, req *ListBatchesByProductionRunRequest) (*apiresource.List[apiresource.Batch], *apierror.APIError)
}

type ProductionRunSvcConfig struct {
	CoreClient pb.CoreProductionRunServiceClient
}

type productionRunSvcImpl struct {
	coreClient pb.CoreProductionRunServiceClient
}

var productionRunEpSvcTracer = tracing.GetTracer("api-gateway.endpoints.production-runs.service")

func (c *ProductionRunSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("production run endpoint service: core client is required")
	}
	return nil
}

func NewProductionRunSvc(config *ProductionRunSvcConfig) ProductionRunSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &productionRunSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *productionRunSvcImpl) ListProductionRuns(ctx context.Context, req *ListProductionRunsRequest) (*apiresource.List[apiresource.ProductionRunSummary], *apierror.APIError) {
	pbReq := &pb.ListProductionRunsRequest{
		Cursor:     req.Cursor,
		Limit:      req.Limit,
		Query:      req.Query,
		Status:     req.Status,
		ItemIds:    req.ItemIDs,
		MachineIds: req.MachineIDs,
		StartDate:  req.StartDate,
		EndDate:    req.EndDate,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productionRunEpSvcTracer, "service.production_runs.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListProductionRunsResponse, error) {
			return m.coreClient.ListProductionRuns(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return productionRunListFromProto(ctx, resp), nil
}

func (m *productionRunSvcImpl) GetProductionRun(ctx context.Context, req *RetrieveProductionRunRequest) (*apiresource.ProductionRunDetail, *apierror.APIError) {
	pbReq := &pb.GetProductionRunRequest{
		Id: req.ProductionRunID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productionRunEpSvcTracer, "service.production_runs.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetProductionRunResponse, error) {
			return m.coreClient.GetProductionRun(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := productionRunDetailFromProto(resp.ProductionRun)
	stashProductionRunDetailMeta(meta, resp.ProductionRun)
	return &result, nil
}

func (m *productionRunSvcImpl) CreateProductionRun(ctx context.Context, req *CreateProductionRunRequest) (*apiresource.ProductionRunDetail, *apierror.APIError) {
	pbReq := &pb.CreateProductionRunRequest{
		ResponsibleUserId: req.ResponsibleUserID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productionRunEpSvcTracer, "service.production_runs.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateProductionRunResponse, error) {
			return m.coreClient.CreateProductionRun(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := productionRunDetailFromProto(resp.ProductionRun)
	stashProductionRunDetailMeta(meta, resp.ProductionRun)
	return &result, nil
}

func (m *productionRunSvcImpl) UpdateProductionRun(ctx context.Context, req *UpdateProductionRunRequest) (*apiresource.ProductionRunDetail, *apierror.APIError) {
	pbReq := &pb.UpdateProductionRunRequest{
		Id:                req.ProductionRunID,
		Number:            req.Number.Ptr(),
		ResponsibleUserId: req.ResponsibleUserID.Ptr(),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productionRunEpSvcTracer, "service.production_runs.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateProductionRunResponse, error) {
			return m.coreClient.UpdateProductionRun(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := productionRunDetailFromProto(resp.ProductionRun)
	stashProductionRunDetailMeta(meta, resp.ProductionRun)
	return &result, nil
}

func (m *productionRunSvcImpl) DeleteProductionRun(ctx context.Context, req *DeleteProductionRunRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteProductionRunRequest{Id: req.ProductionRunID}

	_, apiErr := grpcutil.CallRPC(ctx, productionRunEpSvcTracer, "service.production_runs.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteProductionRun(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *productionRunSvcImpl) AddBatchesToProductionRun(ctx context.Context, req *AddBatchesToProductionRunRequest) (*apiresource.List[apiresource.Batch], *apierror.APIError) {
	pbBatches := make([]*pb.AddBatchInput, len(req.Batches))
	for i, b := range req.Batches {
		pbBatches[i] = &pb.AddBatchInput{
			ItemId:            b.ItemID,
			QuantityValue:     b.QuantityValue,
			QuantityUnitId:    b.QuantityUnitID,
			SecondsValue:      b.SecondsValue,
			SecondsUnitId:     b.SecondsUnitID,
			WasteValue:        b.WasteValue,
			WasteUnitId:       b.WasteUnitID,
			ProductionStepId:  b.ProductionStepID,
			ScanningStationId: b.ScanningStationID,
		}
	}

	pbReq := &pb.AddBatchesToProductionRunRequest{
		ProductionRunId: req.ProductionRunID,
		Batches:         pbBatches,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productionRunEpSvcTracer, "service.production_runs.add_batches", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AddBatchesToProductionRunResponse, error) {
			return m.coreClient.AddBatchesToProductionRun(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return addBatchesFromProto(resp), nil
}

func (m *productionRunSvcImpl) ListBatchesByProductionRun(ctx context.Context, req *ListBatchesByProductionRunRequest) (*apiresource.List[apiresource.Batch], *apierror.APIError) {
	pbReq := &pb.ListBatchesByProductionRunRequest{
		ProductionRunId: req.ProductionRunID,
		Limit:           req.Limit,
	}
	if req.Cursor != nil {
		pbReq.Cursor = req.Cursor
	}
	if req.Query != nil {
		pbReq.Query = req.Query
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productionRunEpSvcTracer, "service.production_runs.list_batches", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListBatchesByProductionRunResponse, error) {
			return m.coreClient.ListBatchesByProductionRun(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return listBatchesFromProto(ctx, resp), nil
}

func productionRunSummaryFromProto(info *pb.ProductionRunSummaryInfo) apiresource.ProductionRunSummary {
	s := apiresource.ProductionRunSummary{
		ID:         info.Id,
		Object:     constants.ObjectTypeProductionRun,
		Number:     info.Number,
		BatchCount: info.BatchCount,
		CreatedAt:  grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt:  grpcutil.TimestampToTime(info.UpdatedAt),
	}

	if info.ResponsibleUserId != "" {
		s.ResponsibleUser = &apiresource.AccountUser{
			ID:        info.ResponsibleUserId,
			Object:    constants.ObjectTypeAccountUser,
			Name:      info.ResponsibleUserName,
			Status:    constants.AccountUserStatus(info.GetResponsibleUserStatusCode()),
			CreatedAt: grpcutil.TimestampToTime(info.ResponsibleUserCreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(info.ResponsibleUserUpdatedAt),
		}
	}

	s.StartedAt = grpcutil.TimestampToTimePtr(info.StartedAt)
	s.CompletedAt = grpcutil.TimestampToTimePtr(info.CompletedAt)

	return s
}

func productionRunListFromProto(ctx context.Context, resp *pb.ListProductionRunsResponse) *apiresource.List[apiresource.ProductionRunSummary] {
	runs := make([]apiresource.ProductionRunSummary, len(resp.ProductionRuns))
	for i, pr := range resp.ProductionRuns {
		runs[i] = productionRunSummaryFromProto(pr)
	}

	return apiresource.NewList(runs, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}

func productionRunDetailFromProto(info *pb.ProductionRunInfo) apiresource.ProductionRunDetail {
	d := apiresource.ProductionRunDetail{
		ID:         info.Id,
		Object:     constants.ObjectTypeProductionRun,
		Number:     info.Number,
		BatchCount: info.BatchCount,
		CreatedAt:  grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt:  grpcutil.TimestampToTime(info.UpdatedAt),
	}

	d.StartedAt = grpcutil.TimestampToTimePtr(info.StartedAt)
	d.CompletedAt = grpcutil.TimestampToTimePtr(info.CompletedAt)

	return d
}

func stashProductionRunDetailMeta(meta *resourcekit.LoadMeta, info *pb.ProductionRunInfo) {
	if info == nil || info.ResponsibleUserId == "" {
		return
	}
	user := &apiresource.AccountUser{
		ID:        info.ResponsibleUserId,
		Object:    constants.ObjectTypeAccountUser,
		Name:      info.ResponsibleUserName,
		Status:    constants.AccountUserStatus(info.GetResponsibleUserStatusCode()),
		CreatedAt: grpcutil.TimestampToTime(info.ResponsibleUserCreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(info.ResponsibleUserUpdatedAt),
	}
	meta.Set(constants.ObjectTypeProductionRun, info.Id, "responsible_user", user)
}

func addBatchesFromProto(resp *pb.AddBatchesToProductionRunResponse) *apiresource.List[apiresource.Batch] {
	batches := make([]apiresource.Batch, len(resp.Batches))
	for i, b := range resp.Batches {
		batches[i] = batchep.BaseBatchPresenter(b)
	}

	return apiresource.NewList(batches, apiresource.PageInfo{})
}

func listBatchesFromProto(ctx context.Context, resp *pb.ListBatchesByProductionRunResponse) *apiresource.List[apiresource.Batch] {
	batches := make([]apiresource.Batch, len(resp.Batches))
	for i, b := range resp.Batches {
		batches[i] = batchep.BatchPresenter(b)
	}

	var pi apiresource.PageInfo
	if resp.PageInfo != nil {
		pi = grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)
	}

	return apiresource.NewList(batches, pi)
}
