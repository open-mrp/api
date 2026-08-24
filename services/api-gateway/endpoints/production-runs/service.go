package productionrunep

import (
	"context"
	"fmt"

	batchep "github.com/open-mrp/api/services/api-gateway/endpoints/batches"
	jobep "github.com/open-mrp/api/services/api-gateway/endpoints/jobs"
	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ProductionRunSvc interface {
	ListProductionRuns(ctx context.Context, req *ListProductionRunsRequest) (*apiresource.List[apiresource.ProductionRun], *apierror.APIError)
	ExportProductionRuns(ctx context.Context, req *ExportProductionRunsRequest) (*apiresource.Job, *apierror.APIError)
	GetProductionRun(ctx context.Context, req *RetrieveProductionRunRequest) (*apiresource.ProductionRun, *apierror.APIError)
	CreateProductionRun(ctx context.Context, req *CreateProductionRunRequest) (*apiresource.ProductionRun, *apierror.APIError)
	UpdateProductionRun(ctx context.Context, req *UpdateProductionRunRequest) (*apiresource.ProductionRun, *apierror.APIError)
	DeleteProductionRun(ctx context.Context, req *DeleteProductionRunRequest) (*apiresource.EmptyResource, *apierror.APIError)
	AddBatchesToProductionRun(ctx context.Context, req *AddBatchesToProductionRunRequest) (*apiresource.List[apiresource.Batch], *apierror.APIError)
	ListBatchesByProductionRun(ctx context.Context, req *ListBatchesByProductionRunRequest) (*apiresource.List[apiresource.Batch], *apierror.APIError)
	BulkCreateProductionRuns(ctx context.Context, req *BulkCreateProductionRunsRequest) (*apiresource.Job, *apierror.APIError)
}

type ProductionRunSvcConfig struct {
	// CoreClient (required) is the core-service production-run gRPC client.
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

func (m *productionRunSvcImpl) ExportProductionRuns(ctx context.Context, req *ExportProductionRunsRequest) (*apiresource.Job, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, productionRunEpSvcTracer, "service.production_runs.export", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ExportProductionRunsResponse, error) {
			return m.coreClient.ExportProductionRuns(ctx, &pb.ExportProductionRunsRequest{Query: req.Query}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return jobep.JobFromProto(resp.GetJob()), nil
}

func (m *productionRunSvcImpl) ListProductionRuns(ctx context.Context, req *ListProductionRunsRequest) (*apiresource.List[apiresource.ProductionRun], *apierror.APIError) {
	pbReq := &pb.ListProductionRunsRequest{
		Cursor:     req.Cursor,
		Limit:      req.Limit,
		Query:      req.Query,
		Status:     req.Status.StringPtr(),
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

func (m *productionRunSvcImpl) GetProductionRun(ctx context.Context, req *RetrieveProductionRunRequest) (*apiresource.ProductionRun, *apierror.APIError) {
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
	result := ProductionRunFromProto(resp.ProductionRun)
	StashProductionRunMeta(meta, resp.ProductionRun)
	return &result, nil
}

func (m *productionRunSvcImpl) CreateProductionRun(ctx context.Context, req *CreateProductionRunRequest) (*apiresource.ProductionRun, *apierror.APIError) {
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
	result := ProductionRunFromProto(resp.ProductionRun)
	StashProductionRunMeta(meta, resp.ProductionRun)
	return &result, nil
}

func (m *productionRunSvcImpl) UpdateProductionRun(ctx context.Context, req *UpdateProductionRunRequest) (*apiresource.ProductionRun, *apierror.APIError) {
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
	result := ProductionRunFromProto(resp.ProductionRun)
	StashProductionRunMeta(meta, resp.ProductionRun)
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
			SecondsValue:      b.SecondsValue.Ptr(),
			SecondsUnitId:     b.SecondsUnitID.Ptr(),
			WasteValue:        b.WasteValue.Ptr(),
			WasteUnitId:       b.WasteUnitID.Ptr(),
			ProductionStepId:  b.ProductionStepID.Ptr(),
			ScanningStationId: b.ScanningStationID.Ptr(),
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

func (m *productionRunSvcImpl) BulkCreateProductionRuns(ctx context.Context, req *BulkCreateProductionRunsRequest) (*apiresource.Job, *apierror.APIError) {

	pbRuns := make([]*pb.BulkCreateProductionRunInput, len(req.ProductionRuns))
	for i, r := range req.ProductionRuns {
		batches := make([]*pb.BulkCreateBatchInput, len(r.Batches))
		for j, b := range r.Batches {
			batches[j] = &pb.BulkCreateBatchInput{
				Item:             apirequest.ItemIdentifierToProto(b.Item),
				QuantityValue:    b.QuantityValue,
				QuantityUnit:     apirequest.UnitIdentifierToProto(b.QuantityUnit),
				SecondsValue:     b.SecondsValue.Ptr(),
				SecondsUnit:      apirequest.OptionalUnitIdentifierToProto(b.SecondsUnit),
				WasteValue:       b.WasteValue.Ptr(),
				WasteUnit:        apirequest.OptionalUnitIdentifierToProto(b.WasteUnit),
				ProductionStepId: b.ProductionStepID.Ptr(),
				ScanningStation:  apirequest.OptionalObjectIdentifierToProto(b.ScanningStation),
			}
		}
		pbRuns[i] = &pb.BulkCreateProductionRunInput{
			ResponsibleUserId: r.ResponsibleUserID,
			Batches:           batches,
		}
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productionRunEpSvcTracer, "service.production_runs.bulk_create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BulkCreateProductionRunsResponse, error) {
			return m.coreClient.BulkCreateProductionRuns(ctx, &pb.BulkCreateProductionRunsRequest{ProductionRuns: pbRuns}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return jobep.JobFromProto(resp.GetJob()), nil
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

func productionRunFromSummaryProto(info *pb.ProductionRunSummaryInfo) apiresource.ProductionRun {
	s := apiresource.ProductionRun{
		ID:         info.Id,
		Object:     constants.ObjectTypeProductionRun,
		Number:     info.Number,
		BatchCount: info.BatchCount,
		CreatedAt:  grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt:  grpcutil.TimestampToTime(info.UpdatedAt),
	}

	// responsible_user is an expandable reference: the FK id is stashed in
	// LoadMeta so LoadAccountUsers fetches the real account user on
	// ?include=responsible_user. Never fabricate.

	s.StartedAt = grpcutil.TimestampToTimePtr(info.StartedAt)
	s.CompletedAt = grpcutil.TimestampToTimePtr(info.CompletedAt)

	return s
}

func productionRunListFromProto(ctx context.Context, resp *pb.ListProductionRunsResponse) *apiresource.List[apiresource.ProductionRun] {
	meta := resourcekit.GetLoadMeta(ctx)
	runs := make([]apiresource.ProductionRun, len(resp.ProductionRuns))
	for i, pr := range resp.ProductionRuns {
		runs[i] = productionRunFromSummaryProto(pr)
		if pr.ResponsibleUserId != "" {
			meta.Set(constants.ObjectTypeProductionRun, pr.Id, "responsible_user_id", pr.ResponsibleUserId)
		}
	}

	return apiresource.NewList(runs, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}

// ProductionRunFromProto maps a core ProductionRunInfo to the ProductionRun API resource.
// The responsible_user expandable is left nil; pair with StashProductionRunMeta so it
// resolves on ?include=responsible_user.
func ProductionRunFromProto(info *pb.ProductionRunInfo) apiresource.ProductionRun {
	d := apiresource.ProductionRun{
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

// StashProductionRunMeta stashes the responsible_user FK id in LoadMeta so
// LoadAccountUsers can resolve the expandable responsible_user on ?include=responsible_user.
func StashProductionRunMeta(meta *resourcekit.LoadMeta, info *pb.ProductionRunInfo) {
	if info == nil || info.ResponsibleUserId == "" {
		return
	}
	// responsible_user is an expandable reference: stash the FK id so
	// LoadAccountUsers fetches the real account user on
	// ?include=responsible_user. Never fabricate.
	meta.Set(constants.ObjectTypeProductionRun, info.Id, "responsible_user_id", info.ResponsibleUserId)
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
